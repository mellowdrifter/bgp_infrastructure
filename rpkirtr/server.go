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
)

const (
	cacheurl = "https://rpki.cloudflare.com/rpki.json"
	//cache    = "data/rpki.json"
	logfile = "/var/log/rpkirtr.log"

	// Each region will just be an enum.
	afrinic rir = 0
	apnic   rir = 1
	arin    rir = 2
	lacnic  rir = 3
	ripe    rir = 4

	// refresh is the amount of seconds to wait until a new json is pulled.
	// refresh = 4 * time.Minute
	refresh = 60 * time.Second

	// 8282 is the RFC port for RPKI-RTR
	port = 8282
	loc  = "localhost"
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
	MinMask int
	MaxMask int
	ASN     int
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
}

func main() {

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
	fmt.Printf("Listening on port %d\n", port)

}

func (s *CacheServer) printClients() {
	for {
		s.mutex.RLock()
		fmt.Printf("I currently have %d clients connected\n", len(s.clients))
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
			fmt.Printf("%v\n", err)
		} else {
			client := s.accept(conn)
			go s.serve(client)
		}
	}
}

// Mux out the incoming connections. Should probably be called handleClient
func (s *CacheServer) serve(client *client) {
	fmt.Printf("Serving %s\n", client.conn.RemoteAddr().String())

	for {
		// Handle incoming PDU
		var header headerPDU
		binary.Read(client.conn, binary.BigEndian, &header)

		switch {
		// I only support version 1 for now.
		case header.Version != 1:
			client.error(4, "Unsupported Protocol Version")
			client.conn.Close()
			return

		case header.Ptype == resetQuery:
			var r resetQueryPDU
			binary.Read(client.conn, binary.BigEndian, &r)
			fmt.Printf("received a reset Query PDU: %+v\n", r)
			client.sendRoa()

		case header.Ptype == serialQuery:
			var q serialQueryPDU
			binary.Read(client.conn, binary.BigEndian, &q)
			fmt.Printf("received a serial query PDU, so going to send a reset: %+v\n", q)
			// For now send a cache reset
			// TODO: adjust this once I have diffs working
			client.sendReset()
		}
	}
}

// accept adds a new client to the current list of clients being served.
func (s *CacheServer) accept(conn net.Conn) *client {
	fmt.Printf("Connection from %v, total clients: %d\n",
		conn.RemoteAddr().String(), len(s.clients)+1)

	s.mutex.Lock()
	defer s.mutex.Unlock()

	// If existing client, close the old connection.
	for _, client := range s.clients {
		ip, _, _ := net.SplitHostPort(conn.RemoteAddr().String())
		if client.addr == ip {
			fmt.Printf("Already have a connection from %s, so closing existing one\n", client.addr)
			s.remove(client)
			break
		}
	}
	fmt.Println("### End of close loop")

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
	fmt.Printf("Removing client %s\n", c.conn.RemoteAddr().String())

	// remove the connection from client array
	for i, check := range s.clients {
		if check == c {
			s.clients = append(s.clients[:i], s.clients[i+1:]...)
		}
	}
	fmt.Println("End of check loop")
	err := c.conn.Close()
	if err != nil {
		fmt.Printf("*** Error closing connection! %v\n", err)
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
		fmt.Printf("roas updated, serial is now %d\n", s.serial)
		s.mutex.Unlock()
	}

}

// makeDiff will return a list of ROAs that need to be deleted or updated
// in order for a particular serial version to updated to the latest version.
func makeDiff(new []roa, old []roa, serial uint32) serialDiff {
	newMap := make(map[string]roa, len(new))
	oldMap := make(map[string]roa, len(old))
	var addROA, delROA []roa

	for _, roa := range new {
		newMap[fmt.Sprintf("%s%d%d%d", roa.Prefix, roa.MinMask, roa.MaxMask, roa.ASN)] = roa
	}
	for _, roa := range old {
		oldMap[fmt.Sprintf("%s%d%d%d", roa.Prefix, roa.MinMask, roa.MaxMask, roa.ASN)] = roa
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
		fmt.Println("No addROA diff this time")
	}
	if len(delROA) == 0 {
		fmt.Println("No delROA diff this time")
	}
	if len(addROA) > 0 {
		fmt.Printf("New ROAs to be added: %+v\n", addROA)
	}
	if len(delROA) > 0 {
		fmt.Printf("Old ROAs to be deleted: %+v\n", delROA)
	}

	return serialDiff{
		oldSerial: serial,
		newSerial: serial + 1,
		addRoa:    addROA,
		delRoa:    delROA,
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
			MinMask: stringToInt(prefix[2]),
			MaxMask: int(r.Mask),
			ASN:     asnToInt(r.ASN),
			RIR:     rirs[r.RIR],
		})

	}

	return roas, nil

}
