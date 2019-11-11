package main

import (
	"encoding/binary"
	"io"
	"log"
	"net"
)

const (
	// PDU Types
	serialNotify  uint8 = 0
	serialQuery   uint8 = 1
	resetQuery    uint8 = 2
	cacheResponse uint8 = 3
	ipv4Prefix    uint8 = 4
	ipv6Prefix    uint8 = 6
	endOfData     uint8 = 7
	cacheReset    uint8 = 8
	routerKey     uint8 = 9
	errorReport   uint8 = 10

	// protocol versions
	version0 uint8 = 0
	version1 uint8 = 1

	// flags
	withdraw uint8 = 0
	announce uint8 = 1
)

type headerPDU struct {
	Version uint8
	Ptype   uint8
}

type serialNotifyPDU struct {
	/*
	   0          8          16         24        31
	   .-------------------------------------------.
	   | Protocol |   PDU    |                     |
	   | Version  |   Type   |     Session ID      |
	   |    1     |    0     |                     |
	   +-------------------------------------------+
	   |                                           |
	   |                Length=12                  |
	   |                                           |
	   +-------------------------------------------+
	   |                                           |
	   |               Serial Number               |
	   |                                           |
	   `-------------------------------------------'
	*/
	Session uint16
	Serial  uint32
}

func (p *serialNotifyPDU) serialize(wr io.Writer) {
	log.Printf("Sending a serial notify PDU: %+v\n", *p)
	binary.Write(wr, binary.BigEndian, version1)
	binary.Write(wr, binary.BigEndian, serialNotify)
	binary.Write(wr, binary.BigEndian, p.Session)
	binary.Write(wr, binary.BigEndian, uint32(12))
	binary.Write(wr, binary.BigEndian, p.Serial)
}

type serialQueryPDU struct {
	/*
	   0          8          16         24        31
	   .-------------------------------------------.
	   | Protocol |   PDU    |                     |
	   | Version  |   Type   |     Session ID      |
	   |    1     |    1     |                     |
	   +-------------------------------------------+
	   |                                           |
	   |                 Length=12                 |
	   |                                           |
	   +-------------------------------------------+
	   |                                           |
	   |               Serial Number               |
	   |                                           |
	   `-------------------------------------------'
	*/
	Session uint16
	Length  uint32
	Serial  uint32
}

type resetQueryPDU struct {
	/*
	   0          8          16         24        31
	   .-------------------------------------------.
	   | Protocol |   PDU    |                     |
	   | Version  |   Type   |         zero        |
	   |    1     |    2     |                     |
	   +-------------------------------------------+
	   |                                           |
	   |                 Length=8                  |
	   |                                           |
	   `-------------------------------------------'
	*/
	Zero   uint16
	Length uint32
}

type cacheResponsePDU struct {
	/*
	   0          8          16         24        31
	   .-------------------------------------------.
	   | Protocol |   PDU    |                     |
	   | Version  |   Type   |     Session ID      |
	   |    1     |    3     |                     |
	   +-------------------------------------------+
	   |                                           |
	   |                 Length=8                  |
	   |                                           |
	   `-------------------------------------------'
	*/
	sessionID uint16
}

func (p *cacheResponsePDU) serialize(wr io.Writer) {
	log.Printf("Sending a cache Repsonse PDU: %v\n", *p)
	binary.Write(wr, binary.BigEndian, version1)
	binary.Write(wr, binary.BigEndian, cacheResponse)
	binary.Write(wr, binary.BigEndian, p.sessionID)
	binary.Write(wr, binary.BigEndian, uint32(8))
}

type ipv4PrefixPDU struct {
	/*
	   0          8          16         24        31
	   .-------------------------------------------.
	   | Protocol |   PDU    |                     |
	   | Version  |   Type   |         zero        |
	   |    1     |    4     |                     |
	   +-------------------------------------------+
	   |                                           |
	   |                 Length=20                 |
	   |                                           |
	   +-------------------------------------------+
	   |          |  Prefix  |   Max    |          |
	   |  Flags   |  Length  |  Length  |   zero   |
	   |          |   0..32  |   0..32  |          |
	   +-------------------------------------------+
	   |                                           |
	   |                IPv4 Prefix                |
	   |                                           |
	   +-------------------------------------------+
	   |                                           |
	   |         Autonomous System Number          |
	   |                                           |
	   `-------------------------------------------'
	*/
	flags  uint8
	min    uint8
	max    uint8
	prefix net.IP // For IPv4 this should be 4 bytes
	asn    uint32
}

func (p *ipv4PrefixPDU) serialize(wr io.Writer) {
	binary.Write(wr, binary.BigEndian, version1)
	binary.Write(wr, binary.BigEndian, ipv4Prefix)
	binary.Write(wr, binary.BigEndian, uint16(0))
	binary.Write(wr, binary.BigEndian, uint32(20))
	binary.Write(wr, binary.BigEndian, p.flags)
	binary.Write(wr, binary.BigEndian, p.min)
	binary.Write(wr, binary.BigEndian, p.max)
	binary.Write(wr, binary.BigEndian, uint8(0))
	binary.Write(wr, binary.BigEndian, p.prefix)
	binary.Write(wr, binary.BigEndian, p.asn)

}

