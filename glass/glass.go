package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"path"
	"strconv"
	"strings"

	com "github.com/mellowdrifter/bgp_infrastructure/common"
	bpb "github.com/mellowdrifter/bgp_infrastructure/proto/bgpinfo"
	pb "github.com/mellowdrifter/bgp_infrastructure/proto/glass"
	"google.golang.org/grpc"
	"gopkg.in/ini.v1"
)

type server struct{}

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

	logfile := cf.Section("log").Key("logfile").String()

	// Set up log file
	f, err := os.OpenFile(logfile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatalf("failed to open logfile: %v\n", err)
	}
	defer f.Close()
	log.SetOutput(f)

	// set up gRPC server
	log.Printf("Listening on port %d\n", 7181)
	lis, err := net.Listen("tcp", ":7181")
	if err != nil {
		log.Fatalf("Failed to bind: %v", err)
	}
	grpcServer := grpc.NewServer()
	pb.RegisterLookingGlassServer(grpcServer, &server{})

	grpcServer.Serve(lis)

}

// Origin will return the origin ASN for the active route.
func (s *server) Origin(ctx context.Context, r *pb.OriginRequest) (*pb.OriginResponse, error) {
	log.Printf("Running Origin")

	ip, err := com.ValidateIP(r.GetIpAddress().GetAddress())
	if err != nil {
		log.Printf("Error: %v", err)
		return nil, err
	}

	asn, exists, err := getOriginFromDaemon(ip)
	if err != nil {
		log.Printf("Error: %v", err)
		return nil, err
	}
	if !exists {
		return &pb.OriginResponse{}, nil
	}

	return &pb.OriginResponse{
		OriginAsn: uint32(asn),
		Exists:    exists,
	}, nil

}

func (s *server) Totals(ctx context.Context, e *pb.Empty) (*pb.TotalResponse, error) {
	log.Printf("Running Totals")

	// load in config
	exe, err := os.Executable()
	if err != nil {
		return &pb.TotalResponse{}, errors.New("Unable to load config in Totals")
	}
	path := fmt.Sprintf("%s/config.ini", path.Dir(exe))
	cf, err := ini.Load(path)
	if err != nil {
		return &pb.TotalResponse{}, fmt.Errorf("failed to read config file: %v", err)
	}

	// gRPC dial the grapher
	bgpinfo := cf.Section("bgpinfo").Key("server").String()
	conn, err := grpc.Dial(bgpinfo, grpc.WithInsecure())
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	b := bpb.NewBgpInfoClient(conn)

	totals, err := b.GetPrefixCount(ctx, &bpb.Empty{})
	if err != nil {
		return &pb.TotalResponse{}, err
	}

	return &pb.TotalResponse{
		Active_4: totals.GetActive_4(),
		Active_6: totals.GetActive_6(),
		Time:     totals.GetTime(),
	}, nil

}

// Aspath returns a list of ASNs for an IP address.
func (s *server) Aspath(ctx context.Context, r *pb.AspathRequest) (*pb.AspathResponse, error) {
	log.Printf("Running Aspath")

	ip, err := com.ValidateIP(r.GetIpAddress().GetAddress())
	if err != nil {
		log.Printf("Error: %v", err)
		return nil, err
	}

	asns, sets, exists, err := getASPathFromDaemon(ip)
	if err != nil {
		log.Printf("Error: %v", err)
		return nil, err
	}
	if !exists {
		return &pb.AspathResponse{}, nil
	}

	return &pb.AspathResponse{
		Asn:    asns,
		Set:    sets,
		Exists: exists,
	}, nil
}

// Route returns the primary active RIB entry for the IP passed.
func (s *server) Route(ctx context.Context, r *pb.RouteRequest) (*pb.RouteResponse, error) {
	log.Printf("Running Route")

	ip, err := com.ValidateIP(r.GetIpAddress().GetAddress())
	if err != nil {
		return nil, errors.New("Unable to validate IP")
	}

	ipnet, exists, err := getRouteFromDaemon(ip)
	if err != nil {
		log.Printf("Error: %v", err)
		return nil, err
	}

	if !exists {
		return &pb.RouteResponse{}, nil
	}

	mask, _ := ipnet.Mask.Size()
	ipaddr := &pb.IpAddress{
		Address: ipnet.IP.String(),
		Mask:    uint32(mask),
	}

	return &pb.RouteResponse{
		IpAddress: ipaddr,
		Exists:    exists,
	}, nil
}

