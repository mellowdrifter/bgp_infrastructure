//watch will speak BGP and output adjustments to the BGP table, maybe output via streaming gRPC?

package main

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
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

var (
	rid = [4]byte{0, 0, 0, 1}
)

type bgpWatchServer struct {
	listener net.Listener
	peers    []*peer
	mutex    sync.RWMutex
}

type peer struct {
	buf   *bufio.ReadWriter
	rasn  uint32
	addr  string
	conn  net.Conn
	mutex *sync.RWMutex
	fsm   uint8
	param parameters
}

func main() {

	serv := bgpWatchServer{
		mutex: sync.RWMutex{},
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

	rw := bufio.NewReadWriter(bufio.NewReader(conn), bufio.NewWriter(conn))

	// Each client will have a pointer to a load of the server's data.
	peer := &peer{
		conn:  conn,
		buf:   rw,
		addr:  ip,
		mutex: &s.mutex,
	}

	s.peers = append(s.peers, peer)

	log.Printf("New peer added to list: %+v\n", s.peers)

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
	for {
		msg := getMessage(p.conn)
		b := bytes.NewReader(msg)

		// BGP always has a marker of 128s bits worth of 1s
		var m marker
		binary.Read(b, binary.BigEndian, &m)

		// What is the incoming packet?
		var h header
		binary.Read(b, binary.BigEndian, &h)

		if h.Type == open {
			fmt.Println("Received Open Message")
			var o msgOpen
			binary.Read(b, binary.BigEndian, &o)
			log.Printf("Version: %v\n", o.Version)
			log.Printf("ASN: %v\n", o.ASN)
			log.Printf("Holdtime: %v\n", o.HoldTime)
			log.Printf("ID: %v\n", o.BGPID.string())
			log.Printf("Optional Parameters length: %d\n", int(o.ParamLen))

			// Read parameters into buffer
			pbuffer := make([]byte, int(o.ParamLen))
			// errors
			io.ReadFull(b, pbuffer)

			// Decode and assign
			p.param = decodeOptionalParameters(pbuffer)

			op := msgOpen{}
			op.serialize(p.conn)
			// are there any unsupported parameters?
			if len(p.param.unsupported) > 0 {
				log.Println("Sending a notify")
				notify := msgNotification{
					Code:    2,
					Subcode: 7,
				}
				notify.unsupported(p.conn, p.param.unsupported)
			}

		}
		if h.Type == keepalive {
			log.Println("received keepalive")
			sendKeepAlive(p.conn)
			log.Println("sent keepalive")
		}

		if h.Type == update {
			log.Println("received update message")
			var u msgUpdate
			binary.Read(b, binary.BigEndian, &u)
			// More reading. Seems different address familes are different.
			if u.Length == 0 && u.Attr == 0 {
				log.Printf("Received End-Of-RIB marker")
			}

		}
	}
}

// If BGP session sends notification, we die here
func getMessage(c net.Conn) []byte {
	// 4096 is the max BGP packet size
	buffer := make([]byte, 4096)
	// 19 is the minimum size (keepalive)
	io.ReadFull(c, buffer[:19])
	log.Println("Read the first 19 bytes")

	// These calculations should go into a function
	length := int(buffer[16])*256 + int(buffer[17])
	log.Printf("The total packet length will be %d\n", length)

	// Check for errors on these readfulls
	io.ReadFull(c, buffer[19:length])

	// Only the bytes of interest should be returned
	return buffer[:length]

}