type ipv6PrefixPDU struct {
	/*
	   0          8          16         24        31
	   .-------------------------------------------.
	   | Protocol |   PDU    |                     |
	   | Version  |   Type   |         zero        |
	   |    1     |    6     |                     |
	   +-------------------------------------------+
	   |                                           |
	   |                 Length=32                 |
	   |                                           |
	   +-------------------------------------------+
	   |          |  Prefix  |   Max    |          |
	   |  Flags   |  Length  |  Length  |   zero   |
	   |          |  0..128  |  0..128  |          |
	   +-------------------------------------------+
	   |                                           |
	   +---                                     ---+
	   |                                           |
	   +---            IPv6 Prefix              ---+
	   |                                           |
	   +---                                     ---+
	   |                                           |
	   +-------------------------------------------+
	   |                                           |
	   |         Autonomous System Number          |
	   |                                           |
	   `-------------------------------------------'
	*/
	flags  uint8
	min    uint8
	max    uint8
	prefix net.IP // For IPv6 this should be 16 bytes
	asn    uint32
}

func (p *ipv6PrefixPDU) serialize(wr io.Writer) {
	binary.Write(wr, binary.BigEndian, version1)
	binary.Write(wr, binary.BigEndian, ipv6Prefix)
	binary.Write(wr, binary.BigEndian, uint16(0))
	binary.Write(wr, binary.BigEndian, uint32(32))
	binary.Write(wr, binary.BigEndian, p.flags)
	binary.Write(wr, binary.BigEndian, p.min)
	binary.Write(wr, binary.BigEndian, p.max)
	binary.Write(wr, binary.BigEndian, uint8(0))
	binary.Write(wr, binary.BigEndian, p.prefix)
	binary.Write(wr, binary.BigEndian, p.asn)

}

type endOfDataPDU struct {
	/*
	   0          8          16         24        31
	   .-------------------------------------------.
	   | Protocol |   PDU    |                     |
	   | Version  |   Type   |     Session ID      |
	   |    1     |    7     |                     |
	   +-------------------------------------------+
	   |                                           |
	   |                 Length=24                 |
	   |                                           |
	   +-------------------------------------------+
	   |                                           |
	   |               Serial Number               |
	   |                                           |
	   +-------------------------------------------+
	   |                                           |
	   |              Refresh Interval             |
	   |                                           |
	   +-------------------------------------------+
	   |                                           |
	   |               Retry Interval              |
	   |                                           |
	   +-------------------------------------------+
	   |                                           |
	   |              Expire Interval              |
	   |                                           |
	   `-------------------------------------------'
	*/
	sessionID uint16
	serial    uint32
	refresh   uint32
	retry     uint32
	expire    uint32
}

func (p *endOfDataPDU) serialize(wr io.Writer) {
	binary.Write(wr, binary.BigEndian, version1)
	binary.Write(wr, binary.BigEndian, endOfData)
	binary.Write(wr, binary.BigEndian, p.sessionID)
	binary.Write(wr, binary.BigEndian, uint32(24))
	binary.Write(wr, binary.BigEndian, p.serial)
	binary.Write(wr, binary.BigEndian, p.refresh)
	binary.Write(wr, binary.BigEndian, p.retry)
	binary.Write(wr, binary.BigEndian, p.expire)
	log.Printf("Finished sending end of data PDU: %v\n", *p)
}

type cacheResetPDU struct {
	/*
	   0          8          16         24        31
	   .-------------------------------------------.
	   | Protocol |   PDU    |                     |
	   | Version  |   Type   |         zero        |
	   |    1     |    8     |                     |
	   +-------------------------------------------+
	   |                                           |
	   |                 Length=8                  |
	   |                                           |
	   `-------------------------------------------'
	*/
}

func (p *cacheResetPDU) serialize(wr io.Writer) {
	log.Printf("Sending a cache reset PDU: %v\n", *p)
	binary.Write(wr, binary.BigEndian, version1)
	binary.Write(wr, binary.BigEndian, cacheReset)
	binary.Write(wr, binary.BigEndian, uint16(0))
	binary.Write(wr, binary.BigEndian, uint32(8))
}

type errorReportPDU struct {
	/*
	   0          8          16         24        31
	   .-------------------------------------------.
	   | Protocol |   PDU    |                     |
	   | Version  |   Type   |     Error Code      |
	   |    1     |    10    |                     |
	   +-------------------------------------------+
	   |                                           |
	   |                  Length                   |
	   |                                           |
	   +-------------------------------------------+
	   |                                           |
	   |       Length of Encapsulated PDU          |
	   |                                           |
	   +-------------------------------------------+
	   |                                           |
	   ~               Erroneous PDU               ~
	   |                                           |
	   +-------------------------------------------+
	   |                                           |
	   |           Length of Error Text            |
	   |                                           |
	   +-------------------------------------------+
	   |                                           |
	   |              Arbitrary Text               |
	   |                    of                     |
	   ~          Error Diagnostic Message         ~
	   |                                           |
	   `-------------------------------------------'
	*/
	code   uint16
	report string
}

func (p *errorReportPDU) serialize(wr io.Writer) {
	log.Printf("Sending an error report PDU: %v\n", *p)
	// length of encapped PDU 0 for now
	// not encapping PDU, so empty field there
	// TODO: Make this better of course
	reportLength := len([]byte(p.report))
	totalLength := 128 + reportLength

	binary.Write(wr, binary.BigEndian, version1)
	binary.Write(wr, binary.BigEndian, errorReport)
	binary.Write(wr, binary.BigEndian, p.code)
	binary.Write(wr, binary.BigEndian, totalLength)
	binary.Write(wr, binary.BigEndian, uint32(0))
	binary.Write(wr, binary.BigEndian, reportLength)
	binary.Write(wr, binary.BigEndian, p.report)

}
