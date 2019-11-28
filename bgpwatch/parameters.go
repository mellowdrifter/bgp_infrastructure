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
	mpbgp    uint8 = 1
	cap4byte uint8 = 65
)

type parameters struct {
	fourByteASN  uint32
	addrFamilies []addr
	supported    []uint8
	unsupported  []uint8
}

type addr struct {
	AFI  uint16
	_    uint8
	SAFI uint8
}

func decodeOptionalParameters(param []byte) parameters {
	r := bytes.NewReader(param)

	var peerParameters parameters
	peerParameters.addrFamilies = []addr{}

	log.Println("*** DECODING PARAMETERS ***")
	var par parameterHeader
	binary.Read(r, binary.BigEndian, &par)

	// following should be some sort of error check
	log.Printf("Parameter type: %d\n", par.Type)
	log.Printf("Parameter length: %d\n", int(par.Length))

	// keep reading the parameters until the parameter field is empty.
	for {
		if r.Len() == 0 {
			break
		}
		var cap msgCapability
		binary.Read(r, binary.BigEndian, &cap)
		var b []byte
		t := bytes.NewBuffer(b)
		switch cap.Code {
		// Each handled capability should be read in turn.
		case cap4byte:
			log.Printf("case cap4byte")
			io.CopyN(t, r, int64(cap.Length))
			peerParameters.fourByteASN = decode4OctetAS(t)
			peerParameters.supported = append(peerParameters.supported, cap.Code)

		case mpbgp:
			log.Printf("case mpbgp")
			io.CopyN(t, r, int64(cap.Length))
			addr := decodeMPBGP(t)
			log.Printf("AFI is %d, SAFI is %d\n", addr.AFI, addr.SAFI)
			peerParameters.addrFamilies = append(peerParameters.addrFamilies, addr)
			peerParameters.supported = append(peerParameters.supported, cap.Code)

		default:
			log.Printf("unsupported")
			peerParameters.unsupported = append(peerParameters.unsupported, cap.Code)
			// As capability is not supported, drop the rest of the parameter message.
			io.CopyN(ioutil.Discard, r, int64(cap.Length))
		}
	}

	return peerParameters
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
