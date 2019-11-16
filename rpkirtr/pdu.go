package main

import (
	"encoding/binary"
	"io"
	"log"
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

// headerPDU is used to extract the header of each incoming PDU
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
	pdu := struct {
		version uint8
		ptype   uint8
		session uint16
		length  uint32
		serial  uint32
	}{
		version1,
		serialNotify,
		p.Session,
		uint32(12),
		p.Serial,
	}
	binary.Write(wr, binary.BigEndian, pdu)
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
	log.Printf("Sending a cache Response PDU: %v\n", *p)
	pdu := struct {
		version uint8
		ptype   uint8
		session uint16
		length  uint32
	}{
		version1,
		cacheResponse,
		p.sessionID,
		uint32(8),
	}
	binary.Write(wr, binary.BigEndian, pdu)
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
	prefix [4]byte
	asn    uint32
}

func (p *ipv4PrefixPDU) serialize(wr io.Writer) {

	pdu := struct {
		version uint8
		ptype   uint8
		zero16  uint16
		length  uint32
		flags   uint8
		min     uint8
		max     uint8
		zero8   uint8
		prefix  [4]byte
		asn     uint32
	}{
		version1,
		ipv4Prefix,
		uint16(0),
		uint32(20),
		p.flags,
		p.min,
		p.max,
		uint8(0),
		p.prefix,
		p.asn,
	}
	binary.Write(wr, binary.BigEndian, pdu)

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
	prefix [16]byte
	asn    uint32
}

func (p *ipv6PrefixPDU) serialize(wr io.Writer) {

	pdu := struct {
		version uint8
		ptype   uint8
		zero16  uint16
		length  uint32
		flags   uint8
		min     uint8
		max     uint8
		zero8   uint8
		prefix  [16]byte
		asn     uint32
	}{
		version1,
		ipv6Prefix,
		uint16(0),
		uint32(32),
		p.flags,
		p.min,
		p.max,
		uint8(0),
		p.prefix,
		p.asn,
	}
	binary.Write(wr, binary.BigEndian, pdu)

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
	log.Printf("Sending end of data PDU: %v\n", *p)
	pdu := struct {
		version uint8
		ptype   uint8
		session uint16
		length  uint32
		serual  uint32
		refresh uint32
		retry   uint32
		expire  uint32
	}{
		version1,
		endOfData,
		p.sessionID,
		uint32(24),
		p.serial,
		p.refresh,
		p.retry,
		p.expire,
	}
	binary.Write(wr, binary.BigEndian, pdu)

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
	pdu := struct {
		version uint8
		ptype   uint8
		zero    uint16
		length  uint32
	}{
		version1,
		cacheReset,
		uint16(0),
		uint32(8),
	}
	binary.Write(wr, binary.BigEndian, pdu)
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
