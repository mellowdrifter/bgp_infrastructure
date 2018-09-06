package main

import (
	"context"
	"log"
	"net"

	pb "github.com/mellowdrifter/bgp_infrastructure/proto/bgpinfo"
	"google.golang.org/grpc"
)

type server struct{}

func main() {
	//Set up gRPC server
	log.Println("Listening on port 7179")
	lis, err := net.Listen("tcp", ":7179")
	if err != nil {
		log.Fatalf("Failed to bind: %v", err)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterBgpInfoServer(grpcServer, &server{})

	grpcServer.Serve(lis)
}

func (s *server) AddLatest(ctx context.Context, v *pb.Values) (*pb.Result, error) {
	log.Printf("Received an update: %+v", v)

	return &pb.Result{
		Success: true,
	}, nil
}
