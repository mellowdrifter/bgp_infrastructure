package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"
	com "github.com/mellowdrifter/bgp_infrastructure/pkg/common"
	pb "github.com/mellowdrifter/bgp_infrastructure/proto/bgpsql"
	_ "modernc.org/sqlite"
)

func readOne(f string) *pb.Values {
	file := fmt.Sprintf("./testdata/%s", f)
	in, err := os.ReadFile(file)
	if err != nil {
		log.Fatalln("Error reading file:", err)
	}

	values := pb.Values{}

	if err := proto.UnmarshalText(string(in), &values); err != nil {
		log.Fatalln("Failed to parse latest values:", err)
	}

	return &values
}

func readAnnual(f string) []*com.BgpUpdate {
	file := fmt.Sprintf("./testdata/%s", f)
	in, err := os.ReadFile(file)
	if err != nil {
		log.Fatalln("Error reading file:", err)
	}

	values := pb.ListOfValues{}
	if err := proto.UnmarshalText(string(in), &values); err != nil {
		log.Fatalln("Failed to parse latest values:", err)
	}

	var structValues []*com.BgpUpdate

	for _, value := range values.GetValues() {
		structValues = append(structValues, com.ProtoToStruct(value))
	}

	return structValues
}

func populate(db *sql.DB) {
	values := readAnnual("month.pb")
	ctx := context.Background()
	for _, b := range values {
		if err := addLatestHelper(ctx, b, db, "sqlite"); err != nil {
			log.Fatalln("Error on populate addLatestHelper:", err)
		}
	}
}

func createTestDatabase() {
	db, err := sql.Open("sqlite", "./testdata/bgpinfo.db")
	if err != nil {
		log.Panic("Unable to open test database:", err)
	}
	defer db.Close()

	tx, err := db.Begin()
	if err != nil {
		log.Panic("Unable to begin tx:", err)
	}
	tx.Exec(`DROP VIEW IF EXISTS curated_samples`)
	tx.Exec(`DROP TABLE IF EXISTS INFO`)
	tx.Exec(`DROP TABLE IF EXISTS ASNUMNAME`)
	tx.Exec(`DROP TABLE IF EXISTS ASNUMNAME_NEW`)

	if _, err := tx.Exec(sqliteSchemaDDL); err != nil {
		tx.Rollback()
		log.Panic("Unable to create test schema:", err)
	}

	if err := tx.Commit(); err != nil {
		log.Panic("Unable to create test database:", err)
	}
}

func TestAddLatest(t *testing.T) {
	createTestDatabase()

	var bgpinfoServer server
	bgpinfoServer.cfg = config{
		driver:        "sqlite",
		defaultSource: "bgp1",
		valCfg: validationConfig{
			v4Min:          700000,
			v4Max:          1600000,
			v6Min:          50000,
			v6Max:          600000,
			maxDeltaPct:    10.0,
			maxPeerDiffPct: 15.0,
			minPeerRatio:   0.5,
		},
	}

	db, err := sql.Open("sqlite", "./testdata/bgpinfo.db")
	if err != nil {
		t.Fatalf("can't open database: %v", err)
	}
	bgpinfoServer.db = db
	defer db.Close()

	pbVal := readOne("latest.pb")
	res, err := bgpinfoServer.AddLatest(context.Background(), pbVal)
	if err != nil {
		t.Fatalf("AddLatest failed: %v", err)
	}
	if !res.GetSuccess() {
		t.Errorf("expected success true, got %v", res.GetSuccess())
	}
}

func TestGetPrefixCount(t *testing.T) {
	createTestDatabase()

	db, err := sql.Open("sqlite", "./testdata/bgpinfo.db")
	if err != nil {
		t.Fatalf("can't open database: %v", err)
	}
	defer db.Close()

	populate(db)

	res, err := getPrefixCountHelper(context.Background(), db)
	if err != nil {
		t.Fatalf("getPrefixCountHelper failed: %v", err)
	}
	if res.GetActive_4() == 0 {
		t.Errorf("expected non-zero active v4 count")
	}
}

func TestGetPieSubnets(t *testing.T) {
	createTestDatabase()

	db, err := sql.Open("sqlite", "./testdata/bgpinfo.db")
	if err != nil {
		t.Fatalf("can't open database: %v", err)
	}
	defer db.Close()

	populate(db)

	res, err := getPieSubnetsHelper(context.Background(), db)
	if err != nil {
		t.Fatalf("getPieSubnetsHelper failed: %v", err)
	}
	if res.GetV4Total() == 0 {
		t.Errorf("expected non-zero v4 total")
	}
}

