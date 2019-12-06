package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
)

const (
	// tc is Type Code
	tcOrigin         uint8 = 1
	tcASPath         uint8 = 2
	tcNextHop        uint8 = 3
	tcMED            uint8 = 4
	tcLPref          uint8 = 5
	tcAtoAgg         uint8 = 6
	tcAggregator     uint8 = 7
	tcCommunity      uint8 = 8
	tcMPReachNLRI    uint8 = 14
	tcMPUnreachNLRI  uint8 = 15
	tcLargeCommunity uint8 = 32

	// origin codes
	igp        uint8 = 0
	egp        uint8 = 1
	incomplete uint8 = 2
)

type attrHeader struct {
	Type flagType
}

type flagType struct {
	Flags byte
	Code  uint8
}

type pathAttr struct {
	origin           uint8
	aspath           []asnSegment
	nextHop          string
	med              uint32
	localPref        uint32
	atomic           bool
	agAS             uint32
	agOrigin         net.IP
	communities      []community
	largeCommunities []largeCommunity
	nextHops         []string
	ipv6NLRI         []v6Addr
}

type community struct {
	High uint16
	Low  uint16
}

type largeCommunity struct {
	Admin uint32
	High  uint32
	Low   uint32
}

type prefixAttributes struct {
	attr       *pathAttr
	v4prefixes []v4Addr
	v6prefixes []v6Addr
	v6NextHops []string
}

func (f *flagType) toString() string {
	return fmt.Sprintf("%v --- %d", f.Flags, f.Code)

}

func decodePathAttributes(attr []byte) *pathAttr {
	r := bytes.NewReader(attr)

	var pa pathAttr
	for {
		if r.Len() == 0 {
			break
		}
		// keep reading the attributes until there are none left!
		// Read in header
		var ah attrHeader
		binary.Read(r, binary.BigEndian, &ah)

		var b []byte
		t := bytes.NewBuffer(b)
		switch ah.Type.Code {
		case tcOrigin:
			var length uint8
			binary.Read(r, binary.BigEndian, &length)
			io.CopyN(t, r, int64(length))
			pa.origin = decodeOrigin(t)
		case tcASPath:
			var length uint8
			binary.Read(r, binary.BigEndian, &length)
			io.CopyN(t, r, int64(length))
			pa.aspath = append(pa.aspath, decodeASPath(t)...)
			// Could have both AS_SEQ and AS_SET
			if r.Len() != 0 {
				pa.aspath = append(pa.aspath, decodeASPath(t)...)
			}
		case tcNextHop:
			var length uint8
			binary.Read(r, binary.BigEndian, &length)
			io.CopyN(t, r, int64(length))
			pa.nextHop = decodeIPv4NextHop(t)
		case tcMED:
			var length uint8
			binary.Read(r, binary.BigEndian, &length)
			io.CopyN(t, r, int64(length))
			pa.med = decode4ByteNumber(t)
		case tcLPref:
			var length uint8
			binary.Read(r, binary.BigEndian, &length)
			io.CopyN(t, r, int64(length))
			pa.localPref = decode4ByteNumber(t)
		case tcAtoAgg:
			io.CopyN(ioutil.Discard, r, 1)
			pa.atomic = true
		case tcAggregator:
			var length uint8
			binary.Read(r, binary.BigEndian, &length)
			io.CopyN(t, r, int64(length))
			pa.agAS, pa.agOrigin = decodeAggregator(t)
		case tcMPReachNLRI:
			var length uint16
			binary.Read(r, binary.BigEndian, &length)
			io.CopyN(t, r, int64(length))
			fmt.Printf("MP-NLRI with length: %d\n", length)
			pa.ipv6NLRI, pa.nextHops = decodeMPReachNLRI(t)
		case tcMPUnreachNLRI:
			// This one is strange. Why is length a byte, when MPREACH is 2 bytes?
			// Is this what's used for end of rib?
			var length uint8
			binary.Read(r, binary.BigEndian, &length)
			io.CopyN(t, r, int64(length))
		case tcCommunity:
			var length uint8
			binary.Read(r, binary.BigEndian, &length)
			io.CopyN(t, r, int64(length))
			pa.communities = decodeCommunities(t, length)
		case tcLargeCommunity:
			var length uint8
			binary.Read(r, binary.BigEndian, &length)
			io.CopyN(t, r, int64(length))
			pa.largeCommunities = decodeLargeCommunities(t, length)

		default:
			log.Printf("Type Code %d is not yet implemented", ah.Type.Code)
			var length uint8
			binary.Read(r, binary.BigEndian, &length)
			io.CopyN(ioutil.Discard, r, int64(length))
		}
	}
	return &pa
}

func decodeOrigin(b *bytes.Buffer) uint8 {
	var o uint8
	binary.Read(b, binary.BigEndian, &o)

	return o
}

func decodeIPv4NextHop(b *bytes.Buffer) string {
	ip := bytes.NewBuffer(make([]byte, 0, 4))
	io.Copy(ip, b)
	return net.IP(ip.Bytes()).String()
}

