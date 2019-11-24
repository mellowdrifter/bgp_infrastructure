//watch will speak BGP and output adjustments to the BGP table, maybe output via streaming gRPC?

package main

import (
	"bufio"
	"encoding/binary"
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
	rid              = "0.0.0.1"
)

type bgpWatchServer struct {
	listener net.Listener
	peers    []*peer
	mutex    *sync.RWMutex
}

type peer struct {
	buf   *bufio.ReadWriter
	rasn  uint32
	addr  string
	conn  net.Conn
	mutex *sync.RWMutex
	fsm   uint8
}

func main() {

	serv := bgpWatchServer{
		mutex: &sync.RWMutex{},
	}
	serv.listen()

	serv.start()

}

// Start listening
func (s *bgpWatchServer) listen() {
	l, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		log.Fatalf("Unable to start server: %v", err)
	}
	s.listener = l
	log.Printf("Listening on port %d\n", port)

}

// start will start the listener as well as accept client and handle each.
func (s *bgpWatchServer) start() {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			log.Printf("%v\n", err)
		} else {
			peer := s.accept(conn)
			go peer.handle()
		}
	}
}

// accept adds a new client to the current list of clients being served.
func (s *bgpWatchServer) accept(conn net.Conn) *peer {
	log.Printf("Connection from %v, total peers: %d\n",
		conn.RemoteAddr().String(), len(s.peers)+1)

	s.mutex.Lock()
	defer s.mutex.Unlock()

	ip, _, _ := net.SplitHostPort(conn.RemoteAddr().String())

	// Each client will have a pointer to a load of the server's data.
	peer := &peer{
		conn:  conn,
		addr:  ip,
		mutex: s.mutex,
	}

	s.peers = append(s.peers, peer)

	log.Printf("New peer added to list: %#v\n", s.peers)

	return peer
}

// how do I get back here??
func (s *bgpWatchServer) remove(p *peer) {
	log.Printf("Removing client %s\n", p.addr)

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

// most of the logic will eventually be here
func (p *peer) handle() {
	// BGP always has a marker of 128s bits worth of 1s
	var m marker
	binary.Read(p.conn, binary.BigEndian, &m)

	// What is the incoming packet?
	var h header
	binary.Read(p.conn, binary.BigEndian, &h)

	// debug logging for now
	log.Printf("Received: %#v\n", m)
	log.Printf("Received: %#v\n", h)
	fmt.Printf("Length is %d\n", h.Length)

	if h.Type == 1 {
		fmt.Println("Received Open Message")
		var o openHeader
		binary.Read(p.conn, binary.BigEndian, &o)
		log.Printf("Received: %#v\n", o)
		log.Printf("Version: %v\n", o.Version)
		log.Printf("ASN: %v\n", o.ASN)
		log.Printf("Holdtime: %v\n", o.HoldTime)
		log.Printf("ID: %v\n", o.BGPID.string())
		log.Printf("ParamLength: %v\n", o.ParamLen)
		keepalive := header{}
		keepalive.serialize(p.conn)
	}
	for {
	}
}
