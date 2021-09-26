package clidecode

import (
	"fmt"
	"net"
	"regexp"
	"strconv"
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
		"/usr/sbin/birdc 'show route primary table master4 where roa_check(roa_v4) = ROA_VALID count' | sed -e '1d'",
		"/usr/sbin/birdc 'show route primary table master4 where roa_check(roa_v4) = ROA_INVALID count' | sed -e '1d'",
		"/usr/sbin/birdc 'show route primary table master4 where roa_check(roa_v4) = ROA_UNKNOWN count' | sed -e '1d'",
		"/usr/sbin/birdc 'show route primary table master6 where roa_check(roa_v6) = ROA_VALID count' | sed -e '1d'",
		"/usr/sbin/birdc 'show route primary table master6 where roa_check(roa_v6) = ROA_INVALID count' | sed -e '1d'",
		"/usr/sbin/birdc 'show route primary table master6 where roa_check(roa_v6) = ROA_UNKNOWN count' | sed -e '1d'",
	}

	for _, cmd := range cmds {
		out, err := c.GetOutput(cmd)
		if err != nil {
			return r, err
		}
		split := strings.Split(out, " ")
		roas = append(roas, c.StringToUint32(split[0]))
	}

	r.V4v = roas[0]
	r.V4i = roas[1]
	r.V4u = roas[2]
	r.V6v = roas[3]
	r.V6i = roas[4]
	r.V6u = roas[5]

	return r, nil
}

// GetInvalids returns a map of ASNs that are advertising RPKI invalid prefixes.
// It also includes all those prefixes being advertised.
func (b Bird2Conn) GetInvalids() (map[string][]string, error) {
	inv := make(map[string][]string)
	num := regexp.MustCompile(`[\d]+`)
	cmds := []string{
		"/usr/sbin/birdc 'show route primary table master4 where roa_check(roa_v4) = ROA_INVALID' | sed -e '1,2d' | awk {'print $NF,$1'}",
		"/usr/sbin/birdc 'show route primary table master6 where roa_check(roa_v6) = ROA_INVALID' | sed -e '1,2d' | awk {'print $NF,$1'}",
	}

	for _, cmd := range cmds {
		out, err := c.GetOutput(cmd)
		if err != nil {
			return inv, err
		}
		lines := strings.Split(out, "\n")
		for _, v := range lines {
			split := strings.Split(v, " ")
			asn := num.FindString(split[0])
			// Ignore invalids when bird doesn't show the source ASN (when AS-SET is used)
			if asn != "" {
				inv[asn] = append(inv[asn], split[1])
			}
		}
	}

	return inv, nil
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
		return nil, err
	}

	var ips []*net.IPNet

	for _, address := range strings.Fields(out) {
		_, net, _ := net.ParseCIDR(address)
		ips = append(ips, net)
	}

	return ips, nil
}

// GetASPathFromIP will return the AS path, as well as as-set if any from a source IP.
func (b Bird2Conn) GetASPathFromIP(ip net.IP) (ASPath, bool, error) {
	var aspath ASPath

	cmd := fmt.Sprintf("/usr/sbin/birdc show route primary all for %s | grep -Ev 'BIRD|device1|name|info|kernel1|Table' | grep as_path | awk '{$1=\"\"; print $0}'", ip.String())
	out, err := c.GetOutput(cmd)
	if err != nil {
		return aspath, false, err
	}

	// If no route exists, no as-path will exist
	if out == "" {
		return aspath, false, nil
	}

	path, set := decodeASPaths(out)
	aspath.Path = path
	aspath.Set = set

	return aspath, true, nil
}

// decodeASPaths will return a slice of AS & AS-Sets from a string as-path output.
func decodeASPaths(in string) ([]uint32, []uint32) {
	if strings.ContainsAny(in, "{}") {
		in = strings.Replace(in, "{", "{ ", 1)
		in = strings.Replace(in, "}", " }", 1)
	}
	paths := strings.Fields(in)
	var path, set []uint32

	// Need to separate as-set
	var isSet bool
	for _, as := range paths {
		if strings.ContainsAny(as, "{}") {
			isSet = true
			continue
		}

		switch {
		case isSet == false:
			path = append(path, c.StringToUint32(as))
		case isSet == true:
			set = append(set, c.StringToUint32(as))
		}
	}

	return path, set
}

