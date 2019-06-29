package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"

	"google.golang.org/grpc/codes"

	com "github.com/mellowdrifter/bgp_infrastructure/common"
	pb "github.com/mellowdrifter/bgp_infrastructure/proto/glass"
	"google.golang.org/grpc"
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

func (s *server) Route(ctx context.Context, r *pb.RouteRequest) (*pb.RouteResponse, error) {
	log.Printf("Running Route")

	ip, err := validateIP(r.GetIpAddress())
	if err != nil {
		log.Printf("Error: %v", err)
		return nil, err
	}

	ipnet, err := getRouteFromDaemon(ip)
	if err != nil {
		log.Printf("Error: %v", err)
		return nil, err
	}

	mask, _ := ipnet.Mask.Size()
	ipaddr := &pb.IpAddress{
		Address: ipnet.IP.String(),
		Mask:    uint32(mask),
	}

	return &pb.RouteResponse{IpAddress: ipaddr}, nil
}

func (s *server) Asname(ctx context.Context, r *pb.AsnameRequest) (*pb.AsnameResponse, error) {
	return nil, grpc.Errorf(codes.Unimplemented, "RPC not yet implemented")
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
func getRouteFromDaemon(ip net.IP) (*net.IPNet, error) {
	log.Printf("Running getRouteFromDaemon")

	var daemon string

	switch ip.To4() {
	case nil:
		daemon = "birdc6"
	default:
		daemon = "birdc"
	}
	cmd := fmt.Sprintf("/usr/sbin/%s show route primary for %s | grep -Ev 'BIRD|device1|name|info|kernel1' | awk '{print $1}' | tr -d '[]ASie?'", daemon, ip.String())
	//log.Printf(cmd)
	out, err := com.GetOutput(cmd)
	if err != nil {
		return nil, err
	}

	log.Printf(out)

	_, net, err := net.ParseCIDR(out)
	if err != nil {
		return nil, fmt.Errorf("Unable to parse IP and subnet from output")
	}

	return net, nil

}