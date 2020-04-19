package clidecode

import "net"

// fakeConn will be a connection to a fake instance.
type fakeConn struct{}

// GetBGPTotal returns rib, fib ipv4. rib, fib ipv6
func (f fakeConn) GetBGPTotal() (Totals, error) {
	return Totals{}, nil
}

// GetPeers returns ipv4 peer configured, established. ipv6 peers configured, established
func (f fakeConn) GetPeers() (Peers, error) {
	return Peers{}, nil
}

// GetTotalSourceASNs returns total amount of unique ASNs
func (f fakeConn) GetTotalSourceASNs() (ASNs, error) {
	return ASNs{}, nil
}

// GetMasks returns the total count of each mask value
// First item is IPv4, second item is IPv6
func (f fakeConn) GetMasks() ([]map[string]uint32, error) {
	v4 := make(map[string]uint32)
	v6 := make(map[string]uint32)
	return []map[string]uint32{v4, v6}, nil
}

// GetROAs returns total amount of all ROA states
func (f fakeConn) GetROAs() (Roas, error) {
	return Roas{}, nil
}

// GetLargeCommunities returns the amount of prefixes that have large communities attached (RFC8092)
func (f fakeConn) GetLargeCommunities() (Large, error) {
	return Large{}, nil
}

// GetIPv4FromSource returns all the IPv4 networks sourced from a source ASN.
func (f fakeConn) GetIPv4FromSource(uint32) ([]*net.IPNet, error) {
	return nil, nil
}

// GetIPv6FromSource returns all the IPv6 networks sourced from a source ASN.
func (f fakeConn) GetIPv6FromSource(uint32) ([]*net.IPNet, error) {
	return nil, nil
}

// GetOriginFromIP will return the origin ASN from a source IP.
func (f fakeConn) GetOriginFromIP(net.IP) (uint32, bool, error) {
	return 0, false, nil
}

// GetASPathFromIP will return the AS path, as well as as-set if any from a source IP.
func (f fakeConn) GetASPathFromIP(net.IP) (ASPath, bool, error) {
	return ASPath{}, false, nil
}

// GetRoute will return the current FIB entry, if any, from a source IP.
func (f fakeConn) GetRoute(net.IP) (*net.IPNet, bool, error) {
	return nil, false, nil
}

// GetROA will return the ROA status, if any, from a source IP.
func (f fakeConn) GetROA(*net.IPNet) (int, bool, error) {
	return 0, false, nil
}
