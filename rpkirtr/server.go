// This app implements RFC8210.
// The Resource Public Key Infrastructure (RPKI) to Router Protocol,
// Version 1

package main

import (
	"fmt"
	"log"
	"math/rand"
	"net"
	"os"
	"sync"
	"time"

	_ "net/http/pprof"

	"github.com/pkg/profile"
)

const (
	cacheurl = "https://rpki.cloudflare.com/rpki.json"
	logfile  = "/var/log/rpkirtr.log"

	// Each region will just be an enum.
	afrinic rir = 0
	apnic   rir = 1
	arin    rir = 2
	lacnic  rir = 3
	ripe    rir = 4

	// refreshROA is the amount of seconds to wait until a new json is pulled.
	refreshROA = 15 * time.Minute

	// 8282 is the RFC port for RPKI-RTR
	port = 8282
	loc  = "localhost"

	// Intervals are the default intervals in seconds if no specific value is configured
	refresh = uint32(3600) // 1 - 86400
	retry   = uint32(600)  // 1 - 7200
	expire  = uint32(7200) // 600 - 172800

	//maxMinMask is the largest min mask wanted
	maxMinMaskv4 = 24
	maxMinMaskv6 = 48
)

// enum used for RIRs
type rir uint8

// jsonroa is a struct to push the cloudflare ROA data into.
type jsonroa struct {
	Prefix string  `json:"prefix"`
	Mask   float64 `json:"maxLength"`
	ASN    string  `json:"asn"`
	RIR    string  `json:"ta"`
}

// Converted ROA struct with all the details.
// Handy calculator - http://golang-sizeof.tips/ - current size 24
type roa struct {
	Prefix  string
	MinMask uint8
	MaxMask uint8
	ASN     uint32
	RIR     rir
	IsV4    bool
}

// rpkiResponse, metadata, and roas are all used to unmarshal the json file.
type rpkiResponse struct {
	metadata `json:"metadata"`
	roas
}
type metadata struct {
	Generated float64 `json:"generated"`
	Valid     float64 `json:"valid"`
}

type roas struct {
	Roas []jsonroa `json:"roas"`
}

// CacheServer is our RPKI server.
type CacheServer struct {
	listener net.Listener
	clients  []*client
	roas     []roa
	mutex    *sync.RWMutex
	serial   uint32
	session  uint16
	diff     serialDiff
}

// serialDiff will have a list of add and deletes of ROAs to get from
// oldSerial to newSerial.
type serialDiff struct {
	oldSerial uint32
	newSerial uint32
	delRoa    []roa
	addRoa    []roa
	// There may be no actual diffs between now and last
	diff bool
}

func main() {
	defer profile.Start(profile.MemProfile).Stop()

	// set up log file
	f, err := os.OpenFile(logfile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal(fmt.Errorf("failed to open logfile: %w", err))
	}
	defer f.Close()
	log.SetOutput(f)

	// random seed used for session ID
	rand.Seed(time.Now().UTC().UnixNano())

	// We need our initial set of ROAs.
	log.Printf("Downloading %s\n", cacheurl)
	roas, err := readROAs(cacheurl)
	if err != nil {
		log.Fatalf("Unable to download ROAs, aborting: %v", err)
	}

	// https://golangcode.com/print-the-current-memory-usage/
	go PrintMemUsage()

	// Set up our server with it's initial data.
	rpki := CacheServer{
		mutex:   &sync.RWMutex{},
		session: uint16(rand.Intn(65535)),
		roas:    roas,
	}
	rpki.mutex = &sync.RWMutex{}
	rpki.listen()

	// ROAs should be updated at every refresh interval
	go rpki.updateROAs(cacheurl)

	// Show me the status of the server. Mostly for debugging
	go rpki.status()

	// I'm listening!
	defer rpki.close()
	rpki.start()

}

// Start listening
func (s *CacheServer) listen() {
	l, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		log.Fatalf("Unable to start server: %v", err)
	}
	s.listener = l
	log.Printf("Listening on port %d\n", port)

}

func (s *CacheServer) status() {
	for {
		s.mutex.RLock()
		log.Printf("I currently have %d clients connected\n", len(s.clients))
		log.Printf("Current ROA count is %d\n", len(s.roas))
		log.Printf("Current serial number is %d\n", s.serial)
		log.Printf("Last diff is %t\n", s.diff.diff)
		log.Printf("Current size of diff is %d\n", len(s.diff.addRoa)+len(s.diff.delRoa))
		s.mutex.RUnlock()
		time.Sleep(5 * time.Minute)
	}

}

// close off the listener if existing
func (s *CacheServer) close() {
	s.listener.Close()
}

// start will start the listener as well as accept client and handle each.
func (s *CacheServer) start() {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			log.Printf("%v\n", err)
		} else {
			client := s.accept(conn)
			go client.handleClient()
		}
	}
}

// accept adds a new client to the current list of clients being served.
func (s *CacheServer) accept(conn net.Conn) *client {
	log.Printf("Connection from %v, total clients: %d\n",
		conn.RemoteAddr().String(), len(s.clients)+1)

	s.mutex.Lock()
	defer s.mutex.Unlock()

	// If existing client, close the old connection if it's still persistant.
	for _, client := range s.clients {
		ip, _, _ := net.SplitHostPort(conn.RemoteAddr().String())
		if client.addr == ip {
			log.Printf("Already have a connection from %s, so attemping to close existing one\n", client.addr)
			s.remove(client)
			break
		}
	}

	ip, _, _ := net.SplitHostPort(conn.RemoteAddr().String())

	// Each client will have a pointer to a load of the server's data.
	client := &client{
		conn:   conn,
		addr:   ip,
		roas:   &s.roas,
		serial: &s.serial,
		mutex:  s.mutex,
		diff:   &s.diff,
	}

	s.clients = append(s.clients, client)

	log.Printf("New client added to list: %#v\n", s.clients)

	return client
}

// remove removes a client from the current list of clients being served.
func (s *CacheServer) remove(c *client) {
	log.Printf("Removing client %s\n", c.conn.RemoteAddr().String())

	// remove the connection from client array
	for i, check := range s.clients {
		if check == c {
			s.clients = append(s.clients[:i], s.clients[i+1:]...)
		}
	}
	err := c.conn.Close()
	if err != nil {
		log.Printf("*** Error closing connection! %v\n", err)
	}

}

// updateROAs will update the server struct with the current list of ROAs
func (s *CacheServer) updateROAs(f string) {
	for {
		time.Sleep(refreshROA)
		s.mutex.Lock()
		roas, err := readROAs(f)
		if err != nil {
			log.Printf("Unable to update ROAs, so keeping existing ROAs for now: %v\n", err)
			s.mutex.Unlock()
			return
			// TODO: What happens if I'm unable to update ROAs? The diff struct could get old.
			// Check the client diff update to ensure it's doing the right thing
		}

		// Calculate diffs
		s.diff = makeDiff(roas, s.roas, s.serial)

		// Increment serial and replace
		s.serial++
		s.roas = roas
		log.Printf("roas updated, serial is now %d\n", s.serial)

		s.mutex.Unlock()
		// Notify all clients that the serial number has been updated.
		for _, c := range s.clients {
			log.Printf("sending a notify to %s\n", c.addr)
			c.notify(s.serial, s.session)
		}
	}
}
