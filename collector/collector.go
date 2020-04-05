package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path"
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
		Masks:          getMasks(router),
		LargeCommunity: getLargeCommunities(router),
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

// getMasks returns the total amount of each subnet mask.
func getMasks(d cli.Decoder) *pb.Masks {
	defer c.TimeFunction(time.Now(), "getMasks")

	m, err := d.GetMasks()
	if err != nil {
		log.Println(err)
	}

	if len(m) < 2 {
		log.Panicf("Masks slice expected to be size 2, actual size is %d", len(m))
	}

	// Pack map into proto
	// TODO: Is there not a nicer way of doing this?
	var masks pb.Masks
	masks.V4_08 = m[0]["8"]
	masks.V4_09 = m[0]["9"]
	masks.V4_10 = m[0]["10"]
	masks.V4_10 = m[0]["10"]
	masks.V4_11 = m[0]["11"]
	masks.V4_12 = m[0]["12"]
	masks.V4_13 = m[0]["13"]
	masks.V4_14 = m[0]["14"]
	masks.V4_15 = m[0]["15"]
	masks.V4_16 = m[0]["16"]
	masks.V4_17 = m[0]["17"]
	masks.V4_18 = m[0]["18"]
	masks.V4_19 = m[0]["19"]
	masks.V4_20 = m[0]["20"]
	masks.V4_21 = m[0]["21"]
	masks.V4_22 = m[0]["22"]
	masks.V4_23 = m[0]["23"]
	masks.V4_24 = m[0]["24"]
	masks.V6_08 = m[1]["8"]
	masks.V6_09 = m[1]["9"]
	masks.V6_10 = m[1]["10"]
	masks.V6_10 = m[1]["10"]
	masks.V6_11 = m[1]["11"]
	masks.V6_12 = m[1]["12"]
	masks.V6_13 = m[1]["13"]
	masks.V6_14 = m[1]["14"]
	masks.V6_15 = m[1]["15"]
	masks.V6_16 = m[1]["16"]
	masks.V6_17 = m[1]["17"]
	masks.V6_18 = m[1]["18"]
	masks.V6_19 = m[1]["19"]
	masks.V6_20 = m[1]["20"]
	masks.V6_21 = m[1]["21"]
	masks.V6_22 = m[1]["22"]
	masks.V6_23 = m[1]["23"]
	masks.V6_24 = m[1]["24"]
	masks.V6_25 = m[1]["25"]
	masks.V6_26 = m[1]["26"]
	masks.V6_27 = m[1]["27"]
	masks.V6_28 = m[1]["28"]
	masks.V6_29 = m[1]["29"]
	masks.V6_30 = m[1]["30"]
	masks.V6_31 = m[1]["31"]
	masks.V6_32 = m[1]["32"]
	masks.V6_33 = m[1]["33"]
	masks.V6_34 = m[1]["34"]
	masks.V6_35 = m[1]["35"]
	masks.V6_36 = m[1]["36"]
	masks.V6_37 = m[1]["37"]
	masks.V6_38 = m[1]["38"]
	masks.V6_39 = m[1]["39"]
	masks.V6_40 = m[1]["40"]
	masks.V6_41 = m[1]["41"]
	masks.V6_42 = m[1]["42"]
	masks.V6_43 = m[1]["43"]
	masks.V6_44 = m[1]["44"]
	masks.V6_45 = m[1]["45"]
	masks.V6_46 = m[1]["46"]
	masks.V6_47 = m[1]["47"]
	masks.V6_48 = m[1]["48"]

	return &masks

}

// getLargeCommunities finds the amount of prefixes that have large communities (RFC8092)
func getLargeCommunities(d cli.Decoder) *pb.LargeCommunity {
	defer c.TimeFunction(time.Now(), "getLargeCommunities")

	l, err := d.GetLargeCommunities()
	if err != nil {
		log.Println(err)
	}

	return &pb.LargeCommunity{
		C4: l.V4,
		C6: l.V6,
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