// GetRoute will return the current FIB entry, if any, from a source IP.
func (b Bird2Conn) GetRoute(ip net.IP) (*net.IPNet, bool, error) {
	cmd := fmt.Sprintf("/usr/sbin/birdc show route primary for %s | grep -Ev 'BIRD|device1|name|info|kernel1|Table' | awk '{print $1}'", ip.String())
	out, err := c.GetOutput(cmd)
	if err != nil {
		return nil, false, err
	}

	_, net, err := net.ParseCIDR(out)
	if err != nil {
		return nil, false, nil
	}

	return net, true, nil
}

// GetOriginFromIP will return the origin ASN from a source IP.
func (b Bird2Conn) GetOriginFromIP(ip net.IP) (uint32, bool, error) {
	cmd := fmt.Sprintf("/usr/sbin/birdc show route primary all for %s | grep -Ev 'BIRD|device1|name|info|kernel1|Table' | grep as_path | sed 's/{.*}//' | awk {'print $NF'}", ip.String())
	out, err := c.GetOutput(cmd)
	if err != nil {
		return 0, false, err
	}

	num := regexp.MustCompile("[0-9]+")
	o := num.FindString(out)
	if o == "" {
		return 0, false, nil
	}

	source, err := strconv.Atoi(o)
	if err != nil {
		return 0, true, err
	}

	return uint32(source), true, nil
}

// GetROA will return the ROA status from a prefix and ASN.
// This function does not check for the existance of the prefix in the table.
func (b Bird2Conn) GetROA(prefix *net.IPNet, asn uint32) (int, bool, error) {
	var table string
	if strings.Contains(prefix.String(), ":") {
		table = "roa_v6"
	} else {
		table = "roa_v4"
	}

	cmd := fmt.Sprintf("/usr/sbin/birdc 'eval roa_check(%s, %s, %d)'", table, prefix, asn)
	out, err := c.GetOutput(cmd)
	if err != nil {
		return 0, false, err
	}

	// Get the enum value
	// example output - (enum 35)1
	val := out[len(out)-1:]

	// Check for an existing ROA
	// 0 = ROA_UNKNOWN
	// 1 = ROA_VALID
	// 2 = ROA_INVALID
	statuses := map[string]int{
		"0": RUnknown,
		"2": RInvalid,
		"1": RValid,
	}

	return statuses[val], true, nil
}

// GetVRPs will return all Validated ROA Payloads for an ASN.
func (b Bird2Conn) GetVRPs(asn uint32) ([]VRP, error) {
	var VRPs []VRP
	cmds := []string{
		fmt.Sprintf("/usr/sbin/birdc show route all table roa_v4 where net.asn=%d | grep -Ev 'BIRD|device1|name|info|kernel1|Table' |grep %d | awk {'print $1'}", asn, asn),
		fmt.Sprintf("/usr/sbin/birdc show route all table roa_v6 where net.asn=%d | grep -Ev 'BIRD|device1|name|info|kernel1|Table' |grep %d | awk {'print $1'}", asn, asn),
	}
	for _, cmd := range cmds {
		out, err := c.GetOutput(cmd)
		if err != nil {
			return VRPs, err
		}
		for _, line := range strings.Split(out, "\n") {
			split := strings.Split(line, "-")
			_, prefix, err := net.ParseCIDR(split[0])
			if err != nil {
				// Set slice to nil to prevent us sending back a half-filled struct
				VRPs = nil
				return VRPs, err
			}
			max, err := strconv.Atoi(split[1])
			if err != nil {
				VRPs = nil
				return VRPs, err
			}

			VRPs = append(VRPs, VRP{Prefix: prefix, Max: max})
		}
	}

	return VRPs, nil
}
