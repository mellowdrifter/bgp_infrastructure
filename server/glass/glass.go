package main

import (
	"context"
	"log"
	"net"

	"google.golang.org/grpc/codes"

	pb "github.com/mellowdrifter/bgp_infrastructure/proto/glass"
	"google.golang.org/grpc"
)

type server struct{}

func main() {
	// set up gRPC server
	log.Printf("Listening on port %d\n", 9999)
	lis, err := net.Listen("tcp", ":9999")
	if err != nil {
		log.Fatalf("Failed to bind: %v", err)
	}
	grpcServer := grpc.NewServer()
	pb.RegisterLookingGlassServer(grpcServer, &server{})

	grpcServer.Serve(lis)
}

func (s *server) Origin(ctx context.Context, r *pb.OriginRequest) (*pb.OriginResponse, error) {
	return nil, grpc.Errorf(codes.Unimplemented, "RPC not yet implemented")
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
