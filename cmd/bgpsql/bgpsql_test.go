package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"
	"testing"

	"github.com/golang/protobuf/proto"

	_ "github.com/mattn/go-sqlite3"
	pb "github.com/mellowdrifter/bgp_infrastructure/proto/bgpsql"
	com "github.com/mellowdrifter/bgp_infrastructure/pkg/common"
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
	values := readAnnual("annual.pb")
	for _, b := range values {
		if err := addLatestHelper(b, db); err != nil {
			log.Fatalln("Error on populate addLatestHelper:", err)
		}
	}
}

func createTestDatabase() {
	db, _ := sql.Open("sqlite3", "./testdata/bgpinfo.db")

	tx, _ := db.Begin()
	tx.Exec(`DROP TABLE IF EXISTS INFO`)
	tx.Exec(`DROP TABLE IF EXISTS ASNUMNAME`)
	tx.Exec(`DROP TABLE IF EXISTS ASNUMNAME_NEW`)
	tx.Exec(`CREATE TABLE INFO (
		source VARCHAR(16) NOT NULL DEFAULT 'bgp1',
		TIME int(12) NOT NULL DEFAULT 0,
		V4COUNT int(10) NOT NULL,
		V6COUNT int(7) NOT NULL,
		PEERS_CONFIGURED int(3) DEFAULT NULL,
		PEERS_UP int(3) DEFAULT NULL,
		V4_24 int(10) DEFAULT NULL,
		V4_23 int(10) DEFAULT NULL,
		V4_22 int(10) DEFAULT NULL,
		V4_21 int(10) DEFAULT NULL,
		V4_20 int(10) DEFAULT NULL,
		V4_19 int(10) DEFAULT NULL,
		V4_18 int(10) DEFAULT NULL,
		V4_17 int(10) DEFAULT NULL,
		V4_16 int(10) DEFAULT NULL,
		V4_15 int(10) DEFAULT NULL,
		V4_14 int(10) DEFAULT NULL,
		V4_13 int(10) DEFAULT NULL,
		V4_12 int(10) DEFAULT NULL,
		V4_11 int(10) DEFAULT NULL,
		V4_10 int(10) DEFAULT NULL,
		V4_09 int(10) DEFAULT NULL,
		V4_08 int(10) DEFAULT NULL,
		V6_48 int(7) DEFAULT NULL,
		V6_47 int(7) DEFAULT NULL,
		V6_46 int(7) DEFAULT NULL,
		V6_45 int(7) DEFAULT NULL,
		V6_44 int(7) DEFAULT NULL,
		V6_43 int(7) DEFAULT NULL,
		V6_42 int(7) DEFAULT NULL,
		V6_41 int(7) DEFAULT NULL,
		V6_40 int(7) DEFAULT NULL,
		V6_39 int(7) DEFAULT NULL,
		V6_38 int(7) DEFAULT NULL,
		V6_37 int(7) DEFAULT NULL,
		V6_36 int(7) DEFAULT NULL,
		V6_35 int(7) DEFAULT NULL,
		V6_34 int(7) DEFAULT NULL,
		V6_33 int(7) DEFAULT NULL,
		V6_32 int(7) DEFAULT NULL,
		V6_31 int(7) DEFAULT NULL,
		V6_30 int(7) DEFAULT NULL,
		V6_29 int(7) DEFAULT NULL,
		V6_28 int(7) DEFAULT NULL,
		V6_27 int(7) DEFAULT NULL,
		V6_26 int(7) DEFAULT NULL,
		V6_25 int(7) DEFAULT NULL,
		V6_24 int(7) DEFAULT NULL,
		V6_23 int(7) DEFAULT NULL,
		V6_22 int(7) DEFAULT NULL,
		V6_21 int(7) DEFAULT NULL,
		V6_20 int(7) DEFAULT NULL,
		V6_19 int(7) DEFAULT NULL,
		V6_18 int(7) DEFAULT NULL,
		V6_17 int(7) DEFAULT NULL,
		V6_16 int(7) DEFAULT NULL,
		V6_15 int(7) DEFAULT NULL,
		V6_14 int(7) DEFAULT NULL,
		V6_13 int(7) DEFAULT NULL,
		V6_12 int(7) DEFAULT NULL,
		V6_11 int(7) DEFAULT NULL,
		V6_10 int(7) DEFAULT NULL,
		V6_09 int(7) DEFAULT NULL,
		V6_08 int(7) DEFAULT NULL,
		PEERS6_UP int(3) DEFAULT NULL,
		PEERS6_CONFIGURED int(3) DEFAULT NULL,
		TWEET bit(1) NOT NULL DEFAULT 0,
		V4TOTAL int(12) DEFAULT NULL,
		V6TOTAL int(10) DEFAULT NULL,
		AS4_LEN int(10) DEFAULT NULL,
		AS6_LEN int(10) DEFAULT NULL,
		AS10_LEN int(10) DEFAULT NULL,
		AS4_ONLY int(10) DEFAULT NULL,
		AS6_ONLY int(10) DEFAULT NULL,
		AS_BOTH int(10) DEFAULT NULL,
		LARGEC4 int(6) DEFAULT NULL,
		LARGEC6 int(6) DEFAULT NULL,
		ROAVALIDV4 int(10) DEFAULT NULL,
		ROAINVALIDV4 int(10) DEFAULT NULL,
		ROAUNKNOWNV4 int(10) DEFAULT NULL,
		ROAVALIDV6 int(10) DEFAULT NULL,
		ROAINVALIDV6 int(10) DEFAULT NULL,
		ROAUNKNOWNV6 int(10) DEFAULT NULL,
		quality TEXT NOT NULL DEFAULT 'ok',
		quality_note TEXT DEFAULT NULL,
		ingested_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		PRIMARY KEY (source, TIME)
	)`)
	tx.Exec(`CREATE TABLE ASNUMNAME (
		ASNUMBER INTEGER NOT NULL,
		ASNAME TEXT NOT NULL,
		LOCALE TEXT DEFAULT NULL
	)`)
	if err := tx.Commit(); err != nil {
		log.Panic("Unable to create test database")
	}
}

