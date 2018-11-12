package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"path"

	"github.com/golang/protobuf/proto"
	pb "github.com/mellowdrifter/bgp_infrastructure/proto/bgpinfo"
	"google.golang.org/grpc"
	ini "gopkg.in/ini.v1"
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
	logFile := fmt.Sprintf(cfg.Section("log").Key("file").String())

	// Open log file
	f, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatalf("failed to open logfile: %v\n", err)
	}
	defer f.Close()
	log.SetOutput(f)

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
	// Receive the latest BGP info updates and add to the database
	log.Println("Received an update")
	log.Println(proto.MarshalTextString(v))

	//TODO: Move this to a new function and error if values empty
	// get database credentials
	cfg, err := ini.Load("config.ini")
	if err != nil {
		return &pb.Result{
			Success: false,
		}, err
	}
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

func (s *server) GetTweetData(ctx context.Context, t *pb.TweetType) (*pb.PrefixCount, error) {
	// Get BGP data from the database to advertise to the world
	log.Println("Fetching data for tweets")
	log.Println(proto.MarshalTextString(t))

	//TODO: Move this to a new function and error if values empty
	// get database credentials
	cfg, err := ini.Load("config.ini")
	if err != nil {
		return nil, err
	}
	sqlcon := sqlCon{
		database: cfg.Section("sql").Key("database").String(),
		username: cfg.Section("sql").Key("username").String(),
		password: cfg.Section("sql").Key("password").String(),
	}
	prefixes, err := getPrefixCount(t, sqlcon)
	if err != nil {
		return nil, fmt.Errorf("error occured: %v", err)
	}
	return prefixes, nil
}
