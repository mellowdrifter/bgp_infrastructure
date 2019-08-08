package main

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	pb "github.com/mellowdrifter/bgp_infrastructure/proto/bgpinfo"
)

// TODO: Insert a year's worh of csvs in here to test all manner of things. Or maybe have a separate 'add' to only add what's required.
// Though then I'd need a separate csv for each test. Not the end of the world.
func createLocalDatabase() {
	database, _ := sql.Open("sqlite3", "./bgpinfo.db")

	statement, _ := database.Prepare(`DROP TABLE IF EXISTS INFO`)
	statement.Exec()

	statement, _ = database.Prepare(`CREATE TABLE INFO (
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
		TWEET bit(1) DEFAULT NULL,
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
		PRIMARY KEY (TIME)
		)`)
	statement.Exec()

	statement, _ = database.Prepare("INSERT INTO INFO (PEERS_CONFIGURED, PEERS_UP) VALUES (?, ?)")
	statement.Exec("5", "5")

	rows, _ := database.Query("SELECT PEERS_CONFIGURED, PEERS_UP FROM INFO")
	var peers1 string
	var peers2 string
	for rows.Next() {
		rows.Scan(&peers1, &peers2)
		fmt.Println(": " + peers1 + " " + peers2)
	}

}

//func (s *server) AddLatest(ctx context.Context, v *pb.Values) (*pb.Result, error) {
func TestAddLatest(t *testing.T) {
	createLocalDatabase()

	var bgpinfoServer server

	database, _ := sql.Open("sqlite3", "./bgpinfo.db")
	bgpinfoServer.db = database

	bgpinfoServer.AddLatest(context.Background(), &pb.Values{
		Time: uint64(time.Now().UnixNano()),
		PrefixCount: &pb.PrefixCount{
			Total_4:  uint32(100),
			Active_4: uint32(10),
			Total_6:  uint32(100),
			Active_6: uint32(10),
			Time:     uint64(time.Now().UnixNano()),
		},
		Peers: &pb.PeerCount{
			PeerCount_4: uint32(10),
			PeerUp_4:    uint32(9),
			PeerCount_6: uint32(10),
			PeerUp_6:    uint32(9),
		},
		AsCount: &pb.AsCount{
			As4:     uint32(200),
			As6:     uint32(200),
			As10:    uint32(200),
			As4Only: uint32(200),
			As6Only: uint32(200),
			AsBoth:  uint32(200),
		},
		Masks: &pb.Masks{
			V4_08: uint32(8),
			V4_22: uint32(22),
			V6_08: uint32(8),
			V6_22: uint32(22),
		},
		LargeCommunity: &pb.LargeCommunity{
			C4: uint32(22),
			C6: uint32(23),
		},
		Roas: &pb.Roas{
			V4Valid: uint32(123),
		},
	})

	res, _ := bgpinfoServer.GetPrefixCount(context.Background(), &pb.Empty{})

	if res.GetActive_4() != 10 {
		t.Errorf("Retrived result was incorrect. got: %d, want: 300", res.GetActive_4())
	}

}

/* func (s *server) AddLatest(ctx context.Context, v *pb.Values) (*pb.Result, error) {
	// Receive the latest BGP info updates and add to the database
	log.Println("Received an update")
	log.Println(proto.MarshalTextString(v))

	// get correct struct
	update := repack(v)

	// update database
	err := add(update, &s.db)
	if err != nil {
		log.Printf("Got error in AddLatest: %s\n", err)
		return &pb.Result{}, err
	}

	return &pb.Result{
		Success: true,
	}, nil
} */