func TestAddLatest(t *testing.T) {
	createTestDatabase()

	var bgpinfoServer server
	bgpinfoServer.cfg = config{
		defaultSource: "bgp1",
		valCfg: validationConfig{
			v4Min:          700000,
			v4Max:          1600000,
			v6Min:          50000,
			v6Max:          600000,
			maxDeltaPct:    5.0,
			maxPeerDiffPct: 5.0,
			minPeerRatio:   0.4,
		},
	}

	db, _ := sql.Open("sqlite3", "./testdata/bgpinfo.db")
	bgpinfoServer.db = db

	want := readOne("latest.pb")
	alignedTime := want.GetTime() - (want.GetTime() % 300)

	resp, err := bgpinfoServer.AddLatest(context.Background(), want)
	if err != nil {
		t.Fatalf("AddLatest failed: %v", err)
	}
	if !resp.GetSuccess() {
		t.Fatalf("AddLatest returned success=false")
	}

	var gotStruct com.BgpUpdate
	var gotSource string
	var gotQuality string
	var gotQualityNote sql.NullString

	query := fmt.Sprintf(`SELECT source, TIME, quality, quality_note, V4COUNT, V6COUNT, PEERS_CONFIGURED, PEERS_UP,
		V4_24, V4_23, V4_22, V4_21, V4_20, V4_19, V4_18, V4_17, V4_16, V4_15, V4_14, V4_13,
		V4_12, V4_11, V4_10, V4_09, V4_08, V6_48, V6_47, V6_46, V6_45, V6_44, V6_43, V6_42,
		V6_41, V6_40, V6_39, V6_38, V6_37, V6_36, V6_35, V6_34, V6_33, V6_32, V6_31, V6_30,
		V6_29, V6_28, V6_27, V6_26, V6_25, V6_24, V6_23, V6_22, V6_21, V6_20, V6_19, V6_18,
		V6_17, V6_16, V6_15, V6_14, V6_13, V6_12, V6_11, V6_10, V6_09, V6_08, PEERS6_UP,
		PEERS6_CONFIGURED, TWEET, V4TOTAL, V6TOTAL, AS4_LEN, AS6_LEN, AS10_LEN, AS4_ONLY,
		AS6_ONLY, AS_BOTH, LARGEC4, LARGEC6, ROAVALIDV4, ROAINVALIDV4, ROAUNKNOWNV4,
		ROAVALIDV6, ROAINVALIDV6, ROAUNKNOWNV6
		FROM INFO WHERE TIME = '%d'`, alignedTime)

	row := db.QueryRow(query)
	err = row.Scan(
		&gotSource,
		&gotStruct.Time,
		&gotQuality,
		&gotQualityNote,
		&gotStruct.V4Count,
		&gotStruct.V6Count,
		&gotStruct.PeersConfigured,
		&gotStruct.PeersUp,
		&gotStruct.V4_24,
		&gotStruct.V4_23,
		&gotStruct.V4_22,
		&gotStruct.V4_21,
		&gotStruct.V4_20,
		&gotStruct.V4_19,
		&gotStruct.V4_18,
		&gotStruct.V4_17,
		&gotStruct.V4_16,
		&gotStruct.V4_15,
		&gotStruct.V4_14,
		&gotStruct.V4_13,
		&gotStruct.V4_12,
		&gotStruct.V4_11,
		&gotStruct.V4_10,
		&gotStruct.V4_09,
		&gotStruct.V4_08,
		&gotStruct.V6_48,
		&gotStruct.V6_47,
		&gotStruct.V6_46,
		&gotStruct.V6_45,
		&gotStruct.V6_44,
		&gotStruct.V6_43,
		&gotStruct.V6_42,
		&gotStruct.V6_41,
		&gotStruct.V6_40,
		&gotStruct.V6_39,
		&gotStruct.V6_38,
		&gotStruct.V6_37,
		&gotStruct.V6_36,
		&gotStruct.V6_35,
		&gotStruct.V6_34,
		&gotStruct.V6_33,
		&gotStruct.V6_32,
		&gotStruct.V6_31,
		&gotStruct.V6_30,
		&gotStruct.V6_29,
		&gotStruct.V6_28,
		&gotStruct.V6_27,
		&gotStruct.V6_26,
		&gotStruct.V6_25,
		&gotStruct.V6_24,
		&gotStruct.V6_23,
		&gotStruct.V6_22,
		&gotStruct.V6_21,
		&gotStruct.V6_20,
		&gotStruct.V6_19,
		&gotStruct.V6_18,
		&gotStruct.V6_17,
		&gotStruct.V6_16,
		&gotStruct.V6_15,
		&gotStruct.V6_14,
		&gotStruct.V6_13,
		&gotStruct.V6_12,
		&gotStruct.V6_11,
		&gotStruct.V6_10,
		&gotStruct.V6_09,
		&gotStruct.V6_08,
		&gotStruct.Peers6Up,
		&gotStruct.Peers6Configured,
		&gotStruct.Tweet,
		&gotStruct.V4Total,
		&gotStruct.V6Total,
		&gotStruct.As4,
		&gotStruct.As6,
		&gotStruct.As10,
		&gotStruct.As4Only,
		&gotStruct.As6Only,
		&gotStruct.AsBoth,
		&gotStruct.LargeC4,
		&gotStruct.LargeC6,
		&gotStruct.Roavalid4,
		&gotStruct.Roainvalid4,
		&gotStruct.Roaunknown4,
		&gotStruct.Roavalid6,
		&gotStruct.Roainvalid6,
		&gotStruct.Roaunknown6,
	)
	if err != nil {
		t.Fatalf("Row scan error: %v", err)
	}

	if gotSource != "bgp1" {
		t.Errorf("Expected fallback source 'bgp1', got '%s'", gotSource)
	}
	if gotQuality != "ok" {
		t.Errorf("Expected quality 'ok', got '%s' (note: %v)", gotQuality, gotQualityNote.String)
	}
	if gotStruct.Time != alignedTime {
		t.Errorf("Expected aligned time %d, got %d", alignedTime, gotStruct.Time)
	}

	gotStruct.Source = gotSource
	got := com.StructToProto(&gotStruct)

	// Compare with expected proto having aligned time and default source
	expectedProto := proto.Clone(want).(*pb.Values)
	expectedProto.Time = alignedTime
	expectedProto.Source = "bgp1"
	expectedProto.SampleTime = alignedTime

	if !proto.Equal(got, expectedProto) {
		t.Errorf("Error on TestAddLatest values mismatch. Got %#v, Want %#v", got, expectedProto)
	}

	// Test Idempotency: re-inserting the same sample with updated metrics should update in-place without duplicate key error
	wantUpdated := proto.Clone(want).(*pb.Values)
	wantUpdated.PrefixCount.Active_4 = 999999
	wantUpdated.Source = "bgp1"
	wantUpdated.SampleTime = alignedTime

	resp, err = bgpinfoServer.AddLatest(context.Background(), wantUpdated)
	if err != nil {
		t.Fatalf("Idempotent re-insert failed: %v", err)
	}
	if !resp.GetSuccess() {
		t.Fatalf("Idempotent re-insert returned success=false")
	}

	var updatedV4 uint32
	err = db.QueryRow("SELECT V4COUNT FROM INFO WHERE source = 'bgp1' AND TIME = ?", alignedTime).Scan(&updatedV4)
	if err != nil {
		t.Fatalf("Failed to query updated V4COUNT: %v", err)
	}
	if updatedV4 != 999999 {
		t.Errorf("Expected updated V4COUNT 999999, got %d", updatedV4)
	}

	// Test explicit source: inserting with source 'bgp3' at same aligned timestamp
	wantBgp3 := proto.Clone(want).(*pb.Values)
	wantBgp3.Source = "bgp3"
	wantBgp3.SampleTime = alignedTime

	resp, err = bgpinfoServer.AddLatest(context.Background(), wantBgp3)
	if err != nil {
		t.Fatalf("AddLatest with explicit source bgp3 failed: %v", err)
	}
	if !resp.GetSuccess() {
		t.Fatalf("AddLatest with source bgp3 returned success=false")
	}

	var rowCount int
	err = db.QueryRow("SELECT COUNT(*) FROM INFO WHERE TIME = ?", alignedTime).Scan(&rowCount)
	if err != nil {
		t.Fatalf("Failed to query row count: %v", err)
	}
	if rowCount != 2 {
		t.Errorf("Expected 2 rows (bgp1 and bgp3) at TIME=%d, got %d", alignedTime, rowCount)
	}
}

