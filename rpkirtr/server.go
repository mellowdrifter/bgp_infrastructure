// This app implements RFC8210.
// The Resource Public Key Infrastructure (RPKI) to Router Protocol,
// Version 1

package main

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net"
	"os"
	"regexp"
	"sync"
	"time"

	"net/http"
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

	// refresh is the amount of seconds to wait until a new json is pulled.
	// refresh = 4 * time.Minute
	refresh = 15 * time.Minute

	// 8282 is the RFC port for RPKI-RTR
	port = 8282
	loc  = "localhost"

	//maxMinMask is the largest min mask wanted
	maxMinMaskv4 = 24
	maxMinMaskv6 = 48
)

// enum used for RIRs
type rir int

// jsonroa is a struct to push the cloudflare ROA data into.
type jsonroa struct {
	Prefix string  `json:"prefix"`
	Mask   float64 `json:"maxLength"`
	ASN    string  `json:"asn"`
	RIR    string  `json:"ta"`
}

// Converted ROA struct with all the details.
type roa struct {
	Prefix  string
	MinMask uint8
	MaxMask uint8
	ASN     uint32
	RIR     rir
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

// CacheServer is our main thing
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
	defer profile.Start().Stop()

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

	// This is our server itself
	thing := CacheServer{
		mutex:   &sync.RWMutex{},
		session: uint16(rand.Intn(65535)),
		roas:    roas,
	}
	thing.mutex = &sync.RWMutex{}
	thing.listen()

	// ROAs should be updated all the time
	go thing.updateROAs(cacheurl)

	// Show me how many clients are connected
	go thing.printClients()

	// I'm listening!
	thing.start()

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

func (s *CacheServer) printClients() {
	for {
		s.mutex.RLock()
		log.Printf("I currently have %d clients connected\n", len(s.clients))
		if len(s.clients) > 0 {
			for _, client := range s.clients {
				client.status()
			}
		}
		s.mutex.RUnlock()
		time.Sleep(time.Minute)
	}
}

func (s *CacheServer) close() {
	s.listener.Close()
}

func (s *CacheServer) start() {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			log.Printf("%v\n", err)
		} else {
			client := s.accept(conn)
			go s.serve(client)
		}
	}
}

// Mux out the incoming connections. Should probably be called handleClient
func (s *CacheServer) serve(client *client) {
	log.Printf("Serving %s\n", client.conn.RemoteAddr().String())

	for {
		// Handle incoming PDU
		var header headerPDU
		binary.Read(client.conn, binary.BigEndian, &header)

		switch {
		// I only support version 1 for now.
		case header.Version != 1:
			log.Printf("Received something I don't know :'(  %+v\n", header)
			client.error(4, "Unsupported Protocol Version")
			client.conn.Close()
			return

		case header.Ptype == resetQuery:
			var r resetQueryPDU
			binary.Read(client.conn, binary.BigEndian, &r)
			log.Printf("received a reset Query PDU from %s\n", client.addr)
			client.sendRoa()

		case header.Ptype == serialQuery:
			var q serialQueryPDU
			binary.Read(client.conn, binary.BigEndian, &q)
			// If the client sends in the current or previous serial, then we can handle it.
			// If the serial is older or unknown, we need to send a reset.
			s.mutex.RLock()
			serial := s.diff.newSerial
			s.mutex.RUnlock()
			// TODO: These serials could change while busy here
			if q.Serial != serial && q.Serial != serial-1 {
				log.Printf("received a serial query PDU, with an unmanagable serial from %s\n", client.addr)
				log.Printf("Serial received: %d. Current server serial: %d\n", q.Serial, serial)
				client.sendReset()
			}
			if q.Serial == serial {
				log.Printf("received a serial number which currently matches my own from %s\n", client.addr)
				log.Printf("Serial received: %d. Current server serial: %d\n", q.Serial, serial)
				client.sendEmpty(q.Session)
			}
			if q.Serial == serial-1 {
				log.Printf("received a serial number one less, so sending diff to %s\n", client.addr)
				log.Printf("Serial received: %d. Current server serial: %d\n", q.Serial, serial)
				s.mutex.RLock()
				client.sendDiff(s.diff, q.Session)
				s.mutex.RUnlock()
			}
		}
	}
}

