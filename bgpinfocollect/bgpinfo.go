package main

import (
	"fmt"
	"log"
	"os/exec"
	"strconv"
	"strings"

	pb "github.com/mellowdrifter/bgp_infrastructure/proto/bgpinfo"
)

func main() {
	tot := getTableTotal()
	fmt.Printf("RIB: %s, FIB: %s\n", tot[0], tot[1])

	AS := getSrcAS()
	fmt.Printf("There are %d unique source ASs\n", len(AS))
	fmt.Printf("%q\n", AS)

	TAS := getTransitAS()
	fmt.Printf("There are %d unique transit ASs\n", len(TAS))
	fmt.Printf("%q\n", TAS)

	peers := getPeers()
	fmt.Printf("Peers: %v\n", peers)

	subnets := getSubnets()
	fmt.Printf("There are %d subnets\n", len(subnets))

	largeCommunities := getLargeCommunities()
	fmt.Printf("There are %d large communities\n", largeCommunities)

	roas := getROAs()
	fmt.Printf("Roas: %v\n", roas)

	getPrivateASLeak(AS)
	getPrivateASLeak(TAS)

}

// getOutput is a helper function to run commands and return outputs to other functions.
func getOutput(cmd string) (string, error) {
	fmt.Printf("Running getOutput with cmd %s\n", cmd)
	cmdOut, err := exec.Command("bash", "-c", cmd).Output()
	if err != nil {
		return string(cmdOut), err
	}

	return strings.TrimSuffix(string(cmdOut), "\n"), err
}

// getTableTotal returns the complete RIB and FIB counts.
func getTableTotal() []string {
	cmd := "/usr/sbin/birdc6 show route count | grep routes | awk {'print $3, $6'}"
	v4, err := getOutput(cmd)
	if err != nil {
		log.Fatal(err)
	}
	return strings.Fields(v4)
}

// getPeers returns how many peers are configured, and how many are established.
func getPeers() *pb.PeerCount {
	var peers []uint32
	cmds := []string{
		"/usr/sbin/birdc show protocols | awk {'print $1'} | grep -Ev 'BIRD|device1|name|info|kernel1' | wc -l",
		"/usr/sbin/birdc show protocols | awk {'print $6'} | grep Established | wc -l",
		"/usr/sbin/birdc6 show protocols | awk {'print $1'} | grep -Ev 'BIRD|device1|name|info|kernel1' | wc -l",
		"/usr/sbin/birdc6 show protocols | awk {'print $6'} | grep Established | wc -l",
	}

	for _, cmd := range cmds {
		out, err := getOutput(cmd)
		if err != nil {
			log.Fatal(err)
		}
		p, _ := strconv.Atoi(out)
		peers = append(peers, uint32(p))
	}

	return &pb.PeerCount{
		PeerCount_4: peers[0],
		PeerUp_4:    peers[1],
		PeerCount_6: peers[2],
		PeerUp_6:    peers[3],
	}

}

// getSrcAS returns a unique slice of all source ASs seen.
func getSrcAS() []string {
	cmd := "/usr/sbin/birdc6 show route primary | awk '{print $NF}' | tr -d '[]ASie?' | sed -n '1!p'"
	v4, err := getOutput(cmd)
	if err != nil {
		log.Fatal(err)
	}

	return setListOfStrings(strings.Fields(v4))

}

// getTransitAS returns a unique slice of all ASNs providing transit.
func getTransitAS() []string {
	cmd := "/usr/sbin/birdc show route all primary | grep BGP.as_path | awk '{$1=$2=$NF=\"\"; print}'"
	v4, err := getOutput(cmd)
	if err != nil {
		log.Fatal(err)
	}
	return setListOfStrings(strings.Fields(v4))
}

