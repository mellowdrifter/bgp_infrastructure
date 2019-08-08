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

type config struct {
	port     string
	priority uint
	peer     string
	logfile  string
	db       dbinfo
}

type dbinfo struct {
	user, pass, dbname string
}

type server struct {
	cfg config
	db  *sql.DB
}

// readConfig is here to read all the config.ini options. Ensure they are correct.
func readConfig() config {

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

	var cfg config
	cfg.port = fmt.Sprintf(":" + cf.Section("grpc").Key("port").String())
	cfg.logfile = fmt.Sprintf(cf.Section("log").Key("file").String())
	cfg.db.dbname = cf.Section("sql").Key("database").String()
	cfg.db.user = cf.Section("sql").Key("username").String()
	cfg.db.pass = cf.Section("sql").Key("password").String()

	return cfg

}

func main() {

	var bgpinfoServer server
	bgpinfoServer.cfg = readConfig()

	// Set up log file
	f, err := os.OpenFile(bgpinfoServer.cfg.logfile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatalf("failed to open logfile: %v\n", err)
	}
	defer f.Close()
	log.SetOutput(f)

	// Create sql handle and test database connection
	sqlserver := fmt.Sprintf("%s:%s@tcp(127.0.0.1:3306)/%s",
		bgpinfoServer.cfg.db.user, bgpinfoServer.cfg.db.pass, bgpinfoServer.cfg.db.dbname)
	db, err := sql.Open("mysql", sqlserver)
	if err != nil {
		log.Fatalf("can't open database. Got %v", err)
	}
	err = db.Ping()
	if err != nil {
		log.Fatalf("can't ping database. Got %v", err)
	}
	defer db.Close()

	// set up gRPC server
	log.Printf("Listening on port %s\n", bgpinfoServer.cfg.port)
	lis, err := net.Listen("tcp", bgpinfoServer.cfg.port)
	if err != nil {
		log.Fatalf("Failed to bind: %v", err)
	}
	grpcServer := grpc.NewServer()
	pb.RegisterBgpInfoServer(grpcServer, &bgpinfoServer)

	grpcServer.Serve(lis)
}

func (s *server) AddLatest(ctx context.Context, v *pb.Values) (*pb.Result, error) {
	// Receive the latest BGP info updates and add to the database
	log.Println("Received an update")
	log.Println(proto.MarshalTextString(v))

	// get correct struct
	update := repack(v)

	// update database
	err := add(update, s.db)
	if err != nil {
		log.Printf("Got error in AddLatest: %s\n", err)
		return &pb.Result{}, err
	}

	return &pb.Result{
		Success: true,
	}, nil
}

func (s *server) GetPrefixCount(ctx context.Context, e *pb.Empty) (*pb.PrefixCountResponse, error) {
	// Pull prefix counts for tweeting. Latest, 6 hours ago, and a week ago.
	log.Println("Running GetPrefixCount")

	res, err := getPrefixCountHelper(s.db)
	if err != nil {
		log.Printf("Got error in GetPrefixCount: %s\n", err)
		return &pb.PrefixCountResponse{}, err
	}

	return res, nil
}

func (s *server) GetPieSubnets(ctx context.Context, e *pb.Empty) (*pb.PieSubnetsResponse, error) {
	// Pull subnets counts to create Pie graph.
	log.Println("Running GetPieSubnets")

	res, err := getPieSubnetsHelper(s.db)
	if err != nil {
		log.Printf("Got error in GetPieSubnets: %s\n", err)
		return &pb.PieSubnetsResponse{}, err
	}

	return res, nil
}

func (s *server) GetMovementTotals(ctx context.Context, t *pb.MovementRequest) (*pb.MovementTotalsResponse, error) {
	// Pull subnets counts to create Pie graph.
	log.Println("Running GetMovementTotals")

	res, err := getMovementTotalsHelper(t, s.db)
	if err != nil {
		log.Printf("Got error in GetMovementTotals: %s\n", err)
		return &pb.MovementTotalsResponse{}, err
	}

	return res, nil
}

func (s *server) UpdateTweetBit(ctx context.Context, t *pb.Timestamp) (*pb.Result, error) {
	// Set the tweet bit to the provided time.
	log.Println("Running UpdateTweetBit")
	res, err := updateTweetBitHelper(t.GetTime(), s.db)
	if err != nil {
		log.Printf("Got error in updateTweetBitHelper: %s\n", err)
		return &pb.Result{}, err
	}

	return res, nil

}

func (s *server) GetRpki(ctx context.Context, e *pb.Empty) (*pb.Roas, error) {
	// Pull RPKI counts to create Pie graph.
	log.Println("Running GetRPKI")

	res, err := getRPKIHelper(s.db)
	if err != nil {
		log.Printf("Got error in GetRPKI: %s\n", err)
		return &pb.Roas{}, err
	}

	return res, nil

}

func (s *server) GetAsname(ctx context.Context, a *pb.GetAsnameRequest) (*pb.GetAsnameResponse, error) {
	log.Println("Running GetAsname")

	res, err := getAsnameHelper(a, s.db)
	if err != nil {
		log.Printf("Got error in GetAsname: %s\n", err)
		return &pb.GetAsnameResponse{}, err
	}

	return res, nil

}

func (s *server) UpdateAsnames(ctx context.Context, asn *pb.AsnamesRequest) (*pb.Result, error) {
	//return nil, grpc.Errorf(codes.Unimplemented, "RPC not yet implemented")
	log.Println("Running UpdateAsname")
	fmt.Printf("There are a total of %d AS numbers\n", len(asn.GetAsnNames()))

	res, err := updateASNHelper(asn, s.db)
	if err != nil {
		log.Printf("Got error in UpdateAsnnames: %s\n", err)
		return &pb.Result{}, err
	}

	return res, nil

}
