package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net"
	"os"
	"path"
	"strings"

	_ "github.com/go-sql-driver/mysql"
	com "github.com/mellowdrifter/bgp_infrastructure/pkg/common"
	pb "github.com/mellowdrifter/bgp_infrastructure/proto/bgpsql"
	"google.golang.org/grpc"
	ini "gopkg.in/ini.v1"
	_ "modernc.org/sqlite"
)

type config struct {
	port          string
	logfile       string
	driver        string // "mysql" or "sqlite"
	dbpath        string // for sqlite
	dbname        string
	user          string
	pass          string
	defaultSource string
	valCfg        validationConfig
}

type server struct {
	cfg    config
	db     *sql.DB // writer DB (or single DB for MySQL)
	readDB *sql.DB // reader pool for SQLite (or nil if using db)
}

func (s *server) readerDB() *sql.DB {
	if s.readDB != nil {
		return s.readDB
	}
	return s.db
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
	cfg.port = fmt.Sprintf(":%s", cf.Section("grpc").Key("port").String())
	cfg.logfile = cf.Section("log").Key("file").String()
	cfg.driver = cf.Section("sql").Key("driver").MustString("mysql")
	cfg.dbpath = cf.Section("sql").Key("path").MustString("/var/lib/bgpsql/bgp.db")
	cfg.dbname = cf.Section("sql").Key("database").String()
	cfg.user = cf.Section("sql").Key("username").String()
	cfg.pass = cf.Section("sql").Key("password").String()
	cfg.defaultSource = cf.Section("sql").Key("source").MustString("bgp1")

	// Read validation thresholds with safe defaults
	val := defaultValidationConfig()
	valSec := cf.Section("validation")
	if valSec != nil {
		if v := valSec.Key("v4_min").MustUint(0); v != 0 {
			val.v4Min = uint32(v)
		}
		if v := valSec.Key("v4_max").MustUint(0); v != 0 {
			val.v4Max = uint32(v)
		}
		if v := valSec.Key("v6_min").MustUint(0); v != 0 {
			val.v6Min = uint32(v)
		}
		if v := valSec.Key("v6_max").MustUint(0); v != 0 {
			val.v6Max = uint32(v)
		}
		if v := valSec.Key("max_delta_pct").MustFloat64(0); v != 0 {
			val.maxDeltaPct = v
		}
		if v := valSec.Key("max_peer_diff_pct").MustFloat64(0); v != 0 {
			val.maxPeerDiffPct = v
		}
		if v := valSec.Key("min_peer_ratio").MustFloat64(0); v != 0 {
			val.minPeerRatio = v
		}
	}
	cfg.valCfg = val

	return cfg
}

func main() {
	var bgpinfoServer server
	bgpinfoServer.cfg = readConfig()

	// Set up log file
	f, err := os.OpenFile(bgpinfoServer.cfg.logfile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		log.Fatalf("failed to open logfile: %v\n", err)
	}
	defer f.Close()
	log.SetOutput(f)

	// Open database pools based on configured driver
	if strings.EqualFold(bgpinfoServer.cfg.driver, "sqlite") {
		writerDB, readerDB, err := openSQLitePools(bgpinfoServer.cfg.dbpath)
		if err != nil {
			log.Fatalf("can't open sqlite database: %v", err)
		}
		bgpinfoServer.db = writerDB
		bgpinfoServer.readDB = readerDB
		defer writerDB.Close()
		defer readerDB.Close()

		if err := initSQLiteSchema(writerDB); err != nil {
			log.Fatalf("failed to init sqlite schema: %v", err)
		}
		log.Printf("Connected to SQLite database at %s with WAL mode (writer=1, readers=4)\n", bgpinfoServer.cfg.dbpath)
	} else {
		// MySQL / MariaDB
		sqlserver := fmt.Sprintf("%s:%s@tcp(127.0.0.1:3306)/%s",
			bgpinfoServer.cfg.user, bgpinfoServer.cfg.pass,
			bgpinfoServer.cfg.dbname)
		db, err := sql.Open("mysql", sqlserver)
		if err != nil {
			log.Fatalf("can't open database. Got %v", err)
		}
		err = db.Ping()
		if err != nil {
			log.Fatalf("can't ping database. Got %v", err)
		}
		bgpinfoServer.db = db
		bgpinfoServer.readDB = db
		defer db.Close()
		log.Printf("Connected to MySQL/MariaDB database %s\n", bgpinfoServer.cfg.dbname)
	}

	// set up gRPC server
	log.Printf("Listening on port %s\n", bgpinfoServer.cfg.port)
	lis, err := net.Listen("tcp", bgpinfoServer.cfg.port)
	if err != nil {
		log.Fatalf("Failed to bind: %v", err)
	}
	grpcServer := grpc.NewServer(
		grpc.MaxRecvMsgSize(64*1024*1024),
		grpc.MaxSendMsgSize(64*1024*1024),
	)
	pb.RegisterBgpInfoServer(grpcServer, &bgpinfoServer)

	grpcServer.Serve(lis)
}

