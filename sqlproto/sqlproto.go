package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path"

	_ "github.com/go-sql-driver/mysql"
	"github.com/golang/protobuf/proto"
	com "github.com/mellowdrifter/bgp_infrastructure/common"
	pb "github.com/mellowdrifter/bgp_infrastructure/proto/bgpinfo"
	"gopkg.in/ini.v1"
)

func main() {

	// Open database connection.
	exe, err := os.Executable()
	if err != nil {
		log.Fatal(err)
	}
	path := fmt.Sprintf("%s/config.ini", path.Dir(exe))
	cf, err := ini.Load(path)
	if err != nil {
		log.Fatalf("failed to read config file: %v\n", err)
	}
	dbname := cf.Section("sql").Key("database").String()
	user := cf.Section("sql").Key("username").String()
	pass := cf.Section("sql").Key("password").String()
	sqlserver := fmt.Sprintf("%s:%s@tcp(127.0.0.1:3306)/%s", user, pass, dbname)
	db, err := sql.Open("mysql", sqlserver)
	if err != nil {
		log.Fatalf("can't open database. Got %v", err)
	}
	defer db.Close()

	latest(db)
	annual(db)

}

func latest(db *sql.DB) {
	f, err := os.Create("latest.pb")
	if err != nil {
		log.Panic(err)
	}

	var b com.BgpUpdate

	// Latest data
	query := `SELECT TIME, V4COUNT, V6COUNT, PEERS_CONFIGURED, PEERS_UP, V4_24, V4_23,
	V4_22, V4_21, V4_20, V4_19, V4_18, V4_17, V4_16, V4_15, V4_14, V4_13, V4_12, V4_11,
	V4_10, V4_09, V4_08, V6_48, V6_47, V6_46, V6_45, V6_44, V6_43, V6_42, V6_41, V6_40,
	V6_39, V6_38, V6_37, V6_36, V6_35, V6_34, V6_33, V6_32, V6_31, V6_30, V6_29, V6_28,
	V6_27, V6_26, V6_25, V6_24, V6_23, V6_22, V6_21, V6_20, V6_19, V6_18, V6_17, V6_16,
	V6_15, V6_14, V6_13, V6_12, V6_11, V6_10, V6_09, V6_08, PEERS6_UP, PEERS6_CONFIGURED,
	V4TOTAL, V6TOTAL, AS4_LEN, AS6_LEN, AS10_LEN, AS4_ONLY, AS6_ONLY, AS_BOTH, LARGEC4,
	LARGEC6, ROAVALIDV4, ROAINVALIDV4, ROAUNKNOWNV4, ROAVALIDV6, ROAINVALIDV6, ROAUNKNOWNV6
	FROM INFO ORDER BY TIME DESC LIMIT 1`
	row := db.QueryRow(query)
	row.Scan(
		&b.Time, &b.V4Count, &b.V6Count, &b.PeersConfigured,
		&b.PeersUp, &b.V4_24,
		&b.V4_23, &b.V4_22, &b.V4_21, &b.V4_20, &b.V4_19, &b.V4_18, &b.V4_17, &b.V4_16,
		&b.V4_15, &b.V4_14, &b.V4_13, &b.V4_12, &b.V4_11, &b.V4_10, &b.V4_09, &b.V4_08,
		&b.V6_48, &b.V6_47, &b.V6_46, &b.V6_45, &b.V6_44, &b.V6_43, &b.V6_42, &b.V6_41,
		&b.V6_40, &b.V6_39, &b.V6_38, &b.V6_37, &b.V6_36, &b.V6_35, &b.V6_34, &b.V6_33,
		&b.V6_32, &b.V6_31, &b.V6_30, &b.V6_29, &b.V6_28, &b.V6_27, &b.V6_26, &b.V6_25,
		&b.V6_24, &b.V6_23, &b.V6_22, &b.V6_21, &b.V6_20, &b.V6_19, &b.V6_18, &b.V6_17,
		&b.V6_16, &b.V6_15, &b.V6_14, &b.V6_13, &b.V6_12, &b.V6_11, &b.V6_10, &b.V6_09,
		&b.V6_08, &b.Peers6Up, &b.Peers6Configured, &b.V4Total, &b.V6Total,
		&b.As4, &b.As6, &b.As10, &b.As4Only, &b.As6Only, &b.AsBoth, &b.LargeC4,
		&b.LargeC6, &b.Roavalid4, &b.Roainvalid4, &b.Roaunknown4, &b.Roavalid6,
		&b.Roainvalid6, &b.Roaunknown6)

	values := com.StructToProto(&b)
	fmt.Println(proto.MarshalTextString(values))
	f.WriteString(proto.MarshalTextString(values))
}

