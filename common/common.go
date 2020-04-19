package common

import (
	"fmt"
	"log"
	"net"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	pb "github.com/mellowdrifter/bgp_infrastructure/proto/bgpsql"
)

// BgpStat holds the AS information altogether
type BgpStat struct {
	Time              uint64
	V4Count, V6Count  uint32
	PeersConfigured   uint8
	Peers6Configured  uint8
	PeersUp, Peers6Up uint8
	V4Total, V6Total  uint32
}

// BgpUpdate holds all the information required for an update
type BgpUpdate struct {
	Time                                uint64
	V4Count, V6Count                    uint32
	V4Total, V6Total                    uint32
	PeersConfigured                     uint32
	Peers6Configured                    uint32
	PeersUp, Peers6Up                   uint32
	Tweet                               uint32
	As4, As6, As10                      uint32
	As4Only, As6Only                    uint32
	AsBoth                              uint32
	LargeC4, LargeC6                    uint32
	MemTable, MemTotal                  string
	MemProto, MemAttr                   string
	MemTable6, MemTotal6                string
	MemProto6, MemAttr6                 string
	Roavalid4, Roainvalid4, Roaunknown4 uint32
	Roavalid6, Roainvalid6, Roaunknown6 uint32
	V4_23, V4_22, V4_21, V4_20, V4_19   uint32
	V4_18, V4_17, V4_16, V4_15, V4_14   uint32
	V4_13, V4_12, V4_11, V4_10, V4_09   uint32
	V4_08, V6_48, V6_47, V6_46, V6_45   uint32
	V6_44, V6_43, V6_42, V6_41, V6_40   uint32
	V6_39, V6_38, V6_37, V6_36, V6_35   uint32
	V6_34, V6_33, V6_32, V6_31, V6_30   uint32
	V6_29, V6_28, V6_27, V6_26, V6_25   uint32
	V6_24, V6_23, V6_22, V6_21, V6_20   uint32
	V6_19, V6_18, V6_17, V6_16, V6_15   uint32
	V6_14, V6_13, V6_12, V6_11, V6_10   uint32
	V6_09, V6_08, V4_24                 uint32
}

// GetOutput is a helper function to run commands and return outputs to other functions.
func GetOutput(cmd string) (string, error) {
	log.Printf("Running getOutput with cmd %s\n", cmd)
	cmdOut, err := exec.Command("bash", "-c", cmd).Output()
	if err != nil {
		return string(cmdOut), err
	}

	return strings.TrimSuffix(string(cmdOut), "\n"), err
}

// StringToUint32 is a helper function as many times I need to do this conversion.
func StringToUint32(s string) uint32 {
	reg := regexp.MustCompile("[^0-9]+")
	c := reg.ReplaceAllString(s, "")
	val, err := strconv.Atoi(c)
	if err != nil {
		log.Fatalf("Can't convert to integer: %s", err)
	}
	return uint32(val)
}

// Uint32ToString is a helper function as many times I need to do this conversion.
func Uint32ToString(u uint32) string {
	val := strconv.Itoa(int(u))
	return val
}

// SetListOfStrings returns a slice of strings with no duplicates.
// Go has no built-in set function, so doing it here
func SetListOfStrings(input []string) []string {
	u := make([]string, 0, len(input))
	m := make(map[string]bool)
	for _, val := range input {
		if _, ok := m[val]; !ok {
			m[val] = true
			u = append(u, val)
		}
	}
	return u

}

// InFirstButNotSecond returns the second slice subtracted from the first.
// Go has no built-in function to do this.
func InFirstButNotSecond(first, second []string) []string {
	var third []string
	iterMap := make(map[string]bool)
	for _, element := range second {
		iterMap[element] = true
	}
	for _, element := range first {
		if _, ok := iterMap[element]; !ok {
			third = append(third, element)
		}
	}
	return third
}

// Intersection returns a slice that is an intersection of two slices.
// Go has no built-in function to do this.
func Intersection(first, second []string) []string {
	var third []string
	iterMap := make(map[string]bool)
	for _, element := range second {
		iterMap[element] = true
	}
	for _, element := range first {
		if _, ok := iterMap[element]; ok {
			third = append(third, element)
		}
	}
	return third
}

