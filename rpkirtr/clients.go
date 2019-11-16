package main

import (
	"encoding/binary"
	"log"
	"math/rand"
	"net"
	"sync"
)

// Each client has their own stuff
type client struct {
	conn    net.Conn
	session *uint16
	addr    string
	roas    *[]roa
	serial  *uint32
	mutex   *sync.RWMutex
	diff    *serialDiff
}

// reset has no data besides the header
func (c *client) sendReset() {
	rpdu := cacheResetPDU{}
	rpdu.serialize(c.conn)
}

// sendDiff should send additions and deletions to the client.
func (c *client) sendDiff(diff *serialDiff, session uint16) {
	cpdu := cacheResponsePDU{
		sessionID: session,
	}
	cpdu.serialize(c.conn)
	if diff.diff {
		for _, roa := range diff.addRoa {
			writePrefixPDU(&roa, c.conn, announce)
		}
		for _, roa := range diff.delRoa {
			writePrefixPDU(&roa, c.conn, withdraw)
		}
		log.Println("Finished sending all diffs")
	}
	epdu := endOfDataPDU{
		sessionID: uint16(session),
		serial:    *c.serial,
		refresh:   uint32(900),
		retry:     uint32(30),
		expire:    uint32(171999),
	}
	epdu.serialize(c.conn)

}

// writePrefixPDU will directly write the update or withdraw prefix PDU.
func writePrefixPDU(r *roa, c net.Conn, flag uint8) {
	IPAddress := net.ParseIP(r.Prefix)
	switch r.IsV4 {
	case true:
		ppdu := ipv4PrefixPDU{
			flags:  flag,
			min:    r.MinMask,
			max:    r.MaxMask,
			prefix: ipv4ToByte(IPAddress.To4()),
			asn:    r.ASN,
		}
		ppdu.serialize(c)
	case false:
		ppdu := ipv6PrefixPDU{
			flags:  flag,
			min:    r.MinMask,
			max:    r.MaxMask,
			prefix: ipv6ToByte(IPAddress.To16()),
			asn:    r.ASN,
		}
		ppdu.serialize(c)
	}
}

// Notify client that an update has taken place
func (c *client) notify(serial uint32, session uint16) {
	npdu := serialNotifyPDU{
		Session: session,
		Serial:  serial,
	}
	npdu.serialize(c.conn)

}

// sendEmpty sends an empty response if there is no update required.
func (c *client) sendEmpty(session uint16) {
	cpdu := cacheResponsePDU{
		sessionID: session,
	}
	cpdu.serialize(c.conn)
	epdu := endOfDataPDU{
		sessionID: uint16(session),
		serial:    *c.serial,
		refresh:   uint32(900),
		retry:     uint32(30),
		expire:    uint32(171999),
	}
	epdu.serialize(c.conn)

}

func (c *client) sendRoa() {
	session := rand.Intn(100)
	cpdu := cacheResponsePDU{
		sessionID: uint16(session),
	}
	cpdu.serialize(c.conn)

	c.mutex.RLock()
	for _, roa := range *c.roas {
		writePrefixPDU(&roa, c.conn, announce)
	}
	c.mutex.RUnlock()
	log.Println("Finished sending all prefixes")
	epdu := endOfDataPDU{
		sessionID: uint16(session),
		serial:    *c.serial,
		refresh:   refresh,
		retry:     retry,
		expire:    expire,
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

// Handle each client.
func (c *client) handleClient() {
	log.Printf("Serving %s\n", c.conn.RemoteAddr().String())

	for {

		// What is the incoming PDU?
		var header headerPDU
		binary.Read(c.conn, binary.BigEndian, &header)
		// debug logging for now
		if header != (headerPDU{}) {
			log.Printf("Received: %#v\n", header)
		}

		switch {
		// I only support version 1 for now.
		case header.Version != 1:
			log.Printf("Received something I don't know :'(  %+v\n", header)
			c.error(4, "Unsupported Protocol Version")
			c.conn.Close()
			return

		case header.Ptype == resetQuery:
			var r resetQueryPDU
			binary.Read(c.conn, binary.BigEndian, &r)
			log.Printf("received a reset Query PDU from %s\n", c.addr)
			c.sendRoa()

		case header.Ptype == serialQuery:
			var q serialQueryPDU
			binary.Read(c.conn, binary.BigEndian, &q)
			log.Printf("received a serial Query PDU from %s\n", c.addr)
			// If the client sends in the current or previous serial, then we can handle it.
			// If the serial is older or unknown, we need to send a reset.
			c.mutex.RLock()
			serial := c.diff.newSerial
			c.mutex.RUnlock()
			if q.Serial != serial && q.Serial != serial-1 {
				log.Printf("received a serial query PDU, with an unmanagable serial from %s\n", c.addr)
				log.Printf("Serial received: %d. Current server serial: %d\n", q.Serial, serial)
				c.sendReset()
			}
			if q.Serial == serial {
				log.Printf("received a serial number which currently matches my own from %s\n", c.addr)
				log.Printf("Serial received: %d. Current server serial: %d\n", q.Serial, serial)
				c.sendEmpty(q.Session)
			}
			if q.Serial == serial-1 {
				log.Printf("received a serial number one less, so sending diff to %s\n", c.addr)
				log.Printf("Serial received: %d. Current server serial: %d\n", q.Serial, serial)
				c.mutex.RLock()
				c.sendDiff(c.diff, q.Session)
				c.mutex.RUnlock()
			}
		}
	}
}