func TestGetMovementTotals(t *testing.T) {
	createTestDatabase()

	db, err := sql.Open("sqlite", "./testdata/bgpinfo.db")
	if err != nil {
		t.Fatalf("can't open database: %v", err)
	}
	defer db.Close()

	populate(db)

	req := &pb.MovementRequest{
		Period: pb.MovementRequest_WEEK,
	}
	res, err := getMovementTotalsHelper(context.Background(), req, db)
	if err != nil {
		t.Fatalf("getMovementTotalsHelper failed: %v", err)
	}
	_ = res
}

func TestUpdateTweetBit(t *testing.T) {
	createTestDatabase()

	db, err := sql.Open("sqlite", "./testdata/bgpinfo.db")
	if err != nil {
		t.Fatalf("can't open database: %v", err)
	}
	defer db.Close()

	populate(db)

	res, err := updateTweetBitHelper(context.Background(), 1580515200, db)
	if err != nil {
		t.Fatalf("updateTweetBitHelper failed: %v", err)
	}
	if !res.GetSuccess() {
		t.Errorf("expected success true")
	}
}

func TestGetRpki(t *testing.T) {
	createTestDatabase()

	db, err := sql.Open("sqlite", "./testdata/bgpinfo.db")
	if err != nil {
		t.Fatalf("can't open database: %v", err)
	}
	defer db.Close()

	populate(db)

	res, err := getRPKIHelper(context.Background(), db)
	if err != nil {
		t.Fatalf("getRPKIHelper failed: %v", err)
	}
	_ = res
}

func TestAsnames(t *testing.T) {
	createTestDatabase()

	db, err := sql.Open("sqlite", "./testdata/bgpinfo.db")
	if err != nil {
		t.Fatalf("can't open database: %v", err)
	}
	defer db.Close()

	req := &pb.AsnamesRequest{
		AsnNames: []*pb.AsnName{
			{AsNumber: 13335, AsName: "CLOUDFLARENET", AsLocale: "US"},
			{AsNumber: 15169, AsName: "GOOGLE", AsLocale: "US"},
		},
	}
	res, err := updateASNHelper(context.Background(), req, db)
	if err != nil {
		t.Fatalf("updateASNHelper failed: %v", err)
	}
	if !res.GetSuccess() {
		t.Errorf("expected success true")
	}

	getReq := &pb.GetAsnameRequest{AsNumber: 13335}
	asResp, err := getAsnameHelper(context.Background(), getReq, db)
	if err != nil {
		t.Fatalf("getAsnameHelper failed: %v", err)
	}
	if !asResp.GetExists() || asResp.GetAsName() != "CLOUDFLARENET" {
		t.Errorf("expected CLOUDFLARENET, got %v", asResp.GetAsName())
	}

	allResp, err := getAsnamesHelper(context.Background(), db)
	if err != nil {
		t.Fatalf("getAsnamesHelper failed: %v", err)
	}
	if len(allResp.GetAsnumnames()) != 2 {
		t.Errorf("expected 2 asnames, got %d", len(allResp.GetAsnumnames()))
	}
}

func TestCuratedViewPreference(t *testing.T) {
	createTestDatabase()

	db, err := sql.Open("sqlite", "./testdata/bgpinfo.db")
	if err != nil {
		t.Fatalf("Open test DB failed: %v", err)
	}
	defer db.Close()

	ts := uint64(1710000000)

	// Insert both bgp1 and bgp3 at the same timestamp with quality='ok'
	_, err = db.Exec(`INSERT INTO INFO (source, TIME, quality, V4COUNT, V6COUNT, V4_24, V6_48, TWEET) VALUES ('bgp1', ?, 'ok', 1000000, 200000, 50000, 10000, 1)`, ts)
	if err != nil {
		t.Fatalf("Insert bgp1 failed: %v", err)
	}
	_, err = db.Exec(`INSERT INTO INFO (source, TIME, quality, V4COUNT, V6COUNT, V4_24, V6_48, TWEET) VALUES ('bgp3', ?, 'ok', 1000500, 200500, 50500, 10500, 1)`, ts)
	if err != nil {
		t.Fatalf("Insert bgp3 failed: %v", err)
	}

	// Verify curated_samples view selects bgp3
	var derivedFrom string
	var v4 uint32
	err = db.QueryRow("SELECT derived_from, V4COUNT FROM curated_samples WHERE TIME = ?", ts).Scan(&derivedFrom, &v4)
	if err != nil {
		t.Fatalf("Query curated_samples failed: %v", err)
	}
	if derivedFrom != "bgp3" || v4 != 1000500 {
		t.Errorf("Expected bgp3 (1000500), got %s (%d)", derivedFrom, v4)
	}

	// Verify getPrefixCountHelper returns bgp3 data
	resp, err := getPrefixCountHelper(context.Background(), db)
	if err != nil {
		t.Fatalf("getPrefixCountHelper failed: %v", err)
	}
	if resp.GetActive_4() != 1000500 {
		t.Errorf("Expected getPrefixCountHelper to return bgp3 active_4 1000500, got %d", resp.GetActive_4())
	}
}

