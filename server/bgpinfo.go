package main

import (
	"context"
	"log"
	"net"

	"github.com/golang/protobuf/proto"
	"github.com/mellowdrifter/bgp_infrastructure/proto/bgpinfo"
	"google.golang.org/grpc"

	pb "github.com/mellowdrifter/bgp_infrastructure/proto/bgpinfo"
)

type server struct{}

func main() {
	// Connected to db server check
	add()

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
	log.Println("Received an update")
	log.Println(proto.MarshalTextString(v))

	return &pb.Result{
		Success: true,
	}, nil
}

func (s *server) Tweet(ctx context.Context, t *bgpinfo.TweetType) (*pb.Result, error) {
	log.Println("Not yet implemented")
	return &pb.Result{}, nil

}
