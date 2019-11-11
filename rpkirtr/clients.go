package main

import (
	"log"
	"math/rand"
	"net"
	"strings"
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
}

// reset has no data besides the header
func (c *client) sendReset() {
	rpdu := cacheResetPDU{}
	rpdu.serialize(c.conn)
}

// sendDiff should send additions and deletions to the client.
func (c *client) sendDiff(diff serialDiff, session uint16) {
	cpdu := cacheResponsePDU{
		sessionID: session,
	}
	cpdu.serialize(c.conn)
	if diff.diff {
		for _, roa := range diff.addRoa {
			IPAddress := net.ParseIP(roa.Prefix)
			// TODO put ipv4/ipv6 signal in when creating the ROAs
			switch strings.Contains(roa.Prefix, ":") {
			case true:
				ppdu := ipv6PrefixPDU{
					flags:  announce,
					min:    uint8(roa.MinMask),
					max:    uint8(roa.MaxMask),
					prefix: IPAddress.To16(),
					asn:    uint32(roa.ASN),
				}
				ppdu.serialize(c.conn)
			case false:
				ppdu := ipv4PrefixPDU{
					flags:  announce,
					min:    uint8(roa.MinMask),
					max:    uint8(roa.MaxMask),
					prefix: IPAddress.To4(),
					asn:    uint32(roa.ASN),
				}
				ppdu.serialize(c.conn)
			}
		}
		// TODO: Better to put add/remove all in a single list with the flag type
		for _, roa := range diff.delRoa {
			IPAddress := net.ParseIP(roa.Prefix)
			// TODO put ipv4/ipv6 signal in when creating the ROAs
			switch strings.Contains(roa.Prefix, ":") {
			case true:
				ppdu := ipv6PrefixPDU{
					flags:  withdraw,
					min:    uint8(roa.MinMask),
					max:    uint8(roa.MaxMask),
					prefix: IPAddress.To16(),
					asn:    uint32(roa.ASN),
				}
				ppdu.serialize(c.conn)
			case false:
				ppdu := ipv4PrefixPDU{
					flags:  withdraw,
					min:    uint8(roa.MinMask),
					max:    uint8(roa.MaxMask),
					prefix: IPAddress.To4(),
					asn:    uint32(roa.ASN),
				}
				ppdu.serialize(c.conn)
			}
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

// Notify client that an update has taken place
func (c *client) notify(serial uint32, session uint16) {
	npdu := serialNotifyPDU{
		Session: session,
		Serial:  serial,
	}
	npdu.serialize(c.conn)

}

// sendEmpty sends an empty response. Not sure if this is the right thing to do when getting
// a serial query in which the serial numbers match :/
func (c *client) sendEmpty(session uint16) {
	cpdu := cacheResponsePDU{
		// TODO: Not sure what, where to get this? OR what it's for!
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
		IPAddress := net.ParseIP(roa.Prefix)
		// TODO put ipv4/ipv6 signal in when creating the ROAs
		switch strings.Contains(roa.Prefix, ":") {
		case true:
			ppdu := ipv6PrefixPDU{
				flags:  announce,
				min:    uint8(roa.MinMask),
				max:    uint8(roa.MaxMask),
				prefix: IPAddress.To16(),
				asn:    uint32(roa.ASN),
			}
			ppdu.serialize(c.conn)
		case false:
			ppdu := ipv4PrefixPDU{
				flags:  announce,
				min:    uint8(roa.MinMask),
				max:    uint8(roa.MaxMask),
				prefix: IPAddress.To4(),
				asn:    uint32(roa.ASN),
			}
			ppdu.serialize(c.conn)
		}
	}
	c.mutex.RUnlock()
	log.Println("Finished sending all prefixes")
	epdu := endOfDataPDU{
		sessionID: uint16(session),
		serial:    *c.serial,
		refresh:   uint32(900),
		retry:     uint32(30),
		expire:    uint32(171999),
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
	log.Println("Status of client:")
	log.Printf("Address is %s\n", c.addr)
}