func TestCuratedViewFallback(t *testing.T) {
	createTestDatabase()

	db, err := sql.Open("sqlite", "./testdata/bgpinfo.db")
	if err != nil {
		t.Fatalf("Open test DB failed: %v", err)
	}
	defer db.Close()

	ts := uint64(1710000000)

	// bgp3 is degraded/suspect, bgp1 is ok
	_, err = db.Exec(`INSERT INTO INFO (source, TIME, quality, V4COUNT, V6COUNT, V4_24, V6_48, TWEET) VALUES ('bgp1', ?, 'ok', 1000000, 200000, 50000, 10000, 1)`, ts)
	if err != nil {
		t.Fatalf("Insert bgp1 failed: %v", err)
	}
	_, err = db.Exec(`INSERT INTO INFO (source, TIME, quality, V4COUNT, V6COUNT, V4_24, V6_48, TWEET) VALUES ('bgp3', ?, 'suspect', 200000, 50000, 10000, 2000, 1)`, ts)
	if err != nil {
		t.Fatalf("Insert bgp3 failed: %v", err)
	}

	// Verify curated_samples view falls back to bgp1
	var derivedFrom string
	var v4 uint32
	err = db.QueryRow("SELECT derived_from, V4COUNT FROM curated_samples WHERE TIME = ?", ts).Scan(&derivedFrom, &v4)
	if err != nil {
		t.Fatalf("Query curated_samples failed: %v", err)
	}
	if derivedFrom != "bgp1" || v4 != 1000000 {
		t.Errorf("Expected fallback to bgp1 (1000000), got %s (%d)", derivedFrom, v4)
	}

	// Verify getPrefixCountHelper returns bgp1 data
	resp, err := getPrefixCountHelper(context.Background(), db)
	if err != nil {
		t.Fatalf("getPrefixCountHelper failed: %v", err)
	}
	if resp.GetActive_4() != 1000000 {
		t.Errorf("Expected getPrefixCountHelper fallback to 1000000, got %d", resp.GetActive_4())
	}
}

func TestCuratedMovementTotalsDeduplicated(t *testing.T) {
	createTestDatabase()

	db, err := sql.Open("sqlite", "./testdata/bgpinfo.db")
	if err != nil {
		t.Fatalf("Open test DB failed: %v", err)
	}
	defer db.Close()

	baseTime := uint64(time.Now().Unix() - 66600 - 604800 + 3600) // within WEEK window
	baseTime = baseTime - (baseTime % 300)

	// Insert 10 dual-written timestamps
	for i := 0; i < 10; i++ {
		ts := baseTime + uint64(i*300)
		_, err = db.Exec(`INSERT INTO INFO (source, TIME, quality, V4COUNT, V6COUNT) VALUES ('bgp1', ?, 'ok', ?, 200000)`, ts, 1000000+i)
		if err != nil {
			t.Fatalf("Insert bgp1 failed: %v", err)
		}
		_, err = db.Exec(`INSERT INTO INFO (source, TIME, quality, V4COUNT, V6COUNT) VALUES ('bgp3', ?, 'ok', ?, 200000)`, ts, 1050000+i)
		if err != nil {
			t.Fatalf("Insert bgp3 failed: %v", err)
		}
	}

	req := &pb.MovementRequest{
		Period: pb.MovementRequest_WEEK,
	}
	resp, err := getMovementTotalsHelper(context.Background(), req, db)
	if err != nil {
		t.Fatalf("getMovementTotalsHelper failed: %v", err)
	}

	var lastTime uint64
	for idx, val := range resp.GetValues() {
		if idx > 0 && val.GetTime() <= lastTime {
			t.Errorf("Timestamps not strictly increasing: prev=%d, curr=%d", lastTime, val.GetTime())
		}
		lastTime = val.GetTime()
		if val.GetV4Values() < 1050000 {
			t.Errorf("Expected bgp3 value >= 1050000, got %d", val.GetV4Values())
		}
	}
}

