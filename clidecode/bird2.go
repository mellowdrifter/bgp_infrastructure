package clidecode

import (
	"fmt"
	"net"
	"strings"

	c "github.com/mellowdrifter/bgp_infrastructure/common"
)

// Bird2Conn will be a connection to a Bird2 instance. In reality this
// will need to be on the local server
type Bird2Conn struct{}

// GetBGPTotal returns rib, fib ipv4. rib, fib ipv6
func (b Bird2Conn) GetBGPTotal() (Totals, error) {
	cmd := "/usr/sbin/birdc show route count | grep routes | awk {'print $3, $6'}"

	var t Totals
	out, err := c.GetOutput(cmd)
	if err != nil {
		return t, err
	}
	numbers := strings.Fields(out)

	t.V4Rib = c.StringToUint32(numbers[0])
	t.V4Fib = c.StringToUint32(numbers[1])
	t.V6Rib = c.StringToUint32(numbers[2])
	t.V6Fib = c.StringToUint32(numbers[3])

	return t, nil
}

// GetPeers returns ipv4 peer configured, established. ipv6 peers configured, established
func (b Bird2Conn) GetPeers() (Peers, error) {
	var peers []uint32
	cmds := []string{
		"/usr/sbin/birdc show protocols | awk {'print $1'} | grep _v4 | grep -Ev 'BIRD|device1|name|info|kernel1' | wc -l",
		"/usr/sbin/birdc show protocols | awk {'print $1 $6'} | grep _v4 | grep Estab | wc -l",
		"/usr/sbin/birdc show protocols | awk {'print $1'} | grep _v6 | grep -Ev 'BIRD|device1|name|info|kernel1' | wc -l",
		"/usr/sbin/birdc show protocols | awk {'print $1 $6'} | grep _v6 | grep Estab | wc -l",
	}

	var p Peers

	for _, cmd := range cmds {
		out, err := c.GetOutput(cmd)
		if err != nil {
			return p, err
		}
		peers = append(peers, c.StringToUint32(out))
	}

	p.V4c = peers[0]
	p.V4e = peers[1]
	p.V6c = peers[2]
	p.V6e = peers[3]

	return p, nil

}

// GetTotalSourceASNs returns total amount of unique ASNs
// as4:     ASNs originating IPv4
// as6:     ASNs originating IPv6
// as10:    ASNs originating either v4, v6, or both
// as4Only: ASNs originating IPv4 only
// as6Only: ASNs originaring IPv6 only
// asBoth:  ASNs originating both IPv4 and IPv6
func (b Bird2Conn) GetTotalSourceASNs() (ASNs, error) {
	cmd1 := "/usr/sbin/birdc show route primary table master4 | awk '{print $NF}' | tr -d '[]ASie?' | sed -e '1,2d'"
	cmd2 := "/usr/sbin/birdc show route primary table master6 | awk '{print $NF}' | tr -d '[]ASie?' | sed -e '1,2d'"

	var s ASNs
	as4, err := c.GetOutput(cmd1)
	if err != nil {
		return s, err
	}
	as6, err := c.GetOutput(cmd2)
	if err != nil {
		return s, err
	}

	// asNSet is the list of unique source AS numbers in each address family
	as4Set := c.SetListOfStrings(strings.Fields(as4))
	as6Set := c.SetListOfStrings(strings.Fields(as6))

	// as10Set is the total number of unique source AS numbers.
	var as10 []string
	as10 = append(as10, as4Set...)
	as10 = append(as10, as6Set...)
	as10Set := c.SetListOfStrings(as10)

	// as4Only is the set of ASs that only source IPv4
	as4Only := c.InFirstButNotSecond(as4Set, as6Set)

	// as6Only is the set of ASs that only source IPv6
	as6Only := c.InFirstButNotSecond(as6Set, as4Set)

	// asBoth is the set of ASs that source both IPv4 and IPv6
	asBoth := c.Intersection(as4Set, as6Set)

	s.As4 = uint32(len(as4Set))
	s.As6 = uint32(len(as6Set))
	s.As10 = uint32(len(as10Set))
	s.As4Only = uint32(len(as4Only))
	s.As6Only = uint32(len(as6Only))
	s.AsBoth = uint32(len(asBoth))

	return s, nil

}