// TimeFunction logs total time to execute a function.
func TimeFunction(start time.Time, name string) {
	log.Printf("%s took %s\n", name, time.Since(start))
}

// ValidateIP ensures the IP address is valid.
// non Public IPs are not valid.
func ValidateIP(ip string) (net.IP, error) {
	log.Printf("Running validateIP")

	var parsed net.IP

	if strings.Contains(ip, "/") {
		parsed, _, _ = net.ParseCIDR(ip)
	} else {
		parsed = net.ParseIP(ip)
	}
	if parsed == nil {
		return nil, fmt.Errorf("Unable to parse IP: %s", ip)
	}

	if !IsPublicIP(parsed) {
		return nil, fmt.Errorf("IP is not public")
	}

	return parsed, nil

}

// IsPublicIP determines if the IPv4 address is public.
// Go 1.13 might have a function that does all this for me.
// TODO: Check functions in latest Go
func IsPublicIP(ip net.IP) bool {
	if ip4 := ip.To4(); ip4 != nil {
		return IsPublicIPv4(ip4)
	}
	return IsPublicIPv6(ip)

}

// IsPublicIPv4 checks if the IPv4 address is a valid public address.
func IsPublicIPv4(ip net.IP) bool {
	if !ip.IsGlobalUnicast() {
		return false
	}

	switch {
	// rfc1918
	case ip[0] == 10:
		return false
	// rfc1918
	case ip[0] == 172 && ip[1] >= 16 && ip[1] <= 31:
		return false
	// rfc1918
	case ip[0] == 192 && ip[1] == 168:
		return false
	// rfc6598
	case ip[0] == 100 && ip[1] >= 64 && ip[1] <= 127:
		return false
	// rfc3927
	case ip[0] == 169 && ip[1] == 254:
		return false
	// rfc6980 && rfc5737
	case ip[0] == 192 && ip[1] == 0 && (ip[2] == 0 || ip[2] == 2):
		return false
	// rfc5737
	case ip[0] == 198 && ip[1] == 51 && ip[2] == 100:
		return false
	// rfc5737
	case ip[0] == 203 && ip[1] == 0 && ip[2] == 113:
		return false
	}

	return true

}

// IsPublicIPv6 checks if the IPv6 address is a valid public address.
func IsPublicIPv6(ip net.IP) bool {
	if !ip.IsGlobalUnicast() {
		return false
	}
	// For now, just ensure the route belongs to 2000::/3
	return ip[0] >= 32 && ip[0] <= 63

}

// ValidateIPNet ensures the IP address and mask is valid. We only care about public IPs.
func ValidateIPNet(ip string, mask uint32) (*net.IPNet, error) {
	log.Printf("Running validateIPNet")

	parsed, net, err := net.ParseCIDR(fmt.Sprintf("%s/%d", ip, mask))
	if err != nil {
		return nil, fmt.Errorf("Unable to parse IP and subnet")
	}

	if !IsPublicIP(parsed) {
		return nil, fmt.Errorf("IP is not public")
	}

	return net, nil

}

// ValidateASN ensures the AS number is a public, non-documentation, non-reserved AS.
func ValidateASN(asn uint32) bool {
	switch {
	case asn == 0:
		return false
	case asn == 23456:
		return false
	case asn >= 64000 && asn <= 131071:
		return false
	case asn >= 4200000000:
		return false
	}
	return true
}

// ASPlainToASDot will convert an ASPLAIN AS number to a ASDOT representation.
func ASPlainToASDot(asn uint32) string {
	if asn < 1 || asn > 4294967295 {
		return ""
	}
	dot1 := asn / 65536
	dot2 := asn % 65536

	if dot1 == 0 {
		return fmt.Sprintf("%d", dot2)
	}

	return fmt.Sprintf("%d.%d", int(dot1), dot2)

}

