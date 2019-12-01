package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"io/ioutil"
	"log"
)

const (
	// tc is Type Code
	tcOrigin     uint8 = 1
	tcASPath     uint8 = 2
	tcNextHop    uint8 = 3
	tcMED        uint8 = 4
	tcLPref      uint8 = 5
	tcAtoAgg     uint8 = 6
	tcAggregator uint8 = 7

	// origin codes
	igp        uint8 = 0
	egp        uint8 = 1
	incomplete uint8 = 2
)

type attrHeader struct {
	Type   flagType
	Length uint8
}

type flagType struct {
	Flags byte
	Code  uint8
}

type pathAttr struct {
	origin   uint8
	aspath   []asnSegment
	nextHop  string
	med      uint32
	lPref    uint32
	atomic   bool
	agAS     uint32
	agOrigin ipv4Address
}

type prefixAttributes struct {
	attr     *pathAttr
	prefixes []v4Addr
}

func (f *flagType) toString() string {
	return fmt.Sprintf("%v --- %d", f.Flags, f.Code)

}

func decodeRouteAttributes(attr []byte) *pathAttr {
	r := bytes.NewReader(attr)

	log.Println("*** DECODING PREFIX ATTRIBUTES ***")
	var pa pathAttr
	for {
		if r.Len() == 0 {
			break
		}
		// keep reading the attributes until there are none left!
		// Read in header
		var a attrHeader
		binary.Read(r, binary.BigEndian, &a)

		var b []byte
		t := bytes.NewBuffer(b)
		switch a.Type.Code {
		case tcOrigin:
			io.CopyN(t, r, int64(a.Length))
			pa.origin = decodeOrigin(t)
		case tcASPath:
			io.CopyN(t, r, int64(a.Length))
			pa.aspath = append(pa.aspath, decodeASPath(t)...)
			// Could have both AS_SEQ and AS_SET
			if r.Len() != 0 {
				pa.aspath = append(pa.aspath, decodeASPath(t)...)
			}
		case tcNextHop:
			io.CopyN(t, r, int64(a.Length))
			pa.nextHop = decodeNextHop(t)
		case tcMED:
			io.CopyN(t, r, int64(a.Length))
			pa.med = decode4ByteNumber(t)
		case tcLPref:
			io.CopyN(t, r, int64(a.Length))
			pa.lPref = decode4ByteNumber(t)
		case tcAtoAgg:
			pa.atomic = true
		case tcAggregator:
			io.CopyN(t, r, int64(a.Length))
			pa.agAS, pa.agOrigin = decodeAggregator(t)
		default:
			log.Printf("Type Code %d is not yet implemented", a.Type.Code)
			io.CopyN(ioutil.Discard, r, int64(a.Length))
		}
	}
	return &pa
}

func decodeOrigin(b *bytes.Buffer) uint8 {
	var o uint8
	binary.Read(b, binary.BigEndian, &o)

	return o
}

func decodeNextHop(b *bytes.Buffer) string {
	// Could there ever be more than 1 IP?
	// Would need to check switch above for v4/v6/dual v6
	var ip ipv4Address
	binary.Read(b, binary.BigEndian, &ip)
	return fourByteString(ip)
}

func decode4ByteNumber(b *bytes.Buffer) uint32 {
	var n uint32
	binary.Read(b, binary.BigEndian, &n)
	return n
}

type asnTL struct {
	Type   uint8
	Length uint8
}

type asnSegment struct {
	Type uint8
	ASN  uint32
}

// If empty, could be iBGP update and so should deal with that
func decodeASPath(b *bytes.Buffer) []asnSegment {
	var asnTL asnTL
	binary.Read(b, binary.BigEndian, &asnTL)
	var asns = make([]asnSegment, asnTL.Length)
	for i := uint8(0); i < asnTL.Length; i++ {
		var asn asnSegment
		asn.Type = asnTL.Type
		binary.Read(b, binary.BigEndian, &asn.ASN)
		asns[i] = asn
	}
	return asns
}

func decodeAggregator(b *bytes.Buffer) (uint32, ipv4Address) {
	var asn uint32
	var ip ipv4Address
	binary.Read(b, binary.BigEndian, &asn)
	binary.Read(b, binary.BigEndian, &ip)
	return asn, ip
}
