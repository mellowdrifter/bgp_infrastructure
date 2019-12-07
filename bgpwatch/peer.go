package main

import (
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

type peer struct {
	twoASN        uint16
	hold          uint16
	ip            string
	conn          net.Conn
	mutex         sync.RWMutex
	param         parameters
	keepalives    uint64
	lastKeepalive time.Time
	updates       uint64
	withdraws     uint64
	startTime     time.Time
	in            *bytes.Reader
	out           *bytes.Buffer
	prefixes      []*prefixAttributes
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
		// Create a reader from that byte slice and insert into the peer struct
		p.in = bytes.NewReader(msg)

		// marker should be checked and removed via the getMessage function
		var m marker
		binary.Read(p.in, binary.BigEndian, &m)

		// Grab the header
		header, err := p.getHeader()
		if err != nil {
			log.Printf("Unable to decode header: %v\n", err)
			return
		}

		// Do some work
		switch header.Type {
		case open:
			p.HandleOpen()
			p.createOpen()
			// Following shoudl go outside of the switch statement once the rest are done
			p.encodeOutgoing()

		case keepalive:
			p.HandleKeepalive()
			p.createKeepAlive()
			p.encodeOutgoing()

		//case notification:
		//	p.handleNotification()
		//	return

		case update:
			p.handleUpdate()

			// output and dump that update
			p.currentPrefixes()

		default:
			io.CopyN(ioutil.Discard, p.in, int64(header.Length))
		}
	}
}

func (p *peer) getHeader() (header, error) {
	var h header
	binary.Read(p.in, binary.BigEndian, &h)

	return h, nil
}

func (p *peer) HandleKeepalive() {
	p.mutex.Lock()
	p.keepalives++
	p.lastKeepalive = time.Now()
	p.mutex.Unlock()
	log.Printf("received keepalive #%d\n", p.keepalives)
}

func (p *peer) HandleOpen() {
	// TODO: I only support 4 byte ASN. If peer does not support. Send a notify and tear down
	// the session.
	defer p.mutex.Unlock()
	fmt.Println("Received Open Message")
	var o msgOpen
	binary.Read(p.in, binary.BigEndian, &o)

	// Read parameters into new buffer
	pbuffer := make([]byte, int(o.ParamLen))
	// errors
	io.ReadFull(p.in, pbuffer)

	// Grab the ASN and Hold Time.
	p.twoASN = o.ASN
	p.hold = o.HoldTime

	p.mutex.Lock()
	p.param = decodeOptionalParameters(&pbuffer)

}

func (p *peer) handleNotification() {
	fmt.Println("Received a notify message")
	// Handle this better. Minimum is code and subcode. Could be more
	var n msgNotification
	binary.Read(p.in, binary.BigEndian, &n)

	fmt.Printf("Notification code is %d with subcode %d\n", n.Code, n.Subcode)
	fmt.Println("Closing session")
	p.conn.Close()

}

// Lots of work needed here
func (p *peer) handleUpdate() {
	var u msgUpdate
	binary.Read(p.in, binary.BigEndian, &u)
	// More reading. Seems different address familes are different.
	if u.WithdrawLength == 0 && u.AttrLength.toUint16() == 0 {
		log.Printf("Received End-Of-RIB marker")
		return
	}

	log.Println("received update message")
	log.Printf("Update attribute is %d bytes long\n", u.AttrLength.toUint16())

	if u.WithdrawLength == 0 {
		log.Printf("No routes withdrawn with this update")
	}

	if u.AttrLength.toUint16() == 0 {
		log.Println("There are no new prefixes being advertised")
		return
	}

	var pa prefixAttributes

	// drain the the attributes into a new buffer
	abuf := make([]byte, u.AttrLength.toUint16())
	io.ReadFull(p.in, abuf)

	// decode attributes
	pa.attr = decodePathAttributes(abuf)

	// IPv6 updates are done via attributes. Only pass the remainder of the buffer to decodeIPv4NLRI if
	// there are no IPv6 updates in the attributes.
	if len(pa.attr.ipv6NLRI) == 0 {
		// dump the rest of the update message into a buffer to use for NLRI
		// It is possible to work this out as well... needed for a copy.
		// for now just read the last of the in buffer :(
		pa.v4prefixes = decodeIPv4NLRI(p.in)
		// TODO: What about withdraws???
	} else {
		pa.v6prefixes = pa.attr.ipv6NLRI
		pa.v6NextHops = pa.attr.nextHops
	}

	p.mutex.Lock()
	p.prefixes = append(p.prefixes, &pa)
	p.mutex.Unlock()

}

func (p *peer) currentPrefixes() {
	p.mutex.RLock()
	for _, v := range p.prefixes {
		fmt.Printf("*** Current prefixes: %+v\n", v)
		fmt.Println("*** Have the following attributes:")
		fmt.Printf("Origin is %d\n", v.attr.origin)
		fmt.Printf("ASPATH is %v\n", v.attr.aspath)
		if v.attr.atomic {
			fmt.Printf("Has the atomic aggregates set")
		}
		if v.attr.agAS != 0 {
			fmt.Printf("As aggregate ASN as %v\n", v.attr.agAS)
		}
		if len(v.attr.communities) > 0 {
			fmt.Printf("Has the following communities: %v\n", v.attr.communities)
		}
		if len(v.attr.largeCommunities) > 0 {
			fmt.Printf("Has the following large communities: %v\n", v.attr.largeCommunities)
		}
	}
	// Empty the slice
	p.prefixes = p.prefixes[:0]
	p.mutex.RUnlock()
}