func TestValidationBounds(t *testing.T) {
	createTestDatabase()

	var s server
	s.cfg = config{
		defaultSource: "bgp1",
		valCfg: validationConfig{
			v4Min:          800000,
			v4Max:          1600000,
			v6Min:          150000,
			v6Max:          600000,
			maxDeltaPct:    3.0,
			maxPeerDiffPct: 5.0,
			minPeerRatio:   0.5,
		},
	}
	db, _ := sql.Open("sqlite3", "./testdata/bgpinfo.db")
	s.db = db

	want := readOne("latest.pb")
	// Active_4 in latest.pb is 785490, which is < 800000
	_, err := s.AddLatest(context.Background(), want)
	if err != nil {
		t.Fatalf("AddLatest failed: %v", err)
	}

	alignedTime := want.GetTime() - (want.GetTime() % 300)
	var quality, note string
	err = db.QueryRow("SELECT quality, quality_note FROM INFO WHERE source = 'bgp1' AND TIME = ?", alignedTime).Scan(&quality, &note)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}

	if quality != "suspect" {
		t.Errorf("Expected quality 'suspect' for v4 count < 800000, got '%s'", quality)
	}
	if !strings.Contains(note, "outside bounds") {
		t.Errorf("Expected note to contain 'outside bounds', got: '%s'", note)
	}
}

