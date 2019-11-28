//watch will speak BGP and output adjustments to the BGP table, maybe output via streaming gRPC?

package main

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"sync"
	"time"
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
	buf        *bufio.ReadWriter // ever going to use this?
	rasn       uint32
	ip         string
	conn       net.Conn
	mutex      sync.RWMutex
	param      parameters
	keepalives uint64
	updates    uint64
	withdraws  uint64
	startTime  time.Time
	buff       *bytes.Reader
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

	rw := bufio.NewReadWriter(bufio.NewReader(conn), bufio.NewWriter(conn))

	// Each client will have a pointer to a load of the server's data.
	peer := &peer{
		conn:      conn,
		buf:       rw,
		ip:        ip,
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

func (p *peer) peerWorker() {
	for {

		// Grab incoming BGP message and place into a reader.
		msg, err := getMessage(p.conn)
		if err != nil {
			log.Printf("Bad BGP message: %v\n", err)
			p.conn.Close()
			return

		}
		p.buff = bytes.NewReader(msg)

		// marker should be checked and removed via the getMessage function
		var m marker
		binary.Read(p.buff, binary.BigEndian, &m)

		// Grab the header
		header, err := p.getHeader()
		if err != nil {
			log.Printf("Unable to decode header: %v\n", err)
		}

		// Do some work
		switch header.Type {
		case open:
			p.HandleOpen()
			// Not sure if all the below is correct...
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

		case keepalive:
			p.HandleKeepalive()

		case notification:
			p.handleNotification()
			return

		case update:
			p.handleUpdate()
		}
	}
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

func (p *peer) getHeader() (header, error) {
	var h header
	binary.Read(p.buff, binary.BigEndian, &h)

	return h, nil
}

func (p *peer) HandleKeepalive() {
	p.keepalives++
	log.Printf("received keepalive #%d\n", p.keepalives)
	sendKeepAlive(p.conn)
	log.Println("sent keepalive")
}

func (p *peer) HandleOpen() {
	fmt.Println("Received Open Message")
	var o msgOpen
	binary.Read(p.buff, binary.BigEndian, &o)
	log.Printf("Version: %v\n", o.Version)
	log.Printf("ASN: %v\n", o.ASN)
	log.Printf("Holdtime: %v\n", o.HoldTime)
	log.Printf("ID: %v\n", fourByteString(o.BGPID))

	// Read parameters into buffer
	pbuffer := make([]byte, int(o.ParamLen))
	// errors
	io.ReadFull(p.buff, pbuffer)

	// Read in parameters
	log.Println("Decoding parameters...")
	p.param = decodeOptionalParameters(pbuffer)

}

func (p *peer) handleNotification() {
	fmt.Println("Received a notify message")
	// Handle this better. Minimum is code and subcode. Could be more
	var n msgNotification
	binary.Read(p.buff, binary.BigEndian, &n)

	fmt.Printf("Notification code is %d with subcode %d\n", n.Code, n.Subcode)
	fmt.Println("Closing session")
	p.conn.Close()

}

func (p *peer) handleUpdate() {
	var u msgUpdate
	binary.Read(p.buff, binary.BigEndian, &u)
	// More reading. Seems different address familes are different.
	if u.WithdrawLength == 0 && u.AttrLength.toUint16() == 0 {
		log.Printf("Received End-Of-RIB marker")
		return
	}

	log.Println("received update message")
	log.Printf("Update attribute is %d bytes long\n", u.AttrLength.toUint16())

	log.Println("*** DECODING UPDATES ***")
	if u.WithdrawLength == 0 {
		log.Printf("No routes withdrawn with this update")
	}
	// drain the attributes for now
	io.CopyN(ioutil.Discard, p.buff, u.AttrLength.toInt64())

	for {
		// data can be padded. If less than three it could never be an IP address
		if p.buff.Len() <= 3 {
			return
		}
		/*
			This variable length field contains a list of IP address
			prefixes.  The length, in octets, of the Network Layer
			Reachability Information is not encoded explicitly, but can be
			calculated as:

			 UPDATE message Length - 23 - Total Path Attributes Length
			 - Withdrawn Routes Length
		*/

		// This should be in a struct
		var mask uint8
		var address ipv4Address
		binary.Read(p.buff, binary.BigEndian, &mask)
		binary.Read(p.buff, binary.BigEndian, &address)

		fmt.Printf("Received a new prefix: %s\n", fmt.Sprintf("%s/%d", fourByteString(address), mask))

	}

}
