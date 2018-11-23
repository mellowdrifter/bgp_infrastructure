package main

import (
	"context"
	"database/sql"
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

type config struct {
	port     string
	priority uint
	peer     string
}

var cfg config
var db *sql.DB

// init is here to read all the config.ini options. Ensure they are correct.
// also sets up database connection and does initial db ping to check connectivity.
func init() {

	// read config
	exe, err := os.Executable()
	if err != nil {
		log.Fatal(err)
	}
	path := fmt.Sprintf("%s/config.ini", path.Dir(exe))
	cf, err := ini.Load(path)
	if err != nil {
		log.Fatalf("failed to read config file: %v\n", err)
	}
	cfg.port = fmt.Sprintf(":" + cf.Section("grpc").Key("port").String())
	cfg.peer = cf.Section("failover").Key("peer").String()
	if cfg.peer == "" {
		log.Fatalf("failover peer must be set")
	}
	cfg.priority, err = cf.Section("failover").Key("priority").Uint()
	if err != nil {
		log.Fatal(err)
	}
	logFile := fmt.Sprintf(cf.Section("log").Key("file").String())
	dname := cf.Section("sql").Key("database").String()
	user := cf.Section("sql").Key("username").String()
	pass := cf.Section("sql").Key("password").String()

	// Set up log file
	f, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatalf("failed to open logfile: %v\n", err)
	}
	defer f.Close()
	log.SetOutput(f)

	// Create sql handle and test database connection
	server := fmt.Sprintf("%s:%s@tcp(127.0.0.1:3306)/%s", user, pass, dname)
	db, err = sql.Open("mysql", server)
	if err != nil {
		log.Fatalf("can't open database. Got %v", err)
	}
	err = db.Ping()
	if err != nil {
		log.Fatalf("can't ping database. Got %v", err)
	}
	defer db.Close()
}

func main() {
	// set up gRPC server
	log.Printf("Listening on port %s\n", cfg.port)
	lis, err := net.Listen("tcp", cfg.port)
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

	// get correct struct
	update := repack(v)

	// update database
	err := add(update)
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

	prefixes, err := getPrefixCount(t)
	if err != nil {
		return nil, fmt.Errorf("error occured: %v", err)
	}
	return prefixes, nil
}

func (s *server) Alive(ctx context.Context, req *pb.Empty) (*pb.Response, error) {
	// When incoming request, should do local health check.
	// then return status with priority set
	if isHealthy() {
		return &pb.Response{
			Status:   true,
			Priority: uint32(cfg.priority),
		}, nil
	}

	// If not healthy, return failed
	return &pb.Response{
		Status: false,
	}, nil
}

func (s *server) IsPrimary(ctx context.Context, m *pb.Empty) (*pb.Active, error) {
	// If our priority is 0, then we are not primary
	if cfg.priority == 0 {
		return &pb.Active{}, nil
	}

	// Connect to peer server. If we can't connect, we are primary
	conn, err := grpc.Dial(cfg.peer, grpc.WithInsecure())
	if err != nil {
		return &pb.Active{
			Primary: true,
		}, err
	}
	defer conn.Close()
	c := pb.NewBgpInfoClient(conn)

	// Check to see if peer is okay, and if so it's priority
	peerState, err := c.Alive(ctx, m)
	if err != nil {
		return &pb.Active{
			Primary: true,
		}, err
	}

	// Not primary if our priority is less than or equal, else we're primary at this point
	if peerState.GetStatus() {
		if uint32(cfg.priority) <= peerState.GetPriority() {
			return &pb.Active{}, err
		}
	}
	return &pb.Active{
		Primary: true,
	}, nil
}