func TestValidationPeers(t *testing.T) {
	createTestDatabase()

	var s server
	s.cfg = config{
		defaultSource: "bgp1",
		valCfg: validationConfig{
			v4Min:          700000,
			v4Max:          1600000,
			v6Min:          50000,
			v6Max:          600000,
			maxDeltaPct:    5.0,
			maxPeerDiffPct: 5.0,
			minPeerRatio:   0.5,
		},
	}
	db, _ := sql.Open("sqlite3", "./testdata/bgpinfo.db")
	s.db = db

	sample := readOne("latest.pb")
	sample.Peers.PeerCount_4 = 10
	sample.Peers.PeerUp_4 = 2 // only 2 of 10 peers up (20% < 50%)

	_, err := s.AddLatest(context.Background(), sample)
	if err != nil {
		t.Fatalf("AddLatest failed: %v", err)
	}

	alignedTime := sample.GetTime() - (sample.GetTime() % 300)
	var quality, note string
	err = db.QueryRow("SELECT quality, quality_note FROM INFO WHERE source = 'bgp1' AND TIME = ?", alignedTime).Scan(&quality, &note)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}

	if quality != "suspect" {
		t.Errorf("Expected quality 'suspect' for degraded peers, got '%s'", quality)
	}
	if !strings.Contains(note, "peers degraded") {
		t.Errorf("Expected note to mention 'peers degraded', got: '%s'", note)
	}
}