func TestReconcileIndexAndBatch(t *testing.T) {
	createTestDatabase()

	db, err := sql.Open("sqlite", "./testdata/bgpinfo.db")
	if err != nil {
		t.Fatalf("Open test DB failed: %v", err)
	}
	defer db.Close()

	t0 := uint64(1710000000)
	t1 := t0 + 300
	t2 := t1 + 300

	_, err = db.Exec(`INSERT INTO INFO (source, TIME, quality, quality_note, V4COUNT, V6COUNT, V4TOTAL, V6TOTAL) VALUES ('bgp1', ?, 'ok', 'clean', 1000000, 200000, 1000000, 200000)`, t0)
	if err != nil {
		t.Fatalf("Insert t0 failed: %v", err)
	}
	_, err = db.Exec(`INSERT INTO INFO (source, TIME, quality, quality_note, V4COUNT, V6COUNT, V4TOTAL, V6TOTAL) VALUES ('bgp3', ?, 'ok', 'clean', 1000500, 200500, 1000500, 200500)`, t0)
	if err != nil {
		t.Fatalf("Insert t0 bgp3 failed: %v", err)
	}
	_, err = db.Exec(`INSERT INTO INFO (source, TIME, quality, quality_note, V4COUNT, V6COUNT, V4TOTAL, V6TOTAL) VALUES ('bgp1', ?, 'suspect', 'step change', 900000, 190000, 900000, 190000)`, t1)
	if err != nil {
		t.Fatalf("Insert t1 failed: %v", err)
	}
	_, err = db.Exec(`INSERT INTO INFO (source, TIME, quality, quality_note, V4COUNT, V6COUNT, V4TOTAL, V6TOTAL) VALUES ('bgp3', ?, 'ok', 'clean', 1000600, 200600, 1000600, 200600)`, t2)
	if err != nil {
		t.Fatalf("Insert t2 failed: %v", err)
	}

	indexResp, err := getSampleIndexHelper(context.Background(), t0, t2, db)
	if err != nil {
		t.Fatalf("getSampleIndexHelper failed: %v", err)
	}
	if len(indexResp.GetKeys()) != 4 {
		t.Fatalf("Expected 4 keys in index, got %d", len(indexResp.GetKeys()))
	}
	foundSuspect := false
	for _, k := range indexResp.GetKeys() {
		if k.GetTime() == t1 && k.GetSource() == "bgp1" && k.GetQuality() == "suspect" {
			foundSuspect = true
		}
	}
	if !foundSuspect {
		t.Errorf("Expected to find suspect key for bgp1@%d", t1)
	}

	batchKeys := []*pb.SampleKey{
		{Source: "bgp1", Time: t0},
		{Source: "bgp3", Time: t2},
	}
	batchResp, err := getSampleBatchHelper(context.Background(), batchKeys, db)
	if err != nil {
		t.Fatalf("getSampleBatchHelper failed: %v", err)
	}
	if len(batchResp.GetSamples()) != 2 {
		t.Fatalf("Expected 2 samples, got %d", len(batchResp.GetSamples()))
	}

	targetDBPath := "./testdata/bgpinfo_target.db"
	targetDB, err := sql.Open("sqlite", targetDBPath)
	if err != nil {
		t.Fatalf("Open target DB failed: %v", err)
	}
	defer func() {
		targetDB.Close()
		os.Remove(targetDBPath)
	}()

	if _, err := targetDB.Exec(sqliteSchemaDDL); err != nil {
		t.Fatalf("Failed to create target schema: %v", err)
	}

	addRes, err := addSampleBatchHelper(context.Background(), batchResp.GetSamples(), targetDB, "sqlite")
	if err != nil {
		t.Fatalf("addSampleBatchHelper failed: %v", err)
	}
	if !addRes.GetSuccess() {
		t.Fatalf("addSampleBatchHelper returned success=false")
	}

	var rowCount int
	err = targetDB.QueryRow("SELECT COUNT(*) FROM INFO").Scan(&rowCount)
	if err != nil || rowCount != 2 {
		t.Errorf("Expected 2 rows in targetDB, got %d (err: %v)", rowCount, err)
	}
}

