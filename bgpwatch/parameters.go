package main

import (
	"bytes"
	"encoding/binary"
	"io"
	"io/ioutil"
	"log"
	"net"
)

const (

	// Open optional parameter types
	capabilities = 2

	// capability codes I support
	// https://www.iana.org/assignments/capability-codes/capability-codes.xhtml
	mpbgp    uint8 = 1
	cap4byte uint8 = 65
)

var (
	supportedCapabilties = map[uint8]bool{
		mpbgp:    true,
		cap4byte: true,
	}
)

type parameters struct {
	fourByteASN uint32
	unsupported []uint8
}

type addr struct {
	AFI  uint16
	_    uint8
	SAFI uint8
}

func decodeOptionalParameters(param []byte) parameters {
	r := bytes.NewReader(param)

	var peerParameters parameters

	log.Println("*** DECODING PARAMETERS ***")
	var par parameterHeader
	binary.Read(r, binary.BigEndian, &par)
	log.Printf("Parameter type: %d\n", par.Type)
	log.Printf("Parameter length: %d\n", int(par.Length))

	// keep reading the parameters until the parameter field is empty.
	for {
		if r.Len() == 0 {
			log.Printf("all finished")
			break
		}
		var cap msgCapability
		binary.Read(r, binary.BigEndian, &cap)
		log.Printf("$$ %#v\n", cap)
		var b []byte
		t := bytes.NewBuffer(b)
		log.Printf("Cap Code is %d\n", cap.Code)
		switch cap.Code {
		// Each handled capability should be read in turn.
		case cap4byte:
			log.Printf("case cap4byte")
			io.CopyN(t, r, int64(cap.Length))
			peerParameters.fourByteASN = decode4OctetAS(t)
		case 1:
			log.Printf("case mpbgp")
			io.CopyN(t, r, int64(cap.Length))
			addr := decodeMPBGP(t)
			log.Printf("AFI is %d, SAFI is %d\n", addr.AFI, addr.SAFI)

		default:
			log.Printf("unsupported")
			peerParameters.unsupported = append(peerParameters.unsupported, cap.Code)
			// As capability is not supporte, drop the rest of the parameter message.
			io.CopyN(ioutil.Discard, r, int64(cap.Length))
		}
		log.Printf("size of r is now %d\n", r.Len())
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

func getParamMessage(c net.Conn) []byte {
	// 4096 is the max BGP packet size
	buffer := make([]byte, 4096)
	// 19 is the minimum size (keepalive)
	io.ReadFull(c, buffer[:19])
	log.Println("Read the first 19 bytes")

	// These calculations should go into a function
	length := int(buffer[16])*256 + int(buffer[17])
	log.Printf("The total packet length will be %d\n", length)

	// Check for errors on these readfulls
	io.ReadFull(c, buffer[19:length])

	// Only the bytes of interest should be returned
	return buffer[:length]

}
