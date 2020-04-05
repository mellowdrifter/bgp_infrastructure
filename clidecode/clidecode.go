package clidecode

// Decoder is an interface that represents a router to interigate
type Decoder interface {
	// GetBGPTotal returns rib, fib ipv4. rib, fib ipv6
	GetBGPTotal() (Totals, error)

	// GetPeers returns ipv4 peer configured, established. ipv6 peers configured, established
	GetPeers() (Peers, error)

	// GetTotalSourceASNs returns total amount of unique ASNs
	GetTotalSourceASNs() (SourceASNs, error)

	// GetROAs returns total amount of all ROA states
	GetROAs() (Roas, error)
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

// SourceASNs holds counts for all types of source ASNs.
// as4:     ASNs originating IPv4
// as6:     ASNs originating IPv6
// as10:    ASNs originating either v4, v6, or both
// as4Only: ASNs originating IPv4 only
// as6Only: ASNs originaring IPv6 only
// asBoth:  ASNs originating both IPv4 and IPv6
type SourceASNs struct {
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