// GetROAs returns total amount of all ROA states
func (b Bird2Conn) GetROAs() (Roas, error) {
	var r Roas
	var roas []uint32
	cmds := []string{
		"/usr/sbin/birdc 'show route primary table master4 where roa_check(roa_v4, net, bgp_path.last) = ROA_VALID' | sed -e '1,2d' | wc -l",
		"/usr/sbin/birdc 'show route primary table master4 where roa_check(roa_v4, net, bgp_path.last) = ROA_INVALID' | sed -e '1,2d' | wc -l",
		"/usr/sbin/birdc 'show route primary table master4 where roa_check(roa_v4, net, bgp_path.last) = ROA_UNKNOWN' | sed -e '1,2d' | wc -l",
		"/usr/sbin/birdc 'show route primary table master6 where roa_check(roa_v6, net, bgp_path.last) = ROA_VALID' | sed -e '1,2d' | wc -l",
		"/usr/sbin/birdc 'show route primary table master6 where roa_check(roa_v6, net, bgp_path.last) = ROA_INVALID' | sed -e '1,2d' | wc -l",
		"/usr/sbin/birdc 'show route primary table master6 where roa_check(roa_v6, net, bgp_path.last) = ROA_UNKNOWN' | sed -e '1,2d' | wc -l",
	}

	for _, cmd := range cmds {
		out, err := c.GetOutput(cmd)
		if err != nil {
			return r, err
		}
		roas = append(roas, c.StringToUint32(out))
	}

	r.V4v = roas[0]
	r.V4i = roas[1]
	r.V4u = roas[2]
	r.V6v = roas[3]
	r.V6i = roas[4]
	r.V6u = roas[5]

	return r, nil

}

// GetMasks returns the total count of each mask value
// First item is IPv4, second item is IPv6
func (b Bird2Conn) GetMasks() ([]map[string]uint32, error) {
	v6 := make(map[string]uint32)
	v4 := make(map[string]uint32)
	var m []map[string]uint32

	cmd := "/usr/sbin/birdc show route primary table master6 | awk {'print $1'} | sed -e '1,2d'"
	subnetsV6, err := c.GetOutput(cmd)
	if err != nil {
		return m, err
	}
	for _, s := range strings.Fields(subnetsV6) {
		mask := strings.Split(s, "::/")[1]
		v6[mask]++
	}

	cmd2 := "/usr/sbin/birdc show route primary table master4 | awk {'print $1'} | sed -e '1,2d'"
	subnetsV4, err := c.GetOutput(cmd2)
	if err != nil {
		return m, err
	}
	for _, s := range strings.Fields(subnetsV4) {
		mask := strings.Split(s, "/")[1]
		v4[mask]++
	}

	m = append(m, v4)
	m = append(m, v6)

	return m, nil
}

// GetLargeCommunities returns the amount of prefixes that have large communities attached (RFC8092)
// TODO: Not sure this is doing the right thing
func (b Bird2Conn) GetLargeCommunities() (Large, error) {
	var l Large
	var comm []uint32
	cmds := []string{
		"/usr/sbin/birdc 'show route primary table master4 where bgp_large_community ~ [(*,*,*)]' | sed -e '1,2d' | wc -l",
		"/usr/sbin/birdc 'show route primary table master6 where bgp_large_community ~ [(*,*,*)]' | sed -e '1,2d' | wc -l",
	}

	for _, cmd := range cmds {
		out, err := c.GetOutput(cmd)
		if err != nil {
			return l, err
		}
		comm = append(comm, c.StringToUint32(out))
	}
	l.V4 = comm[0]
	l.V6 = comm[1]

	return l, nil
}

// GetIPv4FromSource returns all the IPv4 networks sourced from a source ASN.
func (b Bird2Conn) GetIPv4FromSource(asn uint32) ([]*net.IPNet, error) {

	cmd := fmt.Sprintf("/usr/sbin/birdc 'show route primary table master4 where bgp_path ~ [= * %d =]' | grep -Ev 'BIRD|device1|name|info|kernel1|Table' | awk '{print $1}'", asn)
	out, err := c.GetOutput(cmd)
	if err != nil {
		return []*net.IPNet{}, err
	}

	var ips []*net.IPNet

	for _, address := range strings.Fields(out) {
		_, net, _ := net.ParseCIDR(address)
		ips = append(ips, net)
	}

	return ips, nil
}

// GetIPv6FromSource returns all the IPv6 networks sourced from a source ASN.
func (b Bird2Conn) GetIPv6FromSource(asn uint32) ([]*net.IPNet, error) {

	cmd := fmt.Sprintf("/usr/sbin/birdc 'show route primary table master6 where bgp_path ~ [= * %d =]' | grep -Ev 'BIRD|device1|name|info|kernel1|Table' | awk '{print $1}'", asn)
	out, err := c.GetOutput(cmd)
	if err != nil {
		return []*net.IPNet{}, err
	}

	var ips []*net.IPNet

	for _, address := range strings.Fields(out) {
		_, net, _ := net.ParseCIDR(address)
		ips = append(ips, net)
	}

	return ips, nil
}
