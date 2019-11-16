package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"regexp"
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

// makeDiff will return a list of ROAs that need to be deleted or updated
// in order for a particular serial version to updated to the latest version.
func makeDiff(new []roa, old []roa, serial uint32) serialDiff {
	var addROA, delROA []roa

	newm := roasToMap(new)
	oldm := roasToMap(old)

	// If ROA is in newMap but not oldMap, we need to add it
	for k, v := range newm {
		_, ok := oldm[k]
		if !ok {
			addROA = append(addROA, v)
		}
	}

	// If ROA is in oldMap but not newMap, we need to delete it.
	for k, v := range oldm {
		_, ok := newm[k]
		if !ok {
			delROA = append(delROA, v)
		}
	}

	// There is only an actual diff is something is added or deleted.
	diff := (len(addROA) > 0 || len(delROA) > 0)

	// The following is for debugging purposes. Will remove eventually once I have test coverage.
	if len(addROA) > 0 {
		log.Printf("New ROAs to be added: %+v\n", addROA)
	}
	if len(delROA) > 0 {
		log.Printf("Old ROAs to be deleted: %+v\n", delROA)
	}
	if !diff {
		log.Println("No diff calculated this run")
	}

	return serialDiff{
		oldSerial: serial,
		newSerial: serial + 1,
		addRoa:    addROA,
		delRoa:    delROA,
		diff:      diff,
	}
}

//roasToMap will convert a slice of ROAs into a map of formatted ROA to a ROA.
func roasToMap(roas []roa) map[string]roa {
	rm := make(map[string]roa, len(roas))
	for _, roa := range roas {
		rm[fmt.Sprintf("%s%d%d%d", roa.Prefix, roa.MinMask, roa.MaxMask, roa.ASN)] = roa

	}
	return rm

}

// readROAs will fetch the latest set of ROAs and add to a local struct
// TODO: For now this is getting data from cloudflare, but eventually I want to get this from
// the RIRs directly.
func readROAs(url string) ([]roa, error) {

	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	f, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	rirs := map[string]rir{
		"Cloudflare - AFRINIC": afrinic,
		"Cloudflare - ARIN":    arin,
		"Cloudflare - APNIC":   apnic,
		"Cloudflare - LACNIC":  lacnic,
		"Cloudflare - RIPE":    ripe,
	}

	var r rpkiResponse
	json.Unmarshal(f, &r)

	// We know how many ROAs we have, so we can add that capacity directly
	roas := make([]roa, 0, len(r.roas.Roas))

	rxp := regexp.MustCompile(`(.*)/(.*)`)

	for _, r := range r.roas.Roas {
		prefix := rxp.FindStringSubmatch(r.Prefix)
		roas = append(roas, roa{
			Prefix:  prefix[1],
			MinMask: uint8(stringToInt(prefix[2])),
			MaxMask: uint8(r.Mask),
			ASN:     uint32(asnToInt(r.ASN)),
			RIR:     rirs[r.RIR],
		})

	}

	return roas, nil

}