// ASDotToASPlain will convert an ASDOT AS number to a ASPLAIN representation.
func ASDotToASPlain(asn string) uint32 {
	asStrings := strings.Split(asn, ".")
	if len(asStrings) > 2 {
		return 0
	}

	// Not using asdot+, so a single field is legitimate.
	if len(asStrings) == 1 {
		asPlain := StringToUint32(asStrings[0])
		if asPlain < 1 || asPlain > 65535 {
			return 0
		}
		return asPlain

	}

	// Else it's regular asdot notation
	dot1 := StringToUint32(asStrings[0])
	dot2 := StringToUint32(asStrings[1])

	if dot1 < 0 || dot1 > 65535 || dot2 < 0 || dot2 > 65535 {
		return 0
	}

	return dot1*65536 + dot2

}

// ProtoToStruct converts a bgpinfo.Values proto to a bgpUpdate struct.
func ProtoToStruct(v *pb.Values) *BgpUpdate {
	// While we receive this information in a protobuf, the
	// format needs to be adjusted a bit to insert into the
	// database later.
	as := v.GetAsCount()
	mask := v.GetMasks()
	p := v.GetPrefixCount()
	roa := v.GetRoas()
	update := &BgpUpdate{
		Time:             v.GetTime(),
		V4Count:          p.GetActive_4(),
		V6Count:          p.GetActive_6(),
		V4Total:          p.GetTotal_4(),
		V6Total:          p.GetTotal_6(),
		As4:              as.GetAs4(),
		As6:              as.GetAs6(),
		As10:             as.GetAs10(),
		As4Only:          as.GetAs4Only(),
		As6Only:          as.GetAs6Only(),
		AsBoth:           as.GetAsBoth(),
		LargeC4:          v.GetLargeCommunity().GetC4(),
		LargeC6:          v.GetLargeCommunity().GetC6(),
		PeersConfigured:  v.GetPeers().GetPeerCount_4(),
		PeersUp:          v.GetPeers().GetPeerUp_4(),
		Peers6Configured: v.GetPeers().GetPeerCount_6(),
		Peers6Up:         v.GetPeers().GetPeerUp_6(),
		Roavalid4:        roa.GetV4Valid(),
		Roainvalid4:      roa.GetV4Invalid(),
		Roaunknown4:      roa.GetV4Unknown(),
		Roavalid6:        roa.GetV6Valid(),
		Roainvalid6:      roa.GetV6Invalid(),
		Roaunknown6:      roa.GetV6Unknown(),
		V4_08:            mask.GetV4_08(),
		V4_09:            mask.GetV4_09(),
		V4_10:            mask.GetV4_10(),
		V4_11:            mask.GetV4_11(),
		V4_12:            mask.GetV4_12(),
		V4_13:            mask.GetV4_13(),
		V4_14:            mask.GetV4_14(),
		V4_15:            mask.GetV4_15(),
		V4_16:            mask.GetV4_16(),
		V4_17:            mask.GetV4_17(),
		V4_18:            mask.GetV4_18(),
		V4_19:            mask.GetV4_19(),
		V4_20:            mask.GetV4_20(),
		V4_21:            mask.GetV4_21(),
		V4_22:            mask.GetV4_22(),
		V4_23:            mask.GetV4_23(),
		V4_24:            mask.GetV4_24(),
		V6_08:            mask.GetV6_08(),
		V6_09:            mask.GetV6_09(),
		V6_10:            mask.GetV6_10(),
		V6_11:            mask.GetV6_11(),
		V6_12:            mask.GetV6_12(),
		V6_13:            mask.GetV6_13(),
		V6_14:            mask.GetV6_14(),
		V6_15:            mask.GetV6_15(),
		V6_16:            mask.GetV6_16(),
		V6_17:            mask.GetV6_17(),
		V6_18:            mask.GetV6_18(),
		V6_19:            mask.GetV6_19(),
		V6_20:            mask.GetV6_20(),
		V6_21:            mask.GetV6_21(),
		V6_22:            mask.GetV6_22(),
		V6_23:            mask.GetV6_23(),
		V6_24:            mask.GetV6_24(),
		V6_25:            mask.GetV6_25(),
		V6_26:            mask.GetV6_26(),
		V6_27:            mask.GetV6_27(),
		V6_28:            mask.GetV6_28(),
		V6_29:            mask.GetV6_29(),
		V6_30:            mask.GetV6_30(),
		V6_31:            mask.GetV6_31(),
		V6_32:            mask.GetV6_32(),
		V6_33:            mask.GetV6_33(),
		V6_34:            mask.GetV6_34(),
		V6_35:            mask.GetV6_35(),
		V6_36:            mask.GetV6_36(),
		V6_37:            mask.GetV6_37(),
		V6_38:            mask.GetV6_38(),
		V6_39:            mask.GetV6_39(),
		V6_40:            mask.GetV6_40(),
		V6_41:            mask.GetV6_41(),
		V6_42:            mask.GetV6_42(),
		V6_43:            mask.GetV6_43(),
		V6_44:            mask.GetV6_44(),
		V6_45:            mask.GetV6_45(),
		V6_46:            mask.GetV6_46(),
		V6_47:            mask.GetV6_47(),
		V6_48:            mask.GetV6_48(),
	}

	return update
}