// getSubnets returns the total amount of each subnet mask.
func getSubnets() []string {
	v6 := make(map[string]int)
	cmd := "/usr/sbin/birdc6 show route primary | awk {'print $1'}"
	subnets, err := getOutput(cmd)
	if err != nil {
		log.Fatal(err)
	}
	for _, s := range strings.Fields(subnets)[1:] {
		mask := strings.Split(s, "::/")[1]
		v6[mask]++
	}
	fmt.Printf("%v\n", v6)

	v4 := make(map[string]int)
	cmd2 := "/usr/sbin/birdc show route primary | awk {'print $1'}"
	subnets2, err := getOutput(cmd2)
	if err != nil {
		log.Fatal(err)
	}
	for _, s := range strings.Fields(subnets2)[1:] {
		mask := strings.Split(s, "/")[1]
		v4[mask]++
	}
	fmt.Printf("%v\n", v4)

	return strings.Fields(subnets)[1:]
}

// getLargeCommunities finds the amount of prefixes that have large communities (RFC8092)
// TODO: I'm not sure this command is right
func getLargeCommunities() *pb.LargeCommunity {
	var comm []uint32
	cmds := []string{
		"/usr/sbin/birdc 'show route primary where bgp_large_community ~ [(*,*,*)]' | sed -n '1!p' | wc -l",
		"/usr/sbin/birdc6 'show route primary where bgp_large_community ~ [(*,*,*)]' | sed -n '1!p' | wc -l",
	}

	for _, cmd := range cmds {
		out, err := getOutput(cmd)
		if err != nil {
			log.Fatal(err)
		}
		c, _ := strconv.Atoi(out)
		comm = append(comm, uint32(c))
	}

	return &pb.LargeCommunity{
		C4: comm[0],
		C6: comm[1],
	}
}

// getROAs returns the amount of RPKI ROAs in VALID, INVALID, and UNKNOWN status.
func getROAs() *pb.Roas {
	var roas []uint32
	cmds := []string{
		"/usr/sbin/birdc 'show route primary where roa_check(roa_table, net, bgp_path.last) = ROA_VALID' | wc -l",
		"/usr/sbin/birdc 'show route primary where roa_check(roa_table, net, bgp_path.last) = ROA_INVALID' | wc -l",
		"/usr/sbin/birdc 'show route primary where roa_check(roa_table, net, bgp_path.last) = ROA_UNKNOWN' | wc -l",
		"/usr/sbin/birdc6 'show route primary where roa_check(roa_table, net, bgp_path.last) = ROA_VALID' | wc -l",
		"/usr/sbin/birdc6 'show route primary where roa_check(roa_table, net, bgp_path.last) = ROA_INVALID' | wc -l",
		"/usr/sbin/birdc6 'show route primary where roa_check(roa_table, net, bgp_path.last) = ROA_UNKNOWN' | wc -l",
	}

	for _, cmd := range cmds {
		out, err := getOutput(cmd)
		if err != nil {
			log.Fatal(err)
		}
		r, _ := strconv.Atoi(out)
		roas = append(roas, uint32(r))
	}
	return &pb.Roas{
		V4Valid:   roas[0],
		V4Invalid: roas[1],
		V4Unknown: roas[2],
		V6Valid:   roas[0],
		V6Invalid: roas[1],
		V6Unknown: roas[2],
	}

}

// getPrivateASLeak returns how many private ASNs are in the AS-Path
func getPrivateASLeak(ASNs []string) {
	for _, ASN := range ASNs {
		num, _ := strconv.Atoi(ASN)
		switch {
		case num == 0:
			fmt.Println("AS is zero")
		case num == 65535:
			fmt.Println("AS is 65535")
		case num == 4294967295:
			fmt.Println("AS is 4294967295")
		case num > 64511 && num < 65535:
			fmt.Printf("16bit private ASN in use: %d\n", num)
		case num > 4199999999 && num < 4294967295:
			fmt.Println("32bit private ASN in use")
		}
	}

}

// setListOfStrings returns a slice of strings with no duplicates.
// Go has no built-in set function, so doing it here
func setListOfStrings(input []string) []string {
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
