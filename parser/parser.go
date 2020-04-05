package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/golang/protobuf/proto"
	cli "github.com/mellowdrifter/bgp_infrastructure/clidecode"
	c "github.com/mellowdrifter/bgp_infrastructure/common"
	pb "github.com/mellowdrifter/bgp_infrastructure/proto/bgpsql"
	"google.golang.org/grpc"
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
	server := cf.Section("grpc").Key("server").String()
	port := cf.Section("grpc").Key("port").String()

	// Set up log file
	f, err := os.OpenFile(logfile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatalf("failed to open logfile: %v\n", err)
	}
	defer f.Close()
	log.SetOutput(f)

	// TODO: For now daemon is always bird2, but will put the option in to choose others
	var router cli.Bird2Conn

	current := &pb.Values{
		Time:           uint64(time.Now().Unix()),
		PrefixCount:    getTableTotal(router),
		Peers:          getPeers(router),
		AsCount:        getAS(router),
		Masks:          getMasks(),
		LargeCommunity: getLargeCommunities(),
		Roas:           getROAs(router),
	}

	log.Printf("%v\n", current)
	fmt.Println(proto.MarshalTextString(current))

	// gRPC dial and send data
	conn, err := grpc.Dial(fmt.Sprintf("%s:%s", server, port), grpc.WithInsecure())
	if err != nil {
		log.Fatalf("Unable to dial gRPC server: %s", err)
	}
	defer conn.Close()
	c := pb.NewBgpInfoClient(conn)

	resp, err := c.AddLatest(context.Background(), current)
	if err != nil {
		log.Fatalf("Unable to send proto: %s", err)
	}

	fmt.Println(proto.MarshalTextString(resp))

}

// getTableTotal returns the complete RIB and FIB counts.
func getTableTotal(d cli.Decoder) *pb.PrefixCount {
	defer c.TimeFunction(time.Now(), "getTableTotal")

	tot, err := d.GetBGPTotal()
	if err != nil {
		log.Println(err)
	}

	return &pb.PrefixCount{
		Total_4:  tot.V4Rib,
		Active_4: tot.V4Fib,
		Total_6:  tot.V6Rib,
		Active_6: tot.V6Fib,
	}
}

// getPeers returns how many peers are configured, and how many are established.
func getPeers(d cli.Decoder) *pb.PeerCount {
	defer c.TimeFunction(time.Now(), "getPeers")

	peers, err := d.GetPeers()
	if err != nil {
		log.Println(err)
	}

	return &pb.PeerCount{
		PeerCount_4: peers.V4c,
		PeerUp_4:    peers.V4e,
		PeerCount_6: peers.V6c,
		PeerUp_6:    peers.V6e,
	}

}

// getAS returns a unique slice of all source ASs seen.
func getAS(d cli.Decoder) *pb.AsCount {
	defer c.TimeFunction(time.Now(), "getAS")

	as, err := d.GetTotalSourceASNs()
	if err != nil {
		log.Println(err)
	}

	return &pb.AsCount{
		As4:     as.As4,
		As6:     as.As6,
		As10:    as.As10,
		As4Only: as.As4Only,
		As6Only: as.As6Only,
		AsBoth:  as.AsBoth,
	}

}

// getTransitAS returns a unique slice of all ASNs providing transit.
// TODO: Do something with this! Also both ipv4 and ipv6!
func getTransitAS() []string {
	defer c.TimeFunction(time.Now(), "getTransitAS")
	cmd := "/usr/sbin/birdc show route all primary | grep BGP.as_path | awk '{$1=$2=$NF=\"\"; print}'"
	v4, err := c.GetOutput(cmd)
	if err != nil {
		log.Fatal(err)
	}
	return c.SetListOfStrings(strings.Fields(v4))
}

