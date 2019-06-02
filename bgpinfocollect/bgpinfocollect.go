package main

import (
	"fmt"
	"log"
	"os"
	"path"
	"strconv"
	"strings"

	c "github.com/mellowdrifter/bgp_infrastructure/common"
	pb "github.com/mellowdrifter/bgp_infrastructure/proto/bgpinfo"
	"gopkg.in/ini.v1"
)

func main() {
	// load in config
	exe, err := os.Executable()
	if err != nil {
		log.Fatal(err)
	}
	path := fmt.Sprintf("%s/config.ini", path.Dir(exe))
	cf, err := ini.Load(path)
	if err != nil {
		log.Fatalf("failed to read config file: %v\n", err)
	}

	logfile := cf.Section("grpc").Key("logfile").String()
	//server := cf.Section("grpc").Key("server").String()
	//port := cf.Section("grpc").Key("port").String()

	// Set up log file
	f, err := os.OpenFile(logfile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatalf("failed to open logfile: %v\n", err)
	}
	defer f.Close()
	log.SetOutput(f)

	//getPrivateASLeak(as)

	current := &pb.Values{
		PrefixCount:    getTableTotal(),
		Peers:          getPeers(),
		AsCount:        getAS(),
		Masks:          getMasks(),
		LargeCommunity: getLargeCommunities(),
		Roas:           getROAs(),
	}

	log.Printf("%v\n", current)

}

// getTableTotal returns the complete RIB and FIB counts.
func getTableTotal() *pb.PrefixCount {
	var totals [][]string
	cmds := []string{
		"/usr/sbin/birdc show route count | grep routes | awk {'print $3, $6'}",
		"/usr/sbin/birdc6 show route count | grep routes | awk {'print $3, $6'}",
	}

	for _, cmd := range cmds {
		out, err := c.GetOutput(cmd)
		if err != nil {
			log.Fatal(err)
		}
		totals = append(totals, strings.Fields(out))
	}
	return &pb.PrefixCount{
		Total_4:  c.StringToUint32(totals[0][0]),
		Active_4: c.StringToUint32(totals[0][1]),
		Total_6:  c.StringToUint32(totals[1][0]),
		Active_6: c.StringToUint32(totals[1][1]),
	}
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
		out, err := c.GetOutput(cmd)
		if err != nil {
			log.Fatal(err)
		}
		peers = append(peers, c.StringToUint32(out))
	}

	return &pb.PeerCount{
		PeerCount_4: peers[0],
		PeerUp_4:    peers[1],
		PeerCount_6: peers[2],
		PeerUp_6:    peers[3],
	}

}

// getAS returns a unique slice of all source ASs seen.
// TODO: add transit ASs as well and pack into same struct.
func getAS() *pb.AsCount {
	cmd1 := "/usr/sbin/birdc show route primary | awk '{print $NF}' | tr -d '[]ASie?' | sed -n '1!p'"
	cmd2 := "/usr/sbin/birdc6 show route primary | awk '{print $NF}' | tr -d '[]ASie?' | sed -n '1!p'"

	as4, err := c.GetOutput(cmd1)
	if err != nil {
		log.Fatal(err)
	}
	as6, err := c.GetOutput(cmd2)
	if err != nil {
		log.Fatal(err)
	}

	// asNSet is the list of unique source AS numbers in each address family
	as4Set := c.SetListOfStrings(strings.Fields(as4))
	as6Set := c.SetListOfStrings(strings.Fields(as6))

	// as10Set is the total number of unique source AS numbers.
	var as10 []string
	as10 = append(as10, as4Set...)
	as10 = append(as10, as6Set...)
	as10Set := c.SetListOfStrings(as10)

	//TODO: as4_only, as6_only, as_both

	return &pb.AsCount{
		As4:  uint32(len(as4Set)),
		As6:  uint32(len(as6Set)),
		As10: uint32(len(as10Set)),
	}

}

// getTransitAS returns a unique slice of all ASNs providing transit.
// TODO: Do something with this!
func getTransitAS() []string {
	cmd := "/usr/sbin/birdc show route all primary | grep BGP.as_path | awk '{$1=$2=$NF=\"\"; print}'"
	v4, err := c.GetOutput(cmd)
	if err != nil {
		log.Fatal(err)
	}
	return c.SetListOfStrings(strings.Fields(v4))
}

// getMasks returns the total amount of each subnet mask.
func getMasks() *pb.Mask {
	v6 := make(map[string]uint32)
	cmd := "/usr/sbin/birdc6 show route primary | awk {'print $1'}"
	subnets, err := c.GetOutput(cmd)
	if err != nil {
		log.Fatal(err)
	}
	for _, s := range strings.Fields(subnets)[1:] {
		mask := strings.Split(s, "::/")[1]
		v6[mask]++
	}

	v4 := make(map[string]uint32)
	cmd2 := "/usr/sbin/birdc show route primary | awk {'print $1'}"
	subnets2, err := c.GetOutput(cmd2)
	if err != nil {
		log.Fatal(err)
	}
	for _, s := range strings.Fields(subnets2)[1:] {
		mask := strings.Split(s, "/")[1]
		v4[mask]++
	}

	// Pack map into proto
	var v4Masks []*pb.Maskcount
	var v6Masks []*pb.Maskcount

	var i uint32
	for i = 8; i < 25; i++ {
		mc := &pb.Maskcount{
			Mask:  i,
			Count: v4[strconv.Itoa(int(i))],
		}
		v4Masks = append(v4Masks, mc)
		log.Printf("%+v\n", mc)

	}

	for i = 8; i < 49; i++ {
		mc := &pb.Maskcount{
			Mask:  i,
			Count: v6[strconv.Itoa(int(i))],
		}
		v6Masks = append(v6Masks, mc)
		log.Printf("%+v\n", mc)

	}

	return &pb.Mask{
		Ipv4: v4Masks,
		Ipv6: v6Masks,
	}

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
		out, err := c.GetOutput(cmd)
		if err != nil {
			log.Fatal(err)
		}
		comm = append(comm, c.StringToUint32(out))
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
		out, err := c.GetOutput(cmd)
		if err != nil {
			log.Fatal(err)
		}
		roas = append(roas, c.StringToUint32(out))
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
			log.Printf("16bit private ASN in use: %d\n", num)
		case num > 4199999999 && num < 4294967295:
			fmt.Println("32bit private ASN in use")
		}
	}

}
