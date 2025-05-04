package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"testing"

	"github.com/golang/protobuf/proto"

	_ "github.com/mattn/go-sqlite3"
	pb "github.com/mellowdrifter/bgp_infrastructure/internal/bgpsql"
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
	stmt, _ := db.Prepare(`INSERT INTO INFO (TIME, V4COUNT, V6COUNT,
		V4TOTAL, V6TOTAL, PEERS_CONFIGURED,PEERS_UP,
		PEERS6_CONFIGURED, PEERS6_UP, V4_24, V4_23, V4_22,
		V4_21, V4_20, V4_19,
		V4_18, V4_17, V4_16, V4_15, V4_14, V4_13, V4_12,
		V4_11, V4_10, V4_09, V4_08, V6_48, V6_47, V6_46,
		V6_45, V6_44, V6_43, V6_42, V6_41, V6_40, V6_39,
		V6_38, V6_37, V6_36, V6_35, V6_34, V6_33, V6_32,
		V6_31, V6_30, V6_29, V6_28, V6_27, V6_26, V6_25,
		V6_24, V6_23, V6_22, V6_21, V6_20, V6_19, V6_18,
		V6_17, V6_16, V6_15, V6_14, V6_13, V6_12, V6_11,
		V6_10, V6_09, V6_08, AS4_LEN, AS6_LEN, AS10_LEN,
		AS4_ONLY, AS6_ONLY, AS_BOTH, LARGEC4, LARGEC6,
		ROAVALIDV4, ROAINVALIDV4, ROAUNKNOWNV4,
		ROAVALIDV6, ROAINVALIDV6, ROAUNKNOWNV6) values (?, ?, ?, ?, ?,
		?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?,
		?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?,
		?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?,
		?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	defer stmt.Close()
	for _, b := range values {
		_, err := stmt.Exec(b.Time, b.V4Count, b.V6Count, b.V4Total, b.V6Total, b.PeersConfigured,
			b.PeersUp, b.Peers6Configured, b.Peers6Up, b.V4_24,
			b.V4_23, b.V4_22, b.V4_21, b.V4_20, b.V4_19, b.V4_18, b.V4_17, b.V4_16,
			b.V4_15, b.V4_14, b.V4_13, b.V4_12, b.V4_11, b.V4_10, b.V4_09, b.V4_08,
			b.V6_48, b.V6_47, b.V6_46, b.V6_45, b.V6_44, b.V6_43, b.V6_42, b.V6_41,
			b.V6_40, b.V6_39, b.V6_38, b.V6_37, b.V6_36, b.V6_35, b.V6_34, b.V6_33,
			b.V6_32, b.V6_31, b.V6_30, b.V6_29, b.V6_28, b.V6_27, b.V6_26, b.V6_25,
			b.V6_24, b.V6_23, b.V6_22, b.V6_21, b.V6_20, b.V6_19, b.V6_18, b.V6_17,
			b.V6_16, b.V6_15, b.V6_14, b.V6_13, b.V6_12, b.V6_11, b.V6_10, b.V6_09,
			b.V6_08, b.As4, b.As6, b.As10, b.As4Only, b.As6Only, b.AsBoth, b.LargeC4,
			b.LargeC6, b.Roavalid4, b.Roainvalid4, b.Roaunknown4, b.Roavalid6,
			b.Roainvalid6, b.Roaunknown6)
		if err != nil {
			log.Fatalln("Error on statement:", err)
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

	db, _ := sql.Open("sqlite3", "./testdata/bgpinfo.db")
	bgpinfoServer.db = db

	want := readOne("latest.pb")
	bgpinfoServer.AddLatest(context.Background(), want)

	var gotStruct com.BgpUpdate

	query := fmt.Sprintf(`SELECT * FROM INFO WHERE TIME = '%d'`, want.GetTime())
	row := db.QueryRow(query)

	err := row.Scan(
		&gotStruct.Time,
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
		log.Fatal(err)
	}

	got := com.StructToProto(&gotStruct)

	if !proto.Equal(got, want) {
		t.Errorf("Error on TestAddLatest. Got %#v, Want %#v", got, want)
	}
}
