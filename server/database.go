package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/go-sql-driver/mysql"
)

// A struct to hold the AS information altogether
type bgpStat struct {
	time              uint64
	v4Count, v6Count  uint32
	peersConfigured   uint8
	peers6Configured  uint8
	peersUp, peers6Up uint8
	v4Total, v6Total  uint32
}

type bgpUpdate struct {
	time                              uint64
	v4Count, v6Count                  uint32
	v4Total, v6Total                  uint32
	peersConfigured                   uint8
	peers6Configured                  uint8
	peersUp, peers6Up                 uint8
	tweet                             bool
	as4, as6, as10                    uint32
	as4Only, as6Only                  uint32
	asBoth                            uint32
	largeC4, largeC6                  uint32
	memTable, memTotal                string
	memProto, memAttr                 string
	memTable6, memTotal6              string
	memProto6, memAttr6               string
	v4_23, v4_22, v4_21, v4_20, v4_19 uint32
	v4_18, v4_17, v4_16, v4_15, v4_14 uint32
	v4_13, v4_12, v4_11, v4_10, v4_09 uint32
	v4_08, v6_48, v6_47, v6_46, v6_45 uint32
	v6_44, v6_43, v6_42, v6_41, v6_40 uint32
	v6_39, v6_38, v6_37, v6_36, v6_35 uint32
	v6_34, v6_33, v6_32, v6_31, v6_30 uint32
	v6_29, v6_28, v6_27, v6_26, v6_25 uint32
	v6_24, v6_23, v6_22, v6_21, v6_20 uint32
	v6_19, v6_18, v6_17, v6_16, v6_15 uint32
	v6_14, v6_13, v6_12, v6_11, v6_10 uint32
	v6_09, v6_08, v4_24               uint32
}

func query() {
	// Create sql handle
	db, err := sql.Open("mysql",
		"bgpinfo:testpassword@tcp(127.0.0.1:3306)/BGP_STATISTICS")
	if err != nil {
		log.Fatalf("Can't open database. Got %v", err)
	}

	defer db.Close()

	bgpInfo := bgpStat{}
	err = db.QueryRow(`select TIME, V4COUNT, V6COUNT, V4TOTAL, V6TOTAL, PEERS_CONFIGURED,
		PEERS6_CONFIGURED, PEERS_UP, PEERS6_UP
		from INFO ORDER by TIME DESC limit 1`).Scan(
		&bgpInfo.time,
		&bgpInfo.v4Count,
		&bgpInfo.v6Count,
		&bgpInfo.v4Total,
		&bgpInfo.v6Total,
		&bgpInfo.peersConfigured,
		&bgpInfo.peers6Configured,
		&bgpInfo.peersUp,
		&bgpInfo.peers6Up,
	)
	if err != nil {
		log.Fatalf("Can't extract information. Got %v", err)
	}

	fmt.Printf("%+v\n", bgpInfo)
}

func add(b bgpUpdate) (sql.Result, error) {
	// Create sql handle
	db, err := sql.Open("mysql",
		"bgpinfo:testpassword@tcp(127.0.0.1:3306)/BGP_STATISTICS")
	if err != nil {
		log.Fatalf("Can't open database. Got %v", err)
	}
	defer db.Close()

	// Lot's of info...
	result, err := db.Exec(`INSERT INTO INFO (TIME, V4COUNT, V6COUNT, V4TOTAL, V6TOTAL, PEERS_CONFIGURED,
							PEERS_UP, PEERS6_CONFIGURED, PEERS6_UP, V4_24,
							V4_23, V4_22, V4_21, V4_20, V4_19,
							V4_18, V4_17, V4_16, V4_15, V4_14, V4_13, V4_12,
							V4_11, V4_10, V4_09, V4_08, V6_48, V6_47, V6_46,
							V6_45, V6_44, V6_43, V6_42, V6_41, V6_40, V6_39,
							V6_38, V6_37, V6_36, V6_35, V6_34, V6_33, V6_32,
							V6_31, V6_30, V6_29, V6_28, V6_27, V6_26, V6_25,
							V6_24, V6_23, V6_22, V6_21, V6_20, V6_19, V6_18,
							V6_17, V6_16, V6_15, V6_14, V6_13, V6_12, V6_11,
							V6_10, V6_09, V6_08, MEMTABLES, MEMTOTAL,
							MEMPROTOCOLS, MEMATTR, MEMTABLES6,
							MEMTOTAL6, MEMPROTOCOLS6, MEMATTR6, AS4_LEN,
							AS6_LEN, AS10_LEN, AS4_ONLY, AS6_ONLY, AS_BOTH)

							VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9,
							$10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21,
							$22, $23, $24, $25, $26, $27, $28, $29, $30, $31, $32, $33,
							$34, $35, $36, $37, $38, $39, $40, $41, $42, $43, $44, $45,
							$46, $47, $48, $49, $50, $51, $52, $53, $54, $55, $56, $57,
							$58, $59, $60, $61, $62, $63, $64, $65, $66, $67, $68, $69,
							$70, $71, $72, $73, $74, $75, $76, $77, $78, $79, $80, $81)`,
		b.time, b.v4Count, b.v6Count, b.v4Total, b.v6Total, b.peersConfigured,
		b.peersUp, b.peers6Configured, b.peers6Configured, b.peers6Up, b.v4_24,
		b.v4_23, b.v4_22, b.v4_21, b.v4_20, b.v4_19, b.v4_18, b.v4_17, b.v4_16,
		b.v4_15, b.v4_14, b.v4_13, b.v4_12, b.v4_11, b.v4_10, b.v4_09, b.v4_08,
		b.v6_48, b.v6_47, b.v6_46, b.v6_45, b.v6_44, b.v6_43, b.v6_42, b.v6_41,
		b.v6_40, b.v6_39, b.v6_38, b.v6_37, b.v6_36, b.v6_35, b.v6_34, b.v6_33,
		b.v6_32, b.v6_31, b.v6_30, b.v6_29, b.v6_28, b.v6_27, b.v6_26, b.v6_25,
		b.v6_24, b.v6_23, b.v6_22, b.v6_21, b.v6_20, b.v6_19, b.v6_18, b.v6_17,
		b.v6_16, b.v6_15, b.v6_14, b.v6_13, b.v6_12, b.v6_11, b.v6_10, b.v6_09,
		b.v6_08, b.memTable, b.memTotal, b.memProto, b.memAttr, b.memTable6,
		b.memTotal6, b.memProto6, b.memAttr6, b.as4, b.as6, b.as10, b.as4Only,
		b.as6Only, b.asBoth)

	if err != nil {
		return result, fmt.Errorf("Unable to update database: %v", err)
	}
	return result, nil
}
