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

func (f *flagType) toString() string {
	return fmt.Sprintf("%v --- %d", f.Flags, f.Code)

}

func decodeRouteAttributes(attr []byte) {
	r := bytes.NewReader(attr)

	log.Println("*** DECODING PREFIX ATTRIBUTES ***")
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
			log.Printf("case ORIGIN")
			io.CopyN(t, r, int64(a.Length))
			decodeOrigin(t)
		case tcASPath:
			log.Printf("case ASPATH")
			io.CopyN(t, r, int64(a.Length))
			fmt.Println("Loads more work needed for AS-PATH")
		case tcNextHop:
			log.Printf("case NEXTHOP")
			io.CopyN(t, r, int64(a.Length))
			fmt.Printf("The next-hop addres is %s\n", fourByteString(decodeNextHop(t)))
		case tcMED:
			log.Printf("case MED")
			io.CopyN(t, r, int64(a.Length))
			fmt.Printf("the MED value is %d\n", decodeMED(t))
		case tcLPref:
			log.Printf("case Local-Pref")
			io.CopyN(t, r, int64(a.Length))
			fmt.Printf("the Local Preferece value is %d\n", decodeLPref(t))
		default:
			log.Printf("not yet implemented")
			io.CopyN(ioutil.Discard, r, int64(a.Length))
			fmt.Printf("Code is %d\n", a.Type.Code)
		}
	}
}

func decodeOrigin(b *bytes.Buffer) {
	var o uint8
	binary.Read(b, binary.BigEndian, &o)
	switch o {
	case 0:
		fmt.Println("Origin is IGP")
	case 1:
		fmt.Println("Origin is EGP")
	case 2:
		fmt.Println("Origin is INCOMPLETE")
	default:
		fmt.Printf("Origin is unknown, value received: %d\n", o)
	}
}

func decodeNextHop(b *bytes.Buffer) ipv4Address {
	// Could there ever be more than 1 IP?
	// Would need to check switch above for v4/v6/dual v6
	var ip ipv4Address
	binary.Read(b, binary.BigEndian, &ip)
	return ip
}

func decodeMED(b *bytes.Buffer) uint32 {
	// won't this be in hex?
	var med uint32
	binary.Read(b, binary.BigEndian, &med)
	return med
}

func decodeLPref(b *bytes.Buffer) uint32 {
	// won't this be in hex?
	var pref uint32
	binary.Read(b, binary.BigEndian, &pref)
	return pref
}

func decodeASPath(b *bytes.Buffer) []uint32 {
	// Probably need a new struct. AS Paths can be part of sets
	// Also the size really depends on how many ASPATHS are in the list!
	var path uint32
	binary.Read(b, binary.BigEndian, &path)
	return []uint32{path}
}