// Asname will return the registered name of the ASN. As this isn't in bird directly, will need
// to speak to bgpinfo to get information from the database.
func (s *server) Asname(ctx context.Context, r *pb.AsnameRequest) (*pb.AsnameResponse, error) {
	//return nil, grpc.Errorf(codes.Unimplemented, "RPC not yet implemented")
	log.Printf("Running Asname")

	// load in config
	exe, err := os.Executable()
	if err != nil {
		return &pb.AsnameResponse{}, errors.New("Unable to load config in Asname")
	}
	path := fmt.Sprintf("%s/config.ini", path.Dir(exe))
	cf, err := ini.Load(path)
	if err != nil {
		return &pb.AsnameResponse{}, fmt.Errorf("failed to read config file: %v", err)
	}

	number := bpb.GetAsnameRequest{AsNumber: r.GetAsNumber()}

	// gRPC dial the grapher
	bgpinfo := cf.Section("bgpinfo").Key("server").String()
	conn, err := grpc.Dial(bgpinfo, grpc.WithInsecure())
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	b := bpb.NewBgpInfoClient(conn)

	name, err := b.GetAsname(ctx, &number)
	if err != nil {
		return &pb.AsnameResponse{}, err
	}

	return &pb.AsnameResponse{
		AsName: name.GetAsName(),
		Locale: name.GetAsLocale(),
		Exists: name.Exists,
	}, nil

}

func (s *server) Roa(ctx context.Context, r *pb.RoaRequest) (*pb.RoaResponse, error) {
	log.Printf("Running Roa")

	ip, err := com.ValidateIP(r.GetIpAddress().GetAddress())
	if err != nil {
		log.Printf("Error: %v", err)
		return nil, err
	}

	status, err := getRoaFromDaemon(ip)
	if err != nil {
		log.Printf("Error: %v", err)
		return nil, err
	}

	return status, nil

}

func (s *server) Sourced(ctx context.Context, r *pb.SourceRequest) (*pb.SourceResponse, error) {
	log.Printf("Running Sourced")

	if !com.ValidateASN(r.GetAsNumber()) {
		return nil, errors.New("Invalid AS number")
	}

	subnets, err := getSourcedFromDaemon(r.GetAsNumber())
	if err != nil {
		log.Printf("Error: %v", err)
		return nil, err
	}

	if !subnets.Exists {
		return &pb.SourceResponse{}, nil
	}

	return subnets, nil
}

// getSourcedFromDaemon will get all the IPv4 and IPv6 routes sourced from an ASN.
func getSourcedFromDaemon(as uint32) (*pb.SourceResponse, error) {
	log.Printf("Running getSourcedFromDaemon")

	cmd := fmt.Sprintf("/usr/sbin/birdc6 'show route primary where bgp_path ~ [= * %d =]' | grep -Ev 'BIRD|device1|name|info|kernel1' | awk '{print $1}'", as)
	log.Printf(cmd)
	out, err := com.GetOutput(cmd)
	if err != nil {
		return &pb.SourceResponse{}, err
	}
	var prefixes []*pb.IpAddress
	for _, address := range strings.Fields(out) {
		addrMask := strings.Split(address, "/")
		fmt.Printf("%s\n", addrMask)
		prefixes = append(prefixes, &pb.IpAddress{
			Address: addrMask[0],
			Mask:    com.StringToUint32(addrMask[1]),
		})
	}

	v6Count := len(prefixes)

	cmd = fmt.Sprintf("/usr/sbin/birdc 'show route primary where bgp_path ~ [= * %d =]' | grep -Ev 'BIRD|device1|name|info|kernel1' | awk '{print $1}'", as)
	log.Printf(cmd)
	out, err = com.GetOutput(cmd)
	if err != nil {
		return &pb.SourceResponse{}, err
	}

	for _, address := range strings.Fields(out) {
		addrMask := strings.Split(address, "/")
		prefixes = append(prefixes, &pb.IpAddress{
			Address: addrMask[0],
			Mask:    com.StringToUint32(addrMask[1]),
		})
	}

	v4Count := len(prefixes) - v6Count

	if out == "" {
		return &pb.SourceResponse{}, err

	}
	return &pb.SourceResponse{
		Exists:    true,
		IpAddress: prefixes,
		V4Count:   uint32(v4Count),
		V6Count:   uint32(v6Count),
	}, err

}

