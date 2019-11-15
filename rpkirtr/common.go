package main

import (
	"log"
	"net"
	"strconv"
)

// stringToInt does inline convertions and logs errors, instead of panicing.
func stringToInt(s string) int {
	n, err := strconv.Atoi(s)
	if err != nil {
		log.Printf("Unable to convert %s to int", s)
		return 0
	}

	return n
}

// The Cloudflare JSON prepends AS to all numbers. Need to remove it here.
func asnToInt(a string) int {
	n, err := strconv.Atoi(a[2:])
	if err != nil {
		log.Printf("Unable to convert ASN %s to int", a)
		return 0
	}

	return n
}

// ipv4ToByte converts an IPv4 address to a [4]byte array
func ipv4ToByte(ip net.IP) [4]byte {
	var b [4]byte
	for i := range ip {
		b[i] = ip[i]
	}
	return b
}

// ipv6ToByte converts an IPv6 address to a [16]byte array
func ipv6ToByte(ip net.IP) [16]byte {
	var b [16]byte
	for i := range ip {
		b[i] = ip[i]
	}
	return b
}
