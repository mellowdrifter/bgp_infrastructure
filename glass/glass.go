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

	ip, err := validateIP(r.GetIpAddress())
	if err != nil {
		log.Printf("Error: %v", err)
		return nil, err
	}

	asn, err := getOriginFromDaemon(ip)
	if err != nil {
		log.Printf("Error: %v", err)
		return nil, err
	}

	return &pb.OriginResponse{OriginAsn: uint32(asn)}, nil

}

func isPublicIP(ip net.IP) bool {
	// TODO: Go 1.13 will add IsPrivate() or simiar.
	// I might be able to get rid of ALL of this!
	return ip.IsGlobalUnicast() && !(ip.IsInterfaceLocalMulticast() || ip.IsLinkLocalMulticast() || ip.IsLoopback() || ip.IsMulticast() || ip.IsUnspecified())

}

// Aspath returns a list of ASNs for an IP address.
func (s *server) Aspath(ctx context.Context, r *pb.AspathRequest) (*pb.AspathResponse, error) {
	log.Printf("Running Aspath")

	ip, err := validateIP(r.GetIpAddress())
	if err != nil {
		log.Printf("Error: %v", err)
		return nil, err
	}

	asns, err := getASPathFromDaemon(ip)
	if err != nil {
		log.Printf("Error: %v", err)
		return nil, err
	}

	return &pb.AspathResponse{Asn: asns}, nil
}

// Route returns the primary active RIB entry for the IP passed.
func (s *server) Route(ctx context.Context, r *pb.RouteRequest) (*pb.RouteResponse, error) {
	log.Printf("Running Route")

	ip, err := validateIP(r.GetIpAddress())
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

	return &pb.RouteResponse{IpAddress: ipaddr}, nil
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
	}, nil

}

func (s *server) Roa(ctx context.Context, r *pb.RoaRequest) (*pb.RoaResponse, error) {
	log.Printf("Running Roa")

	ip, err := validateIP(r.GetIpAddress())
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

// validateIP ensures the IP address is valid. We only care about public IPs.
func validateIP(r *pb.IpAddress) (net.IP, error) {
	log.Printf("Running validateIP")

	ip := net.ParseIP(r.GetAddress())
	if ip == nil {
		return nil, fmt.Errorf("Unable to parse IP")
	}

	if !isPublicIP(ip) {
		return nil, fmt.Errorf("IP is not public")
	}

	return ip, nil

}

// validateIPNet ensures the IP address and mask is valid. We only care about public IPs.
func validateIPNet(r *pb.IpAddress) (*net.IPNet, error) {
	log.Printf("Running validateIPNet")

	ip, net, err := net.ParseCIDR(fmt.Sprintf("%s/%d", r.GetAddress(), r.GetMask()))
	if err != nil {
		return nil, fmt.Errorf("Unable to parse IP and subnet")
	}

	if !isPublicIP(ip) {
		return nil, fmt.Errorf("IP is not public")
	}

	return net, nil

}

// getOriginFromDaemon will get the origin ASN for the passed in IP directly from the BGP daemon.
func getOriginFromDaemon(ip net.IP) (int, error) {
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
		return 0, err
	}

	log.Printf(out)

	if strings.Contains("not in table", out) {
		return 0, fmt.Errorf("Network is not in table")
	}

	source, err := strconv.Atoi(out)
	if err != nil {
		return 0, err
	}

	return source, nil

}

// getASPathFromDaemon will get the ASN list for the passed in IP directly from the BGP daemon.
func getASPathFromDaemon(ip net.IP) ([]uint32, error) {
	log.Printf("Running getASPathFromDaemon")

	var daemon string
	var asns []uint32

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
		return asns, err
	}

	log.Printf(out)

	if out == "" {
		return asns, fmt.Errorf("Network is not in table")

	}

	aspath := strings.Fields(out)

	// Need uint32 representation of the aspath
	for _, asn := range aspath {
		asns = append(asns, com.StringToUint32(asn))
	}

	return asns, nil

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
	statuses := map[string]string{
		"(enum 35)0": "UNKNOWN",
		"(enum 35)2": "INVALID",
		"(enum 35)1": "VALID",
	}

	var daemon string

	switch ip.To4() {
	case nil:
		daemon = "birdc6"
	default:
		daemon = "birdc"
	}
	// Handle errors here
	prefix, _ := getRouteFromDaemon(ip)
	origin, _ := getOriginFromDaemon(ip)

	fmt.Println("SOMETHING")
	cmd := fmt.Sprintf("/usr/sbin/%s 'eval roa_check(roa_table, %s, %d)' | grep -Ev 'BIRD|device1|name|info|kernel1'", daemon, prefix.String(), origin)
	log.Printf(cmd)
	out, err := com.GetOutput(cmd)
	if err != nil {
		return nil, err
	}

	log.Printf(out)

	return &pb.RoaResponse{
		Status: statuses[out],
	}, nil

}