// TestSQLiteDualPool tests reader/writer concurrency with WAL mode and dual connection pools.
func TestSQLiteDualPool(t *testing.T) {
	testDB := "./testdata/dualpool_test.db"
	defer os.Remove(testDB)
	defer os.Remove(testDB + "-wal")
	defer os.Remove(testDB + "-shm")

	writeDB, readDB, err := openSQLitePools(testDB)
	if err != nil {
		t.Fatalf("openSQLitePools failed: %v", err)
	}
	defer writeDB.Close()
	defer readDB.Close()

	if err := initSQLiteSchema(writeDB); err != nil {
		t.Fatalf("initSQLiteSchema failed: %v", err)
	}

	var wg sync.WaitGroup
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Launch concurrent readers
	readerDone := make(chan struct{})
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-readerDone:
					return
				case <-ctx.Done():
					return
				default:
					var count int
					_ = readDB.QueryRowContext(ctx, "SELECT COUNT(*) FROM INFO").Scan(&count)
					time.Sleep(5 * time.Millisecond)
				}
			}
		}()
	}

	// Writer writes 20 samples sequentially
	for i := 0; i < 20; i++ {
		sample := &com.BgpUpdate{
			Source:     "bgp3",
			Time:       uint64(1720000000 + i*300),
			V4Count:    1000000 + uint32(i),
			V6Count:    200000 + uint32(i),
			Quality:    "ok",
			SampleTime: uint64(1720000000 + i*300),
		}
		if err := addLatestHelper(ctx, sample, writeDB, "sqlite"); err != nil {
			t.Fatalf("Write sample %d failed: %v", i, err)
		}
	}

	close(readerDone)
	wg.Wait()

	var finalCount int
	if err := readDB.QueryRow("SELECT COUNT(*) FROM INFO").Scan(&finalCount); err != nil {
		t.Fatalf("Query count failed: %v", err)
	}
	if finalCount != 20 {
		t.Errorf("Expected 20 rows, got %d", finalCount)
	}
}

// TestAtomicAsnameSwap tests that concurrent readers never encounter missing/broken table states.
func TestAtomicAsnameSwap(t *testing.T) {
	testDB := "./testdata/atomic_asn_test.db"
	defer os.Remove(testDB)
	defer os.Remove(testDB + "-wal")
	defer os.Remove(testDB + "-shm")

	writeDB, readDB, err := openSQLitePools(testDB)
	if err != nil {
		t.Fatalf("openSQLitePools failed: %v", err)
	}
	defer writeDB.Close()
	defer readDB.Close()

	if err := initSQLiteSchema(writeDB); err != nil {
		t.Fatalf("initSQLiteSchema failed: %v", err)
	}

	// Seed initial
	initial := &pb.AsnamesRequest{
		AsnNames: []*pb.AsnName{
			{AsNumber: 15169, AsName: "GOOGLE", AsLocale: "US"},
			{AsNumber: 13335, AsName: "CLOUDFLARENET", AsLocale: "US"},
		},
	}
	if _, err := updateASNHelper(context.Background(), initial, writeDB); err != nil {
		t.Fatalf("Seed updateASNHelper failed: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var wg sync.WaitGroup
	readErrors := make(chan error, 100)

	// Launch reader goroutines
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				default:
					resp, err := getAsnameHelper(ctx, &pb.GetAsnameRequest{AsNumber: 15169}, readDB)
					if err != nil {
						if ctx.Err() == nil {
							readErrors <- err
						}
						return
					}
					if !resp.GetExists() || resp.GetAsName() == "" {
						readErrors <- fmt.Errorf("read encountered missing row during swap")
						return
					}
				}
			}
		}()
	}

	// Writer performs swaps
	for i := 0; i < 5; i++ {
		req := &pb.AsnamesRequest{
			AsnNames: []*pb.AsnName{
				{AsNumber: 15169, AsName: fmt.Sprintf("GOOGLE-V%d", i), AsLocale: "US"},
				{AsNumber: 13335, AsName: fmt.Sprintf("CLOUDFLARE-V%d", i), AsLocale: "US"},
				{AsNumber: 2914, AsName: "NTT", AsLocale: "JP"},
			},
		}
		if _, err := updateASNHelper(ctx, req, writeDB); err != nil {
			t.Fatalf("updateASNHelper failed: %v", err)
		}
		time.Sleep(10 * time.Millisecond)
	}

	cancel()
	wg.Wait()
	close(readErrors)

	for err := range readErrors {
		t.Errorf("Reader error during atomic swap: %v", err)
	}
}

// TestContextDeadlineCancellation verifies that context deadlines abort long or hung queries.
func TestContextDeadlineCancellation(t *testing.T) {
	testDB := "./testdata/ctx_cancel_test.db"
	defer os.Remove(testDB)
	defer os.Remove(testDB + "-wal")
	defer os.Remove(testDB + "-shm")

	db, err := sql.Open("sqlite", testDB)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer db.Close()

	if _, err := db.Exec(sqliteSchemaDDL); err != nil {
		t.Fatalf("init schema failed: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Immediately cancelled

	_, err = getPrefixCountHelper(ctx, db)
	if err == nil {
		t.Errorf("Expected error from cancelled context, got nil")
	}
}
