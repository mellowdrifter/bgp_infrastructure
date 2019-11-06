package main

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net"
	"net/http"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	_ "net/http/pprof"
)

const (
	//cacheurl = "https://rpki.cloudflare.com/rpki.json"
	cache   = "data/rpki.json"
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
	mutex    *sync.Mutex
	serial   uint32
}

// Each client has their own stuff
type client struct {
	conn net.Conn
}

func main() {

	// set up log file
	f, err := os.OpenFile(logfile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal(fmt.Errorf("failed to open logfile: %w", err))
	}
	defer f.Close()
	log.SetOutput(f)

	go func() {
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()

	dir, err := os.Getwd()
	if err != nil {
		log.Fatal(fmt.Errorf("unable to get current working directory: %w", err))
	}

	roaFile := path.Join(dir, cache)
	log.Printf("Downloading %s\n", roaFile)

	// This is our server itself
	var thing CacheServer
	thing.mutex = &sync.Mutex{}
	thing.listen()

	// ROAs should be updated all the time
	go thing.readROAs(roaFile)

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
		s.mutex.Lock()
		fmt.Printf("I currently have %d clients connected\n", len(s.clients))
		if len(s.clients) > 0 {
			for i, client := range s.clients {
				fmt.Printf("Client #%d: Address: %s\n", i, client.conn.RemoteAddr().String())
			}
		}
		s.mutex.Unlock()
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

func (s *CacheServer) serve(client *client) {
	defer s.mutex.Unlock()
	s.mutex.Lock()
	fmt.Printf("Serving %s\n", client.conn.RemoteAddr().String())
	session := rand.Intn(100)

	// TODO: This is crap
	var whatPDU unknownPDU
	var r resetQueryPDU
	var q serialQueryPDU
	binary.Read(client.conn, binary.BigEndian, &whatPDU)
	if whatPDU.Ptype == resetQuery {
		binary.Read(client.conn, binary.BigEndian, &r)
	} else if whatPDU.Ptype == serialQuery {
		binary.Read(client.conn, binary.BigEndian, &q)
	}

	fmt.Printf("whatPDU = %+v\n", whatPDU)
	fmt.Printf("resetPDU = %+v\n", r)
	fmt.Printf("serialPDU = %+v\n", q)

	cpdu := cacheResponsePDU{
		sessionID: uint16(session),
	}
	cpdu.serialize(client.conn)

	for _, roa := range s.roas {
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
			ppdu.serialize(client.conn)
		case false:
			ppdu := ipv4PrefixPDU{
				flags:  uint8(1),
				min:    uint8(roa.MinMask),
				max:    uint8(roa.MaxMask),
				prefix: IPAddress.To4(),
				asn:    uint32(roa.ASN),
			}
			ppdu.serialize(client.conn)
		}
	}
	fmt.Println("Finished sending all prefixes")
	epdu := endOfDataPDU{
		sessionID: uint16(session),
		//serial:    cacheSerial,
		refresh: uint32(900),
		retry:   uint32(30),
		expire:  uint32(171999),
	}
	epdu.serialize(client.conn)

}

func (s *CacheServer) accept(conn net.Conn) *client {
	fmt.Printf("Connection from %v, total clients: %d\n",
		conn.RemoteAddr().String(), len(s.clients)+1)

	s.mutex.Lock()
	defer s.mutex.Unlock()

	client := &client{
		conn: conn,
	}

	s.clients = append(s.clients, client)

	return client
}

// readROAs will update the server struct with the current list of ROAs
func (s *CacheServer) readROAs(f string) {
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
func readROAs(file string) ([]roa, error) {

	f, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, fmt.Errorf("unable to read file: %w", err)
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
