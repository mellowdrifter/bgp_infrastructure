package main

import (
	"context"
	"fmt"
	"log"
	"net"

	"google.golang.org/grpc/codes"

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

	ip, err := validateOriginRequest(r)
	if err != nil {
		log.Printf("Error: %v", err)
		return nil, err
	}

	asn, err := getOriginFromDaemon(ip)
	if err != nil {
		log.Printf("Error: %v", err)
		return nil, err
	}

	return &pb.OriginResponse{OriginAsn: asn}, nil

}

// validateOriginRequest ensures the IP address is valid. We don't care about the mask.
func validateOriginRequest(r *pb.OriginRequest) (net.IP, error) {

	ip := net.ParseIP(r.GetIpAddress().GetAddress())
	if ip == nil {
		return nil, fmt.Errorf("Unable to parse IP")
	}

	if !isPublicIP(ip) {
		return nil, fmt.Errorf("IP is not public")
	}

	return ip, nil

}

// getOriginFromDaemon will get the origin ASN for the passed in IP directly from the BGP daemon.
func getOriginFromDaemon(net.IP) (uint32, error) {

}

func isPublicIP(ip net.IP) bool {
	// TODO: Go 1.13 will add IsPrivate() or simiar.
	// I might be able to get rid of ALL of this!
	return ip.IsGlobalUnicast() && !(ip.IsInterfaceLocalMulticast() || ip.IsLinkLocalMulticast() || ip.IsLoopback() || ip.IsMulticast() || ip.IsUnspecified())

}

func (s *server) Aspath(ctx context.Context, r *pb.AspathRequest) (*pb.AspathResponse, error) {
	return nil, grpc.Errorf(codes.Unimplemented, "RPC not yet implemented")
}

func (s *server) Route(ctx context.Context, r *pb.RouteRequest) (*pb.RouteResponse, error) {
	return nil, grpc.Errorf(codes.Unimplemented, "RPC not yet implemented")
}

func (s *server) Asname(ctx context.Context, r *pb.AsnameRequest) (*pb.AsnameResponse, error) {
	return nil, grpc.Errorf(codes.Unimplemented, "RPC not yet implemented")
}