// getOriginFromDaemon will get the origin ASN for the passed in IP directly from the BGP daemon.
func getOriginFromDaemon(ip net.IP) (int, bool, error) {
	log.Printf("Running getOriginFromDaemon")

	var daemon string

	switch ip.To4() {
	case nil:
		daemon = "birdc6"
	default:
		daemon = "birdc"
	}
	cmd := fmt.Sprintf("/usr/sbin/%s show route primary for %s | grep -Ev 'BIRD|device1|name|info|kernel1' | awk '{print $NF}' | tr -d '[]ASie?'", daemon, ip.String())
	//log.Printf(cmd)
	out, err := com.GetOutput(cmd)
	if err != nil {
		return 0, false, err
	}

	log.Printf(out)

	if strings.Contains("not in table", out) {
		return 0, false, nil
	}

	source, err := strconv.Atoi(out)
	if err != nil {
		return 0, true, err
	}

	return source, true, nil

}

// getASPathFromDaemon will get the ASN list for the passed in IP directly from the BGP daemon.
func getASPathFromDaemon(ip net.IP) ([]*pb.Asn, []*pb.Asn, bool, error) {
	log.Printf("Running getASPathFromDaemon")

	var daemon string

	var asns, asSet []*pb.Asn

	switch ip.To4() {
	case nil:
		daemon = "birdc6"
	default:
		daemon = "birdc"
	}
	cmd := fmt.Sprintf("/usr/sbin/%s show route primary all for %s | grep -Ev 'BIRD|device1|name|info|kernel1' | grep as_path | awk '{$1=\"\"; print $0}'", daemon, ip.String())
	log.Printf(cmd)
	out, err := com.GetOutput(cmd)
	if err != nil {
		return nil, nil, false, err
	}

	log.Printf(out)

	if out == "" {
		return nil, nil, false, nil

	}

	aspath := strings.Fields(out)

	// Need to separate as-set
	var set bool
	for _, as := range aspath {
		if strings.ContainsAny(as, "{}") {
			set = true
			continue
		}

		switch {
		case set == false:
			asns = append(asns, &pb.Asn{
				Asplain: com.StringToUint32(as),
				Asdot:   com.ASPlainToASDot(com.StringToUint32(as)),
			})
		case set == true:
			asns = append(asns, &pb.Asn{
				Asplain: com.StringToUint32(as),
				Asdot:   com.ASPlainToASDot(com.StringToUint32(as)),
			})
		}
	}

	return asns, asSet, true, nil

}

// getRouteFromDaemon will get the prefix for the passed in IP directly from the BGP daemon.
// If network not found, returns false but no error.
func getRouteFromDaemon(ip net.IP) (*net.IPNet, bool, error) {
	log.Printf("Running getRouteFromDaemon")

	var daemon string

	switch ip.To4() {
	case nil:
		daemon = "birdc6"
	default:
		daemon = "birdc"
	}
	cmd := fmt.Sprintf("/usr/sbin/%s show route primary for %s | grep -Ev 'BIRD|device1|name|info|kernel1' | awk '{print $1}' | tr -d '[]ASie?'", daemon, ip.String())
	out, err := com.GetOutput(cmd)
	if err != nil {
		return nil, false, err
	}

	_, net, err := net.ParseCIDR(out)
	if err != nil {
		return nil, false, nil
	}

	return net, true, nil

}

func getRoaFromDaemon(ip net.IP) (*pb.RoaResponse, error) {
	// I need to get the correct things here!
	statuses := map[string]pb.RoaResponse_ROAStatus{
		"(enum 35)0": pb.RoaResponse_UNKNOWN,
		"(enum 35)2": pb.RoaResponse_INVALID,
		"(enum 35)1": pb.RoaResponse_VALID,
	}

	var daemon string

	switch ip.To4() {
	case nil:
		daemon = "birdc6"
	default:
		daemon = "birdc"
	}

	// In order to check the ROA, I need the current route and origin AS.
	prefix, exists, err := getRouteFromDaemon(ip)
	if err != nil || !exists {
		return &pb.RoaResponse{}, err
	}

	origin, exists, err := getOriginFromDaemon(ip)
	if err != nil || !exists {
		return &pb.RoaResponse{}, err
	}

	// Now check for an existing ROA
	cmd := fmt.Sprintf("/usr/sbin/%s 'eval roa_check(roa_table, %s, %d)' | grep -Ev 'BIRD|device1|name|info|kernel1'", daemon, prefix.String(), origin)
	log.Printf(cmd)
	out, err := com.GetOutput(cmd)
	if err != nil {
		return &pb.RoaResponse{}, err
	}

	log.Printf(out)

	mask, _ := prefix.Mask.Size()
	return &pb.RoaResponse{
		IpAddress: &pb.IpAddress{
			Address: prefix.IP.String(),
			Mask:    uint32(mask),
		},
		Status: statuses[out],
		Exists: exists,
	}, nil

}