func TestValidationDeltaAndConsensus(t *testing.T) {
	createTestDatabase()

	var s server
	s.cfg = config{
		defaultSource: "bgp1",
		valCfg: validationConfig{
			v4Min:          700000,
			v4Max:          1600000,
			v6Min:          50000,
			v6Max:          600000,
			maxDeltaPct:    3.0,
			maxPeerDiffPct: 5.0,
			minPeerRatio:   0.5,
		},
	}
	db, _ := sql.Open("sqlite3", "./testdata/bgpinfo.db")
	s.db = db

	// T0: Baseline sample
	t0 := uint64(1700000100)
	s0 := readOne("latest.pb")
	s0.Time = t0
	s0.Source = "bgp1"
	s0.PrefixCount.Active_4 = 1000000
	_, err := s.AddLatest(context.Background(), s0)
	if err != nil {
		t.Fatalf("AddLatest s0 failed: %v", err)
	}

	// T1: 5 minutes later, v4 drops by 10% (1,000,000 -> 900,000)
	t1 := t0 + 300
	s1 := readOne("latest.pb")
	s1.Time = t1
	s1.Source = "bgp1"
	s1.PrefixCount.Active_4 = 900000
	_, err = s.AddLatest(context.Background(), s1)
	if err != nil {
		t.Fatalf("AddLatest s1 failed: %v", err)
	}

	alignedT1 := t1 - (t1 % 300)
	var quality, note string
	err = db.QueryRow("SELECT quality, quality_note FROM INFO WHERE source = 'bgp1' AND TIME = ?", alignedT1).Scan(&quality, &note)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}

	if quality != "suspect" {
		t.Errorf("Expected quality 'suspect' for 10%% drop, got '%s'", quality)
	}
	if !strings.Contains(note, "v4 shifted") {
		t.Errorf("Expected note to mention 'v4 shifted', got: '%s'", note)
	}
}
