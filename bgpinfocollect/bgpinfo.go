package main

import (
	"fmt"
	"log"
	"os/exec"
	"strconv"
	"strings"
)

func main() {
	tot := getTableTotal()
	fmt.Printf("RIB: %s, FIB: %s\n", tot[0], tot[1])

	AS := getSrcAS()
	fmt.Printf("There are %d unique ASs\n", len(AS))

	peers, state := getPeers()
	fmt.Printf("I have %d peers and %d of them are Established\n", peers, state)

	subnets := getSubnets()
	fmt.Printf("There are %d subnets\n", len(subnets))

	largeCommunities := getLargeCommunities()
	fmt.Printf("There are %d large communities\n", largeCommunities)

}

// getOutput is a helper functino to run commands and return outputs to other functions.
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
func getPeers() (int, int) {
	cmd1 := "/usr/sbin/birdc show protocols | awk {'print $1'} | grep -Ev 'BIRD|device1|name|info|kernel1' | wc -l"
	peers, err := getOutput(cmd1)
	if err != nil {
		log.Fatal(err)
	}
	p, _ := strconv.Atoi(peers)

	cmd2 := "/usr/sbin/birdc show protocols | awk {'print $6'} | grep Established | wc -l"
	state, err := getOutput(cmd2)
	if err != nil {
		log.Fatal(err)
	}
	s, _ := strconv.Atoi(state)

	return p, s

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
func getLargeCommunities() int {
	cmd := "/usr/sbin/birdc 'show route primary where bgp_large_community ~ [(*,*,*)]' | sed -n '1!p' | wc -l"
	v4, err := getOutput(cmd)
	if err != nil {
		log.Fatal(err)
	}

	c, _ := strconv.Atoi(v4)
	return c
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