// accept adds a new client to the current list of clients being served.
func (s *CacheServer) accept(conn net.Conn) *client {
	log.Printf("Connection from %v, total clients: %d\n",
		conn.RemoteAddr().String(), len(s.clients)+1)

	s.mutex.Lock()
	defer s.mutex.Unlock()

	// If existing client, close the old connection.
	for _, client := range s.clients {
		ip, _, _ := net.SplitHostPort(conn.RemoteAddr().String())
		if client.addr == ip {
			log.Printf("Already have a connection from %s, so closing existing one\n", client.addr)
			s.remove(client)
			break
		}
	}
	log.Println("### End of close loop")

	ip, _, _ := net.SplitHostPort(conn.RemoteAddr().String())
	client := &client{
		conn:   conn,
		addr:   ip,
		roas:   &s.roas,
		serial: &s.serial,
		mutex:  s.mutex,
	}

	s.clients = append(s.clients, client)

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
	log.Println("End of check loop")
	err := c.conn.Close()
	if err != nil {
		log.Printf("*** Error closing connection! %v\n", err)
	}

}

// updateROAs will update the server struct with the current list of ROAs
func (s *CacheServer) updateROAs(f string) {
	for {
		time.Sleep(refresh)
		s.mutex.Lock()
		roas, err := readROAs(f)
		if err != nil {
			log.Fatal(err)
		}

		// Calculate diffs
		diff := makeDiff(roas, s.roas, s.serial)
		s.diff = diff

		// Increment serial and replace
		s.serial++
		s.roas = roas
		log.Printf("roas updated, serial is now %d\n", s.serial)

		// TODO: Notify should only be sent if there is an actual diff
		for _, c := range s.clients {
			log.Printf("sending a notify to %s\n", c.addr)
			c.notify(s.serial, s.session)
		}
		s.mutex.Unlock()
	}
}

// makeDiff will return a list of ROAs that need to be deleted or updated
// in order for a particular serial version to updated to the latest version.
func makeDiff(new []roa, old []roa, serial uint32) serialDiff {
	newMap := make(map[string]roa, len(new))
	oldMap := make(map[string]roa, len(old))
	var addROA, delROA []roa

	// TODO: This works, but is a bit of a mess. Hash needs more than just prefix
	// as you can have multiple ROAs per prefix (which includes min mask)
	for _, roa := range new {
		newMap[fmt.Sprintf("%s%d%d", roa.Prefix, roa.MaxMask, roa.ASN)] = roa
	}
	for _, roa := range old {
		oldMap[fmt.Sprintf("%s%d%d", roa.Prefix, roa.MaxMask, roa.ASN)] = roa
	}

	// If ROA is in newMap but not oldMap, we need to add it
	for k, v := range newMap {
		_, ok := oldMap[k]
		if !ok {
			addROA = append(addROA, v)
		}
	}

	// If ROA is in oldMap but not newMap, we need to delete it.
	for k, v := range oldMap {
		_, ok := newMap[k]
		if !ok {
			delROA = append(delROA, v)
		}
	}

	if len(addROA) == 0 {
		log.Println("No addROA diff this time")
	}
	if len(delROA) == 0 {
		log.Println("No delROA diff this time")
	}
	if len(addROA) > 0 {
		log.Printf("New ROAs to be added: %+v\n", addROA)
	}
	if len(delROA) > 0 {
		log.Printf("Old ROAs to be deleted: %+v\n", delROA)
	}

	diff := (len(addROA) > 0 || len(delROA) > 0)

	return serialDiff{
		oldSerial: serial,
		newSerial: serial + 1,
		addRoa:    addROA,
		delRoa:    delROA,
		diff:      diff,
	}
}

// readROAs will read the current ROAs into memory.
func readROAs(url string) ([]roa, error) {

	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	f, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	rirs := map[string]rir{
		"Cloudflare - AFRINIC": afrinic,
		"Cloudflare - ARIN":    arin,
		"Cloudflare - APNIC":   apnic,
		"Cloudflare - LACNIC":  lacnic,
		"Cloudflare - RIPE":    ripe,
	}

	var r rpkiResponse
	json.Unmarshal(f, &r)

	// We know how many ROAs we have, so we can add that capacity directly
	roas := make([]roa, 0, len(r.roas.Roas))

	rxp := regexp.MustCompile(`(.*)/(.*)`)

	for _, r := range r.roas.Roas {
		prefix := rxp.FindStringSubmatch(r.Prefix)
		roas = append(roas, roa{
			Prefix:  prefix[1],
			MinMask: uint8(stringToInt(prefix[2])),
			MaxMask: uint8(r.Mask),
			ASN:     uint32(asnToInt(r.ASN)),
			RIR:     rirs[r.RIR],
		})

	}

	return roas, nil

}