func decodeIPv6NextHop(b *bytes.Buffer) string {
	ip := bytes.NewBuffer(make([]byte, 0, 16))
	io.Copy(ip, b)
	return net.IP(ip.Bytes()).String()
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

func decodeAggregator(b *bytes.Buffer) (uint32, net.IP) {
	ip := bytes.NewBuffer(make([]byte, 0, 4))
	var asn uint32
	binary.Read(b, binary.BigEndian, &asn)
	io.Copy(ip, b)
	return asn, net.IP(ip.Bytes())
}

func decodeCommunities(b *bytes.Buffer, len uint8) []community {
	var communities = make([]community, 0, len/4)
	for {
		if b.Len() == 0 {
			break
		}
		var comm community
		binary.Read(b, binary.BigEndian, &comm)
		communities = append(communities, comm)
	}
	return communities
}

func decodeLargeCommunities(b *bytes.Buffer, len uint8) []largeCommunity {
	var communities = make([]largeCommunity, 0, len/12)
	for {
		if b.Len() == 0 {
			break
		}
		var comm largeCommunity
		binary.Read(b, binary.BigEndian, &comm)
		communities = append(communities, comm)
	}
	return communities
}

// BGP only encodes the prefix up to the subnet value in bits, and then pads zeros until the end of the octet.
func decodeIPv4NLRI(b *bytes.Reader) []v4Addr {
	var addrs []v4Addr
	for {
		if b.Len() == 0 {
			break
		}

		var mask uint8
		binary.Read(b, binary.BigEndian, &mask)

		addrs = append(addrs, v4Addr{
			Mask:   mask,
			Prefix: getIPv4Prefix(b, mask),
		})
	}

	return addrs
}

func getIPv4Prefix(b *bytes.Reader, mask uint8) net.IP {
	prefix := bytes.NewBuffer(make([]byte, 0, 4))

	switch {
	case mask >= 1 && mask <= 8:
		io.CopyN(prefix, b, 1)
	case mask >= 9 && mask <= 16:
		io.CopyN(prefix, b, 2)
	case mask >= 17 && mask <= 24:
		io.CopyN(prefix, b, 3)
	case mask >= 25:
		io.CopyN(prefix, b, 4)
	}

	return net.IP(prefix.Bytes())
}

func getIPv6Prefix(b *bytes.Buffer, mask uint8) net.IP {
	prefix := bytes.NewBuffer(make([]byte, 0, 16))

	switch {
	case mask >= 1 && mask <= 8:
		io.CopyN(prefix, b, 1)
	case mask >= 9 && mask <= 16:
		io.CopyN(prefix, b, 2)
	case mask >= 17 && mask <= 24:
		io.CopyN(prefix, b, 3)
	case mask >= 25 && mask <= 32:
		io.CopyN(prefix, b, 4)
	case mask >= 33 && mask <= 40:
		io.CopyN(prefix, b, 5)
	case mask >= 41 && mask <= 48:
		io.CopyN(prefix, b, 6)
	case mask >= 49 && mask <= 56:
		io.CopyN(prefix, b, 7)
	case mask >= 57 && mask <= 64:
		io.CopyN(prefix, b, 8)
	case mask >= 65 && mask <= 72:
		io.CopyN(prefix, b, 9)
	case mask >= 73 && mask <= 80:
		io.CopyN(prefix, b, 10)
	case mask >= 81 && mask <= 88:
		io.CopyN(prefix, b, 11)
	case mask >= 89 && mask <= 96:
		io.CopyN(prefix, b, 12)
	case mask >= 97 && mask <= 104:
		io.CopyN(prefix, b, 13)
	case mask >= 105 && mask <= 112:
		io.CopyN(prefix, b, 14)
	case mask >= 113 && mask <= 120:
		io.CopyN(prefix, b, 15)
	case mask >= 121 && mask <= 128:
		io.CopyN(prefix, b, 16)
	}

	for prefix.Len() < 16 {
		prefix.WriteByte(0)
	}

	return net.IP(prefix.Bytes())
}

func decodeMPReachNLRI(b *bytes.Buffer) ([]v6Addr, []string) {
	// AFI/SAFI - For now I only IPv6 Unicast
	var afi uint16
	var safi uint8
	// Could be two next-hops
	var nextHops []string
	binary.Read(b, binary.BigEndian, &afi)
	binary.Read(b, binary.BigEndian, &safi)
	log.Println(afi)
	log.Println(safi)
	// In the above, I'm really only supporting IPv6 here. The rest is dependant on which AFI/SAFI

	// If the next-hop length is 32 bytes, we have both a public and link-local
	// If the next-hop length is only 16 bytes, the next-hop should be public only
	// But if the actual next-hop is link-local, the initial next-hop is :: ?
	var nhLen uint8
	binary.Read(b, binary.BigEndian, &nhLen)
	log.Println(nhLen)

	nh := bytes.NewBuffer(make([]byte, 0, 16))
	io.CopyN(nh, b, 16)
	nextHops = append(nextHops, decodeIPv6NextHop(nh))

	if nhLen == 32 {
		llnh := bytes.NewBuffer(make([]byte, 0, 16))
		io.CopyN(llnh, b, 16)
		nextHops = append(nextHops, decodeIPv6NextHop(llnh))
	}

	// Ignore one byte SNPA
	io.CopyN(ioutil.Discard, b, 1)

	// Pass the remainder of the buffer to be decoded into NLRI
	return decodeIPv6NLRI(b), nextHops

}

// BGP only encodes the prefix up to the subnet value in bits, and then pads zeros until the end of the octet.
func decodeIPv6NLRI(b *bytes.Buffer) []v6Addr {
	var addrs []v6Addr
	for {
		if b.Len() == 0 {
			break
		}

		var mask uint8
		binary.Read(b, binary.BigEndian, &mask)

		addrs = append(addrs, v6Addr{
			Mask:   mask,
			Prefix: getIPv6Prefix(b, mask),
		})
	}
	return addrs
}
