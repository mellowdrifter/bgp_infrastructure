package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"
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
	//refresh = 4 * time.Hour
	refresh = 10 * time.Second

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

// Server is our main thing
type Server struct {
	roas []roa
	net  net.Conn
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
	var thing Server

	// Keep ROAs updated
	go thing.readROAs(roaFile)

	// Actually do something
	add := fmt.Sprintf("%s:%d", loc, port)
	l, err := net.Listen("tcp", add)
	if err != nil {
		fmt.Printf("Unable to start server: %w", err)
		os.Exit(1)
	}
	defer l.Close()
	for {
		thing.net, err = l.Accept()
		if err != nil {
			fmt.Println(err)
			return
		}
		thing.handleQuery()
	}

}

// readROAs will update the server struct with the current list of ROAs
func (s *Server) readROAs(f string) {
	for {
		roas, err := readROAs(f)
		if err != nil {
			log.Fatal(err)
		}
		s.roas = roas
		fmt.Println("roas updated")
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
	}

	return n
}

// The Cloudflare JSON prepends AS to all numbers. Need to remove it here.
func asnToInt(a string) int {
	n, err := strconv.Atoi(a[2:])
	if err != nil {
		log.Printf("Unable to convert ASN %s to int", a)
	}

	return n
}

func (s *Server) handleQuery() {
	fmt.Printf("Servering %s\n", s.net.RemoteAddr().String())
	for {
		netData, err := bufio.NewReader(s.net).ReadString('\n')
		if err != nil {
			fmt.Println(err)
			return
		}
		temp := strings.TrimSpace(string(netData))
		if temp == "stop" {
			break
		}
		fmt.Println(temp)
		i, err := strconv.Atoi(temp)
		if err != nil {
			break
		}
		for _, roa := range s.roas {
			if roa.ASN == i {
				fmt.Println(roa)
			}
		}
	}
}