func annual(db *sql.DB) {
	var updates []*pb.Values
	f, err := os.Create("annual.pb")
	if err != nil {
		log.Panic(err)
	}

	rows, err := db.Query(`SELECT TIME, V4COUNT, V6COUNT, PEERS_CONFIGURED, PEERS_UP, V4_24, V4_23,
	V4_22, V4_21, V4_20, V4_19, V4_18, V4_17, V4_16, V4_15, V4_14, V4_13, V4_12, V4_11,
	V4_10, V4_09, V4_08, V6_48, V6_47, V6_46, V6_45, V6_44, V6_43, V6_42, V6_41, V6_40,
	V6_39, V6_38, V6_37, V6_36, V6_35, V6_34, V6_33, V6_32, V6_31, V6_30, V6_29, V6_28,
	V6_27, V6_26, V6_25, V6_24, V6_23, V6_22, V6_21, V6_20, V6_19, V6_18, V6_17, V6_16,
	V6_15, V6_14, V6_13, V6_12, V6_11, V6_10, V6_09, V6_08, PEERS6_UP, PEERS6_CONFIGURED,
	V4TOTAL, V6TOTAL, AS4_LEN, AS6_LEN, AS10_LEN, AS4_ONLY, AS6_ONLY, AS_BOTH, LARGEC4,
	LARGEC6, ROAVALIDV4, ROAINVALIDV4, ROAUNKNOWNV4, ROAVALIDV6, ROAINVALIDV6, ROAUNKNOWNV6
	FROM INFO WHERE TIME >= 1562938371`)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var b com.BgpUpdate
		err := rows.Scan(
			&b.Time, &b.V4Count, &b.V6Count, &b.PeersConfigured,
			&b.PeersUp, &b.V4_24,
			&b.V4_23, &b.V4_22, &b.V4_21, &b.V4_20, &b.V4_19, &b.V4_18, &b.V4_17, &b.V4_16,
			&b.V4_15, &b.V4_14, &b.V4_13, &b.V4_12, &b.V4_11, &b.V4_10, &b.V4_09, &b.V4_08,
			&b.V6_48, &b.V6_47, &b.V6_46, &b.V6_45, &b.V6_44, &b.V6_43, &b.V6_42, &b.V6_41,
			&b.V6_40, &b.V6_39, &b.V6_38, &b.V6_37, &b.V6_36, &b.V6_35, &b.V6_34, &b.V6_33,
			&b.V6_32, &b.V6_31, &b.V6_30, &b.V6_29, &b.V6_28, &b.V6_27, &b.V6_26, &b.V6_25,
			&b.V6_24, &b.V6_23, &b.V6_22, &b.V6_21, &b.V6_20, &b.V6_19, &b.V6_18, &b.V6_17,
			&b.V6_16, &b.V6_15, &b.V6_14, &b.V6_13, &b.V6_12, &b.V6_11, &b.V6_10, &b.V6_09,
			&b.V6_08, &b.Peers6Up, &b.Peers6Configured, &b.V4Total, &b.V6Total,
			&b.As4, &b.As6, &b.As10, &b.As4Only, &b.As6Only, &b.AsBoth, &b.LargeC4,
			&b.LargeC6, &b.Roavalid4, &b.Roainvalid4, &b.Roaunknown4, &b.Roavalid6,
			&b.Roainvalid6, &b.Roaunknown6)
		if err != nil {
			log.Panic(err)
		}
		updates = append(updates, com.StructToProto(&b))

	}

	var largeUpdates []*pb.Values
	for i := 1; i <= 11; i++ {
		largeUpdates = append(largeUpdates, updates...)
	}
	var allValues = pb.ListOfValues{
		Values: largeUpdates,
	}
	fmt.Printf("Pulled %d events for annual\n", len(allValues.GetValues()))
	f.WriteString(proto.MarshalTextString(&allValues))

}