// StructToProto converts a BgpUpdate to a bgpinfo.Values proto.
func StructToProto(b *BgpUpdate) *pb.Values {

	return &pb.Values{
		Time: b.Time,
		PrefixCount: &pb.PrefixCount{
			Active_4: b.V4Count,
			Active_6: b.V6Count,
			Total_4:  b.V4Total,
			Total_6:  b.V6Total,
		},
		Peers: &pb.PeerCount{
			PeerCount_4: b.PeersConfigured,
			PeerCount_6: b.Peers6Configured,
			PeerUp_4:    b.PeersUp,
			PeerUp_6:    b.Peers6Up,
		},
		AsCount: &pb.AsCount{
			As4:     b.As4,
			As6:     b.As6,
			As10:    b.As10,
			As4Only: b.As4Only,
			As6Only: b.As6Only,
			AsBoth:  b.AsBoth,
		},
		Masks: &pb.Masks{
			V4_08: b.V4_08,
			V4_09: b.V4_09,
			V4_10: b.V4_10,
			V4_11: b.V4_11,
			V4_12: b.V4_12,
			V4_13: b.V4_13,
			V4_14: b.V4_14,
			V4_15: b.V4_15,
			V4_16: b.V4_16,
			V4_17: b.V4_17,
			V4_18: b.V4_18,
			V4_19: b.V4_19,
			V4_20: b.V4_20,
			V4_21: b.V4_21,
			V4_22: b.V4_22,
			V4_23: b.V4_23,
			V4_24: b.V4_24,
			V6_08: b.V6_08,
			V6_09: b.V6_09,
			V6_10: b.V6_10,
			V6_11: b.V6_11,
			V6_12: b.V6_12,
			V6_13: b.V6_13,
			V6_14: b.V6_14,
			V6_15: b.V6_15,
			V6_16: b.V6_16,
			V6_17: b.V6_17,
			V6_18: b.V6_18,
			V6_19: b.V6_19,
			V6_20: b.V6_20,
			V6_21: b.V6_21,
			V6_22: b.V6_22,
			V6_23: b.V6_23,
			V6_24: b.V6_24,
			V6_25: b.V6_25,
			V6_26: b.V6_26,
			V6_27: b.V6_27,
			V6_28: b.V6_28,
			V6_29: b.V6_29,
			V6_30: b.V6_30,
			V6_31: b.V6_31,
			V6_32: b.V6_32,
			V6_33: b.V6_33,
			V6_34: b.V6_34,
			V6_35: b.V6_35,
			V6_36: b.V6_36,
			V6_37: b.V6_37,
			V6_38: b.V6_38,
			V6_39: b.V6_39,
			V6_40: b.V6_40,
			V6_41: b.V6_41,
			V6_42: b.V6_42,
			V6_43: b.V6_43,
			V6_44: b.V6_44,
			V6_45: b.V6_45,
			V6_46: b.V6_46,
			V6_47: b.V6_47,
			V6_48: b.V6_48,
		},
		LargeCommunity: &pb.LargeCommunity{
			C4: b.LargeC4,
			C6: b.LargeC6,
		},
		Roas: &pb.Roas{
			V4Valid:   b.Roavalid4,
			V4Invalid: b.Roainvalid4,
			V4Unknown: b.Roaunknown4,
			V6Valid:   b.Roavalid6,
			V6Invalid: b.Roainvalid6,
			V6Unknown: b.Roaunknown6,
		},
	}
}
