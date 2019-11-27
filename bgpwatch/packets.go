package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"log"
)

const (
	bgpVersion = 4

	// allOnes is max uint64 used for markers
	allOnes = 18446744073709551615

	// BGP message types
	open         = 1
	update       = 2
	notification = 3
	keepalive    = 4
	refresh      = 5

	// Origin codes
	igp        = 0
	egp        = 1
	incomplete = 2

	// as_path values
	asSet      = 1
	asSequence = 2

	// Error codes
	headerError     = 1
	openError       = 2
	updateError     = 3
	holdTimeExpired = 4
	fsmError        = 5
	cease           = 6

	// Open error subcodes
	unsupportedVersion    uint8 = 1
	badPeerAS             uint8 = 2
	badBGPIdentifier      uint8 = 3
	unsupportedParameter  uint8 = 4
	badHoldTime           uint8 = 6
	unsupportedCapability uint8 = 7

	// min and max BGP message size in bytes
	minMessage = 19
	maxMessage = 4096
)

type bgpid [4]byte

func (b *bgpid) string() string {
	return fmt.Sprintf("%v.%v.%v.%v", b[0], b[1], b[2], b[3])
}

type marker struct {
	//This 16-octet field is included for compatibility
	// It MUST be set to all ones.
	_ uint64
	_ uint64
}

type header struct {
	Length uint16
	Type   uint8
}

type msgOpen struct {
	Version  uint8
	ASN      uint16
	HoldTime uint16
	BGPID    bgpid
	ParamLen uint8
}

func (o *msgOpen) serialize(wr io.Writer) {
	m := struct {
		one     uint64
		two     uint64
		length  uint16
		mtype   uint8
		version uint8
		asn     uint16
		hold    uint16
		id      bgpid
		plength uint8
		pcap    uint8
		pl      uint8
		ptype   uint8
		l       uint8
		afi     uint16
		res     uint8
		safi    uint8
	}{
		allOnes,
		allOnes,
		37,
		open,
		4,
		64500,
		30,
		bgpid{2, 2, 2, 2},
		8, // plength
		2,
		6, //pl
		mpbgp,
		4,
		1,
		0,
		1,
	}
	log.Printf("Sending an open message: %#v\n", m)
	binary.Write(wr, binary.BigEndian, m)
}

type parameterHeader struct {
	Type   uint8
	Length uint8
}

type msgCapability struct {
	Code   uint8
	Length uint8
}

func sendKeepAlive(wr io.Writer) {
	k := struct {
		one    uint64
		two    uint64
		length uint16
		ptype  uint8
	}{
		allOnes,
		allOnes,
		19,
		keepalive,
	}
	log.Printf("Sending a keepalive: %#v\n", k)
	binary.Write(wr, binary.BigEndian, k)
}

type optGres struct {
	Restart uint8
	Time    uint8
}

type opt4Byte struct {
	ASN uint32
}

type msgNotification struct {
	Code    uint8
	Subcode uint8
}

func (m *msgNotification) unsupported(wr io.Writer, non []uint8) {
	n := struct {
		one        uint64
		two        uint64
		length     uint16
		ntype      uint8
		code       uint8
		subcode    uint8
		parameters []uint8
	}{
		allOnes,
		allOnes,
		20,
		notification,
		openError,
		unsupportedCapability,
		non,
	}
	log.Printf("%#v\n", n)
	binary.Write(wr, binary.BigEndian, n)

}

// this is just for End-Of-RIB. Needs more work!
type msgUpdate struct {
	Length uint16
	Attr   uint16
}
