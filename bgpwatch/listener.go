//watch will speak BGP and output adjustments to the BGP table, maybe output via streaming gRPC?

package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net"
	"sync"
	"time"
)

const (
	port uint8 = 179
)

var (
	rid = bgpid{0, 0, 0, 1}
)

type bgpWatchServer struct {
	listener net.Listener
	peers    []*peer
	mutex    sync.RWMutex
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

// start will start the listener as well as start each peer worker.
func (s *bgpWatchServer) start() {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			log.Printf("%v\n", err)
		} else {
			peer := s.accept(conn)
			go peer.peerWorker()
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

	// If new client trying to connect with existing connection, remove old peer from pool
	for _, peer := range s.peers {
		if ip == peer.ip {
			s.remove(peer)
			break
		}
	}

	// Each client will have a buffer with the maximum BGP message size. This is only
	// 4k per client, so it's not a big deal.
	peer := &peer{
		conn:      conn,
		ip:        ip,
		out:       bytes.NewBuffer(make([]byte, 4096)),
		mutex:     sync.RWMutex{},
		startTime: time.Now(),
	}

	s.peers = append(s.peers, peer)

	log.Printf("New peer added to list: %+v\n", s.peers)

	return peer
}

// remove removes a client from the current list of clients being served.
func (s *bgpWatchServer) remove(p *peer) {
	log.Printf("Removing dead peer %s\n", p.conn.RemoteAddr().String())

	// remove the connection from client array
	for i, check := range s.peers {
		if check == p {
			s.peers = append(s.peers[:i], s.peers[i+1:]...)
		}
	}
	// Don't worry about errors as it's mostly because it's already closed.
	p.conn.Close()

}

// Make this better somehow
func getMessage(c net.Conn) ([]byte, error) {
	// 4096 is the max BGP packet size
	buffer := make([]byte, 4096)
	// 19 is the minimum size (keepalive)
	_, err := io.ReadFull(c, buffer[:19])
	if err != nil {
		return nil, err
	}
	log.Println("Read the first 19 bytes")

	// Add a check to ensure the BGP marker is present, else return an error
	// In fact we could just remove the marker if all good as I don't really care about it!

	// These calculations should go into a function
	length := int(buffer[16])*256 + int(buffer[17])
	log.Printf("The total packet length will be %d\n", length)

	// Check for errors on these readfulls
	_, err = io.ReadFull(c, buffer[19:length])
	if err != nil {
		return nil, err
	}

	// Only the bytes of interest should be returned
	return buffer[:length], nil
}
