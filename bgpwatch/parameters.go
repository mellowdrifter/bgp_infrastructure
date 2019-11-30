package main

import (
	"bytes"
	"encoding/binary"
	"io"
	"io/ioutil"
	"log"
)

const (

	// Open optional parameter types
	capabilities = 2

	// capability codes I support
	// https://www.iana.org/assignments/capability-codes/capability-codes.xhtml
	capMpBgp   uint8 = 1
	cap4Byte   uint8 = 65
	capRefresh uint8 = 70 // Only support enhanced refresh
)

type parameters struct {
	ASN32        uint32
	Refresh      bool
	AddrFamilies []addr
	Supported    []uint8
	Unsupported  []uint8
}

type addr struct {
	AFI  uint16
	_    uint8
	SAFI uint8
}

func decodeOptionalParameters(param *[]byte) parameters {
	r := bytes.NewReader(*param)

	var par parameters
	par.AddrFamilies = []addr{}

	log.Println("*** DECODING PARAMETERS ***")
	var p parameterHeader
	binary.Read(r, binary.BigEndian, &p)

	// following should be some sort of error check
	log.Printf("Parameter type: %d\n", p.Type)
	log.Printf("Parameter length: %d\n", int(p.Length))

	// keep reading the parameters until the parameter field is empty.
	for {
		if r.Len() == 0 {
			break
		}
		var cap msgCapability
		binary.Read(r, binary.BigEndian, &cap)

		// Setting 20 here as a random number which should be big enough for any capability?
		buf := bytes.NewBuffer(make([]byte, 20))
		switch cap.Code {

		case cap4Byte:
			log.Printf("case cap4Byte")
			io.CopyN(buf, r, int64(cap.Length))
			par.ASN32 = decode4OctetAS(buf)
			par.Supported = append(par.Supported, cap.Code)

		case capMpBgp:
			log.Printf("case capMpBgp")
			io.CopyN(buf, r, int64(cap.Length))
			addr := decodeMPBGP(buf)
			log.Printf("AFI is %d, SAFI is %d\n", addr.AFI, addr.SAFI)
			par.AddrFamilies = append(par.AddrFamilies, addr)
			par.Supported = append(par.Supported, cap.Code)

		case capRefresh:
			log.Printf("case capRefresh")
			par.Refresh = true
			par.Supported = append(par.Supported, cap.Code)

		default:
			log.Printf("unsupported")
			par.Unsupported = append(par.Unsupported, cap.Code)
			// As capability is not supported, drop the rest of the parameter message.
			io.CopyN(ioutil.Discard, r, int64(cap.Length))
		}
	}
	return par
}

// parameter 65 is 4-octet AS support
func decode4OctetAS(b *bytes.Buffer) uint32 {
	var ASN uint32
	binary.Read(b, binary.BigEndian, &ASN)
	return ASN
}

func decodeMPBGP(b *bytes.Buffer) addr {
	var afisafi addr
	binary.Read(b, binary.BigEndian, &afisafi)
	return afisafi
}