func (s *server) AddLatest(ctx context.Context, v *pb.Values) (*pb.Result, error) {
	// Receive the latest BGP info updates and add to the database
	log.Println("Running AddLatest")

	// get correct struct
	update := com.ProtoToStruct(v)

	// Fallback to server's configured default source if not provided in proto
	if update.Source == "" {
		if s.cfg.defaultSource != "" {
			update.Source = s.cfg.defaultSource
		} else {
			update.Source = "bgp1"
		}
	}

	// Validate sample at ingest
	quality, note := validateSample(ctx, s.readerDB(), update, s.cfg.valCfg)
	update.Quality = quality
	update.QualityNote = note
	if quality == "suspect" {
		log.Printf("Ingest validation flagged sample as SUSPECT for %s (TIME=%d): %s", update.Source, update.Time, note)
	}

	// update database
	err := addLatestHelper(ctx, update, s.db, s.cfg.driver)
	if err != nil {
		log.Printf("Got error in AddLatest: %s with update %q\n", err, v)
		return nil, err
	}

	return &pb.Result{
		Success: true,
	}, nil
}

func (s *server) GetPrefixCount(ctx context.Context, e *pb.Empty) (*pb.PrefixCountResponse, error) {
	// Pull prefix counts for tweeting. Latest, 6 hours ago, and a week ago.
	log.Println("Running GetPrefixCount")

	res, err := getPrefixCountHelper(ctx, s.readerDB())
	if err != nil {
		log.Printf("Got error in GetPrefixCount: %s\n", err)
		return nil, err
	}

	return res, nil
}

func (s *server) GetPieSubnets(ctx context.Context, e *pb.Empty) (*pb.PieSubnetsResponse, error) {
	// Pull subnets counts to create Pie graph.
	log.Println("Running GetPieSubnets")

	res, err := getPieSubnetsHelper(ctx, s.readerDB())
	if err != nil {
		log.Printf("Got error in GetPieSubnets: %s\n", err)
		return nil, err
	}

	return res, nil
}

func (s *server) GetMovementTotals(ctx context.Context, t *pb.MovementRequest) (*pb.MovementTotalsResponse, error) {
	// Pull subnets counts to create Pie graph.
	log.Println("Running GetMovementTotals")

	res, err := getMovementTotalsHelper(ctx, t, s.readerDB())
	if err != nil {
		log.Printf("Got error in GetMovementTotals: %s\n", err)
		return nil, err
	}

	return res, nil
}

func (s *server) UpdateTweetBit(ctx context.Context, t *pb.Timestamp) (*pb.Result, error) {
	// Set the tweet bit to the provided time.
	log.Println("Running UpdateTweetBit")
	res, err := updateTweetBitHelper(ctx, t.GetTime(), s.db)
	if err != nil {
		log.Printf("Got error in updateTweetBitHelper: %s\n", err)
		return nil, err
	}

	return res, nil
}

func (s *server) GetRpki(ctx context.Context, e *pb.Empty) (*pb.Roas, error) {
	// Pull RPKI counts to create Pie graph.
	log.Println("Running GetRPKI")

	res, err := getRPKIHelper(ctx, s.readerDB())
	if err != nil {
		log.Printf("Got error in GetRPKI: %s\n", err)
		return nil, err
	}

	return res, nil
}

func (s *server) GetAsname(ctx context.Context, a *pb.GetAsnameRequest) (*pb.GetAsnameResponse, error) {
	log.Println("Running GetAsname")

	res, err := getAsnameHelper(ctx, a, s.readerDB())
	if err != nil {
		log.Printf("Got error in GetAsname: %s\n", err)
		return nil, err
	}

	return res, nil
}

func (s *server) GetAsnames(ctx context.Context, e *pb.Empty) (*pb.GetAsnamesResponse, error) {
	log.Println("Running GetAsNames")

	res, err := getAsnamesHelper(ctx, s.readerDB())
	if err != nil {
		log.Printf("Got error in GetAsnames: %s\n", err)
		return nil, err
	}
	return res, nil
}

func (s *server) UpdateAsnames(ctx context.Context, asn *pb.AsnamesRequest) (*pb.Result, error) {
	log.Println("Running UpdateAsname")
	fmt.Printf("There are a total of %d AS numbers\n", len(asn.GetAsnNames()))

	res, err := updateASNHelper(ctx, asn, s.db)
	if err != nil {
		log.Printf("Got error in UpdateAsnnames: %s\n", err)
		return nil, err
	}

	return res, nil
}

func (s *server) GetSampleIndex(ctx context.Context, req *pb.SampleIndexRequest) (*pb.SampleIndexResponse, error) {
	log.Printf("Running GetSampleIndex (since=%d, until=%d)\n", req.GetSinceTime(), req.GetUntilTime())
	res, err := getSampleIndexHelper(ctx, req.GetSinceTime(), req.GetUntilTime(), s.readerDB())
	if err != nil {
		log.Printf("Got error in GetSampleIndex: %v\n", err)
		return nil, err
	}
	return res, nil
}

func (s *server) GetSampleBatch(ctx context.Context, req *pb.SampleBatchRequest) (*pb.SampleBatchResponse, error) {
	log.Printf("Running GetSampleBatch (count=%d)\n", len(req.GetKeys()))
	res, err := getSampleBatchHelper(ctx, req.GetKeys(), s.readerDB())
	if err != nil {
		log.Printf("Got error in GetSampleBatch: %v\n", err)
		return nil, err
	}
	return res, nil
}

func (s *server) AddSampleBatch(ctx context.Context, req *pb.SampleBatchResponse) (*pb.Result, error) {
	log.Printf("Running AddSampleBatch (count=%d)\n", len(req.GetSamples()))
	res, err := addSampleBatchHelper(ctx, req.GetSamples(), s.db, s.cfg.driver)
	if err != nil {
		log.Printf("Got error in AddSampleBatch: %v\n", err)
		return nil, err
	}
	return res, nil
}
