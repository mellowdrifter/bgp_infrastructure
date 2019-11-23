//watch will speak BGP and output adjustments to the BGP table, maybe output via streaming gRPC?

package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"sync"
)

const (
	peer1            = "10.20.30.43"
	port      uint8  = 179
	localasn  uint32 = 64500
	remoteans uint32 = 64501
)

type bgpWatchServer struct {
	peers []*peer
	asn   uint32
	mutex *sync.RWMutex
}

type peer struct {
	buf     *bufio.ReadWriter
	lasn    uint32
	rasn    uint32
	address string
	conn    net.Conn
}

func main() {

	var peers []*peer

	newPeer := &peer{
		rasn:    remoteans,
		address: peer1,
	}

	peers = append(peers, newPeer)
	peers = append(peers, &peer{
		rasn:    64501,
		lasn:    localasn,
		address: "10.20.30.44",
	})

	fmt.Printf("I have %d peers configured\n", len(peers))

	var s bgpWatchServer
	s.peers = peers
	s.asn = localasn

	s.start()

	for {
	}

}

func (s *bgpWatchServer) start() {
	for _, peer := range s.peers {
		go peer.handle()
	}

}

// how do I get back here??
func (s *bgpWatchServer) remove(p *peer) {
	log.Printf("Removing client %s\n", p.address)

	// remove the connection from client array
	for i, check := range s.peers {
		if check == p {
			s.peers = append(s.peers[:i], s.peers[i+1:]...)
		}
	}
	err := p.conn.Close()
	if err != nil {
		log.Printf("*** Error closing connection! %v\n", err)
	}

}

// open a BGP connection
func (p *peer) open() error {
	log.Printf("Dialling %s\n", p.address)
	conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", p.address, port))
	if err != nil {
		return fmt.Errorf("Error dialling: %v", err)
	}

	p.buf = bufio.NewReadWriter(bufio.NewReader(conn), bufio.NewWriter(conn))
	p.conn = conn
	return nil
}

// most of the logic will eventually be here
func (p *peer) handle() {
	if err := p.open(); err != nil {
		log.Println(err)
		return
	}
	for {
	}
}
