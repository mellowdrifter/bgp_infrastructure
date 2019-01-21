package main

import (
	"fmt"
	"log"
	"time"

	_ "github.com/go-sql-driver/mysql"
	pb "github.com/mellowdrifter/bgp_infrastructure/proto/bgpinfo"
)

func query() {

	bgpInfo := bgpStat{}
	err := db.QueryRow(`select TIME, V4COUNT, V6COUNT, V4TOTAL, V6TOTAL, PEERS_CONFIGURED,
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

func add(b *bgpUpdate) error {
	// fmt.Printf("Update is %+v\n", b)
	// All the required info. Fields can be added/deleted in future
	result, err := db.Exec(
		`INSERT INTO INFO (TIME, V4COUNT, V6COUNT, V4TOTAL, V6TOTAL, PEERS_CONFIGURED,
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
		AS6_LEN, AS10_LEN, AS4_ONLY, AS6_ONLY, AS_BOTH,
		LARGEC4, LARGEC6)

		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,
				?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,
				?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,
				?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,
				?,?,?)`,
		b.time, b.v4Count, b.v6Count, b.v4Total, b.v6Total, b.peersConfigured,
		b.peersUp, b.peers6Configured, b.peers6Up, b.v4_24,
		b.v4_23, b.v4_22, b.v4_21, b.v4_20, b.v4_19, b.v4_18, b.v4_17, b.v4_16,
		b.v4_15, b.v4_14, b.v4_13, b.v4_12, b.v4_11, b.v4_10, b.v4_09, b.v4_08,
		b.v6_48, b.v6_47, b.v6_46, b.v6_45, b.v6_44, b.v6_43, b.v6_42, b.v6_41,
		b.v6_40, b.v6_39, b.v6_38, b.v6_37, b.v6_36, b.v6_35, b.v6_34, b.v6_33,
		b.v6_32, b.v6_31, b.v6_30, b.v6_29, b.v6_28, b.v6_27, b.v6_26, b.v6_25,
		b.v6_24, b.v6_23, b.v6_22, b.v6_21, b.v6_20, b.v6_19, b.v6_18, b.v6_17,
		b.v6_16, b.v6_15, b.v6_14, b.v6_13, b.v6_12, b.v6_11, b.v6_10, b.v6_09,
		b.v6_08, b.memTable, b.memTotal, b.memProto, b.memAttr, b.memTable6,
		b.memTotal6, b.memProto6, b.memAttr6, b.as4, b.as6, b.as10, b.as4Only,
		b.as6Only, b.asBoth, b.largeC4, b.largeC6)

	log.Printf("updated database: %v", result)

	if err != nil {
		return fmt.Errorf("Unable to update database: %v", err)
	}
	return nil
}

func getCounts() (*pb.Counts, error) {
	var counts pb.Counts
	sq1 := `SELECT TIME, V4COUNT, V6COUNT FROM INFO ORDER BY TIME DESC LIMIT 1`
	err := db.QueryRow(sq1).Scan(
		&counts.Time,
		&counts.Currentv4,
		&counts.Currentv6,
	)
	if err != nil {
		return nil, fmt.Errorf("Can't extract information. Got %v", err)
	}

	sq2 := `SELECT V4COUNT, V6COUNT FROM INFO WHERE TWEET IS NOT NULL
			ORDER BY TIME DESC LIMIT 1`
	err = db.QueryRow(sq2).Scan(
		&counts.Sixhoursv4,
		&counts.Sixhoursv6,
	)
	if err != nil {
		return nil, fmt.Errorf("Can't extract information. Got %v", err)
	}

	lastWeek := int32(time.Now().Unix()) - 604800
	sq3 := fmt.Sprintf(`SELECT V4COUNT, V6COUNT FROM INFO WHERE TWEET IS NOT NULL
				AND TIME < '%d' ORDER BY TIME DESC LIMIT 1`, lastWeek)
	err = db.QueryRow(sq3).Scan(
		&counts.Weekagov4,
		&counts.Weekagov6,
	)
	if err != nil {
		return nil, fmt.Errorf("Can't extract information. Got %v", err)
	}

	sq4 := `SELECT V4_24, V6_48 FROM INFO ORDER BY TIME DESC LIMIT 1`
	err = db.QueryRow(sq4).Scan(
		&counts.Slash24,
		&counts.Slash48,
	)
	if err != nil {
		return nil, fmt.Errorf("Can't extract information. Got %v", err)
	}

	return &counts, nil
}

func getGraph(period *pb.Length) (*pb.GraphData, error) {
	var startTime int32
	graphData := &pb.GraphData{}
	lastNight := int32(time.Now().Unix() - 66600)
	switch period.GetTime() {
	case pb.TimeLength_WEEK:
		startTime = lastNight - 604800
	case pb.TimeLength_MONTH:
		startTime = lastNight - 2628000
	case pb.TimeLength_SIXMONTH:
		startTime = lastNight - 15768000
	case pb.TimeLength_YEAR:
		startTime = lastNight - 31536000
	}
	sq := fmt.Sprintf(`SELECT TIME, V4COUNT, V6COUNT FROM INFO WHERE TIME >= '%d'
						&& TIME <= '%d'`, startTime, lastNight)

	rows, err := db.Query(sq)
	if err != nil {
		return nil, fmt.Errorf("Can't extract information. Got %v", err)
	}
	defer rows.Close()
	for rows.Next() {
		var t46 pb.TimeV4V6
		err = rows.Scan(
			&t46.Time,
			&t46.V4,
			&t46.V6,
		)
		if err != nil {
			return nil, fmt.Errorf("Can't extract information. Got %v", err)
		}
		graphData.Tick = append(graphData.Tick, &t46)
	}
	// return that list of messages
	return graphData, nil
}

func getMasks() (*pb.Masks, error) {
	masks := pb.Masks{}
	sq := `SELECT V4_08,V4_09,V4_10,V4_11,V4_12,V4_13,V4_14,
			V4_15,V4_16,V4_17,V4_18,V4_19,V4_20,V4_21,V4_22,
			V4_23,V4_24,V4COUNT,V6_48,V6_47,V6_46,V6_45,V6_44,
			V6_43,V6_42,V6_41,V6_40,V6_39,V6_38,V6_37,V6_36,
			V6_35,V6_34,V6_33,V6_32,V6_31,V6_30,V6_29,V6_28,
			V6_27,V6_26,V6_25,V6_24,V6_23,V6_22,V6_21,V6_20,
			V6_19,V6_18,V6_17,V6_16,V6_15,V6_14,V6_13,V6_12,
			V6_11,V6_10,V6_09,V6_08,V6COUNT, TIME FROM INFO
			ORDER BY TIME DESC LIMIT 1`

	// Need to hold v4 and v6 total. Need to add v6 here as well
	// Also what to do with the time?
	err := db.QueryRow(sq).Scan(
		&masks.V4_08,
		&masks.V4_09,
		&masks.V4_10,
		&masks.V4_11,
		&masks.V4_12,
		&masks.V4_13,
		&masks.V4_14,
		&masks.V4_15,
		&masks.V4_16,
		&masks.V4_17,
		&masks.V4_18,
		&masks.V4_19,
		&masks.V4_20,
		&masks.V4_21,
		&masks.V4_22,
		&masks.V4_23,
		&masks.V4_24,
	)
	if err != nil {
		return nil, fmt.Errorf("Can't extract information. Got %v", err)
	}

	return &masks, nil

}

func setTweetBit(t *pb.TimeV4V6) error {
	u := fmt.Sprintf(`UPDATE INFO SET TWEET = 1 WHERE TIME = %d`, t.GetTime())
	log.Printf(u)
	_, err := db.Exec(u)
	if err != nil {
		return err
	}
	return nil
}
