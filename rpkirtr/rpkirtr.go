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
	"strconv"
	"strings"
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
}

// Each client has their own stuff
type client struct {
	conn    net.Conn
	session *uint16
	addr    string
	roas    *[]roa
	serial  *uint32
	mutex   *sync.RWMutex
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

	log.Printf("Downloading %s\n", cacheurl)

	// This is our server itself
	thing := CacheServer{
		mutex:   &sync.RWMutex{},
		session: uint16(rand.Intn(65535)),
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
		s.mutex.Lock()
		s.serial++
		roas, err := readROAs(f)
		if err != nil {
			log.Fatal(err)
		}
		s.roas = roas
		fmt.Printf("roas updated, serial is now %d\n", s.serial)
		s.mutex.Unlock()
		time.Sleep(refresh)
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

// stringToInt does inline convertions and logs errors, instead of panicing.
func stringToInt(s string) int {
	n, err := strconv.Atoi(s)
	if err != nil {
		log.Printf("Unable to convert %s to int", s)
		return 0
	}

	return n
}

// The Cloudflare JSON prepends AS to all numbers. Need to remove it here.
func asnToInt(a string) int {
	n, err := strconv.Atoi(a[2:])
	if err != nil {
		log.Printf("Unable to convert ASN %s to int", a)
		return 0
	}

	return n
}

// reset has no data besides the header
func (c *client) sendReset() {
	rpdu := cacheResetPDU{}
	rpdu.serialize(c.conn)
}

func (c *client) sendRoa() {
	session := rand.Intn(100)
	cpdu := cacheResponsePDU{
		sessionID: uint16(session),
	}
	cpdu.serialize(c.conn)

	c.mutex.RLock()
	for _, roa := range *c.roas {
		IPAddress := net.ParseIP(roa.Prefix)
		// TODO put ipv4/ipv6 signal in when creating the ROAs
		switch strings.Contains(roa.Prefix, ":") {
		case true:
			ppdu := ipv6PrefixPDU{
				flags:  uint8(1),
				min:    uint8(roa.MinMask),
				max:    uint8(roa.MaxMask),
				prefix: IPAddress.To16(),
				asn:    uint32(roa.ASN),
			}
			ppdu.serialize(c.conn)
		case false:
			ppdu := ipv4PrefixPDU{
				flags:  uint8(1),
				min:    uint8(roa.MinMask),
				max:    uint8(roa.MaxMask),
				prefix: IPAddress.To4(),
				asn:    uint32(roa.ASN),
			}
			ppdu.serialize(c.conn)
		}
	}
	c.mutex.RUnlock()
	fmt.Println("Finished sending all prefixes")
	epdu := endOfDataPDU{
		sessionID: uint16(session),
		//serial:    cacheSerial,
		refresh: uint32(900),
		retry:   uint32(30),
		expire:  uint32(171999),
	}
	epdu.serialize(c.conn)

}

// TODO: Test this somehow
func (c *client) error(code int, report string) {
	epdu := errorReportPDU{
		code:   uint16(code),
		report: report,
	}
	epdu.serialize(c.conn)

}

func (c *client) status() {
	fmt.Println("Status of client:")
	fmt.Printf("Address is %s\n", c.addr)
	c.mutex.RLock()
	fmt.Printf("Serial is %d\n", *c.serial)
	c.mutex.RUnlock()

}