// getMasks returns the total amount of each subnet mask.
func getMasks() *pb.Masks {
	defer c.TimeFunction(time.Now(), "getMasks")
	v6 := make(map[string]uint32)
	cmd := "/usr/sbin/birdc show route primary table master6 | awk {'print $1'} | sed -e '1,2d'"
	subnets, err := c.GetOutput(cmd)
	if err != nil {
		log.Fatal(err)
	}
	for _, s := range strings.Fields(subnets) {
		mask := strings.Split(s, "::/")[1]
		v6[mask]++
	}

	v4 := make(map[string]uint32)
	cmd2 := "/usr/sbin/birdc show route primary table master4 | awk {'print $1'} | sed -e '1,2d'"
	subnets2, err := c.GetOutput(cmd2)
	if err != nil {
		log.Fatal(err)
	}
	for _, s := range strings.Fields(subnets2) {
		mask := strings.Split(s, "/")[1]
		v4[mask]++
	}

	// Pack map into proto
	var masks pb.Masks
	masks.V4_08 = v4["8"]
	masks.V4_09 = v4["9"]
	masks.V4_10 = v4["10"]
	masks.V4_10 = v4["10"]
	masks.V4_11 = v4["11"]
	masks.V4_12 = v4["12"]
	masks.V4_13 = v4["13"]
	masks.V4_14 = v4["14"]
	masks.V4_15 = v4["15"]
	masks.V4_16 = v4["16"]
	masks.V4_17 = v4["17"]
	masks.V4_18 = v4["18"]
	masks.V4_19 = v4["19"]
	masks.V4_20 = v4["20"]
	masks.V4_21 = v4["21"]
	masks.V4_22 = v4["22"]
	masks.V4_23 = v4["23"]
	masks.V4_24 = v4["24"]
	masks.V6_08 = v6["8"]
	masks.V6_09 = v6["9"]
	masks.V6_10 = v6["10"]
	masks.V6_10 = v6["10"]
	masks.V6_11 = v6["11"]
	masks.V6_12 = v6["12"]
	masks.V6_13 = v6["13"]
	masks.V6_14 = v6["14"]
	masks.V6_15 = v6["15"]
	masks.V6_16 = v6["16"]
	masks.V6_17 = v6["17"]
	masks.V6_18 = v6["18"]
	masks.V6_19 = v6["19"]
	masks.V6_20 = v6["20"]
	masks.V6_21 = v6["21"]
	masks.V6_22 = v6["22"]
	masks.V6_23 = v6["23"]
	masks.V6_24 = v6["24"]
	masks.V6_25 = v6["25"]
	masks.V6_26 = v6["26"]
	masks.V6_27 = v6["27"]
	masks.V6_28 = v6["28"]
	masks.V6_29 = v6["29"]
	masks.V6_30 = v6["30"]
	masks.V6_31 = v6["31"]
	masks.V6_32 = v6["32"]
	masks.V6_33 = v6["33"]
	masks.V6_34 = v6["34"]
	masks.V6_35 = v6["35"]
	masks.V6_36 = v6["36"]
	masks.V6_37 = v6["37"]
	masks.V6_38 = v6["38"]
	masks.V6_39 = v6["39"]
	masks.V6_40 = v6["40"]
	masks.V6_41 = v6["41"]
	masks.V6_42 = v6["42"]
	masks.V6_43 = v6["43"]
	masks.V6_44 = v6["44"]
	masks.V6_45 = v6["45"]
	masks.V6_46 = v6["46"]
	masks.V6_47 = v6["47"]
	masks.V6_48 = v6["48"]

	return &masks

}

// getLargeCommunities finds the amount of prefixes that have large communities (RFC8092)
// TODO: I'm not sure this command is right
func getLargeCommunities() *pb.LargeCommunity {
	defer c.TimeFunction(time.Now(), "getLargeCommunities")
	var comm []uint32
	cmds := []string{
		"/usr/sbin/birdc 'show route primary table master4 where bgp_large_community ~ [(*,*,*)]' | sed -e '1,2d' | wc -l",
		"/usr/sbin/birdc 'show route primary table master6 where bgp_large_community ~ [(*,*,*)]' | sed -e '1,2d' | wc -l",
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
func getROAs(d cli.Decoder) *pb.Roas {
	defer c.TimeFunction(time.Now(), "getROAs")

	r, err := d.GetROAs()
	if err != nil {
		log.Println(err)
	}

	return &pb.Roas{
		V4Valid:   r.V4v,
		V4Invalid: r.V4i,
		V4Unknown: r.V4u,
		V6Valid:   r.V6v,
		V6Invalid: r.V6i,
		V6Unknown: r.V6u,
	}

}

// getPrivateASLeak returns how many private ASNs are in the AS-Path
func getPrivateASLeak(ASNs []string) {
	defer c.TimeFunction(time.Now(), "getPrivateASLeak")
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
