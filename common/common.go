package common

import (
	"fmt"
	"log"
	"net"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// GetOutput is a helper function to run commands and return outputs to other functions.
func GetOutput(cmd string) (string, error) {
	//log.Printf("Running getOutput with cmd %s\n", cmd)
	cmdOut, err := exec.Command("bash", "-c", cmd).Output()
	if err != nil {
		return string(cmdOut), err
	}

	return strings.TrimSuffix(string(cmdOut), "\n"), err
}

// StringToUint32 is a helper function as many times I need to do this conversion.
func StringToUint32(s string) uint32 {
	val, err := strconv.Atoi(s)
	if err != nil {
		log.Fatalf("Can't convert to integer: %s", err)
	}
	return uint32(val)
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

	parsed := net.ParseIP(ip)
	if parsed == nil {
		return nil, fmt.Errorf("Unable to parse IP")
	}

	if !IsPublicIP(parsed) {
		return nil, fmt.Errorf("IP is not public")
	}

	return parsed, nil

}

// IsPublicIP determines if the IPv4 address is public.
// Go 1.13 might have a function that does all this for me.
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
