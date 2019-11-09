package main

import (
	"log"
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
