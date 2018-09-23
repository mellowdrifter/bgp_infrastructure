package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"path"

	"github.com/golang/protobuf/proto"
	"github.com/mellowdrifter/bgp_infrastructure/proto/bgpinfo"
	"google.golang.org/grpc"
	ini "gopkg.in/ini.v1"

	pb "github.com/mellowdrifter/bgp_infrastructure/proto/bgpinfo"
)

type server struct{}

type sqlCon struct {
	database string
	username string
	password string
}

func main() {

	// read config
	exe, err := os.Executable()
	if err != nil {
		log.Fatal(err)
	}
	path := fmt.Sprintf("%s/config.ini", path.Dir(exe))
	cfg, err := ini.Load(path)
	if err != nil {
		log.Fatalf("failed to read config file: %v\n", err)
	}
	port := fmt.Sprintf(":" + cfg.Section("grpc").Key("port").String())

	// set up gRPC server
	log.Printf("Listening on port %s\n", port)
	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("Failed to bind: %v", err)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterBgpInfoServer(grpcServer, &server{})

	grpcServer.Serve(lis)
}

func (s *server) AddLatest(ctx context.Context, v *pb.Values) (*pb.Result, error) {
	// Receive the latest BGP info updates and add this to the database
	log.Println("Received an update")
	log.Println(proto.MarshalTextString(v))

	// get database credentials
	cfg, err := ini.Load("config.ini")
	if err != nil {
		return &pb.Result{
			Success: false,
		}, err
	}
	//TODO: Move this to a new function and error if values empty
	sqlcon := sqlCon{
		database: cfg.Section("sql").Key("database").String(),
		username: cfg.Section("sql").Key("username").String(),
		password: cfg.Section("sql").Key("password").String(),
	}

	// get correct struct
	update := repack(v)

	// update database
	err = add(update, sqlcon)
	if err != nil {
		return &pb.Result{
			Success: false,
		}, err
	}

	return &pb.Result{
		Success: true,
	}, nil
}

func (s *server) GetTweetData(ctx context.Context, t *bgpinfo.TweetType) (*pb.Result, error) {
	log.Println("Not yet implemented")
	return &pb.Result{}, nil

}
