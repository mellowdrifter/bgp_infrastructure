package clidecode

import "net"

// Decoder is an interface that represents a router to interigate
type Decoder interface {
	// GetBGPTotal returns rib, fib ipv4. rib, fib ipv6
	GetBGPTotal() (Totals, error)

	// GetPeers returns ipv4 peer configured, established. ipv6 peers configured, established
	GetPeers() (Peers, error)

	// GetTotalSourceASNs returns total amount of unique ASNs
	GetTotalSourceASNs() (ASNs, error)

	// GetMasks returns the total count of each mask value
	// First item is IPv4, second item is IPv6
	GetMasks() ([]map[string]uint32, error)

	// GetROAs returns total amount of all ROA states
	GetROAs() (Roas, error)

	// GetLargeCommunities returns the amount of prefixes that have large communities attached (RFC8092)
	GetLargeCommunities() (Large, error)

	// GetIPv4FromSource returns all the IPv4 networks sourced from a source ASN.
	GetIPv4FromSource(uint32) ([]*net.IPNet, error)

	// GetIPv6FromSource returns all the IPv6 networks sourced from a source ASN.
	GetIPv6FromSource(uint32) ([]*net.IPNet, error)
}

// Totals holds the total BGP route count.
type Totals struct {
	V4Rib, V4Fib uint32
	V6Rib, V6Fib uint32
}

// Peers holds the total peers configured.
// c = configured
// e = established
type Peers struct {
	V4c, V4e uint32
	V6c, V6e uint32
}

// ASNs holds counts for all types of ASNs.
// as4:     ASNs originating IPv4
// as6:     ASNs originating IPv6
// as10:    ASNs originating either v4, v6, or both
// as4Only: ASNs originating IPv4 only
// as6Only: ASNs originaring IPv6 only
// asBoth:  ASNs originating both IPv4 and IPv6
type ASNs struct {
	As4, As6, As10   uint32
	As4Only, As6Only uint32
	AsBoth           uint32
}

// Roas holds the ROA state.
// v = valid
// i = invalid
// u = unknown
type Roas struct {
	V4v, V4i, V4u uint32
	V6v, V6i, V6u uint32
}

// Large contains the amount of prefixes with large communities.
type Large struct {
	V4, V6 uint32
}
