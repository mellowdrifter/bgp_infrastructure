package main

import (
	"fmt"
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
				flags:  uint8(1),
				min:    uint8(roa.MinMask),
				max:    uint8(roa.MaxMask),
				prefix: IPAddress.To16(),
				asn:    uint32(roa.ASN),
			}
			ppdu.serialize(c.conn)
		case false:
			ppdu := ipv4PrefixPDU{
				flags:  uint8(1),
				min:    uint8(roa.MinMask),
				max:    uint8(roa.MaxMask),
				prefix: IPAddress.To4(),
				asn:    uint32(roa.ASN),
			}
			ppdu.serialize(c.conn)
		}
	}
	c.mutex.RUnlock()
	fmt.Println("Finished sending all prefixes")
	epdu := endOfDataPDU{
		sessionID: uint16(session),
		//serial:    cacheSerial,
		refresh: uint32(900),
		retry:   uint32(30),
		expire:  uint32(171999),
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
	fmt.Println("Status of client:")
	fmt.Printf("Address is %s\n", c.addr)
	c.mutex.RLock()
	fmt.Printf("Serial is %d\n", *c.serial)
	c.mutex.RUnlock()

}
