package main

import (
	"fmt"
	"log"
	"strconv"
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
		V6_10, V6_09, V6_08, AS4_LEN, AS6_LEN, AS10_LEN,
		AS4_ONLY, AS6_ONLY, AS_BOTH, LARGEC4, LARGEC6,
		ROAVALIDV4, ROAINVALIDV4, ROAUNKNOWNV4,
		ROAVALIDV6, ROAINVALIDV6, ROAUNKNOWNV6)

		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,
				?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,
				?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,
				?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		b.time, b.v4Count, b.v6Count, b.v4Total, b.v6Total, b.peersConfigured,
		b.peersUp, b.peers6Configured, b.peers6Up, b.v4_24,
		b.v4_23, b.v4_22, b.v4_21, b.v4_20, b.v4_19, b.v4_18, b.v4_17, b.v4_16,
		b.v4_15, b.v4_14, b.v4_13, b.v4_12, b.v4_11, b.v4_10, b.v4_09, b.v4_08,
		b.v6_48, b.v6_47, b.v6_46, b.v6_45, b.v6_44, b.v6_43, b.v6_42, b.v6_41,
		b.v6_40, b.v6_39, b.v6_38, b.v6_37, b.v6_36, b.v6_35, b.v6_34, b.v6_33,
		b.v6_32, b.v6_31, b.v6_30, b.v6_29, b.v6_28, b.v6_27, b.v6_26, b.v6_25,
		b.v6_24, b.v6_23, b.v6_22, b.v6_21, b.v6_20, b.v6_19, b.v6_18, b.v6_17,
		b.v6_16, b.v6_15, b.v6_14, b.v6_13, b.v6_12, b.v6_11, b.v6_10, b.v6_09,
		b.v6_08, b.as4, b.as6, b.as10, b.as4Only, b.as6Only, b.asBoth, b.largeC4,
		b.largeC6, b.roavalid4, b.roainvalid4, b.roaunknown4, b.roavalid6,
		b.roainvalid6, b.roaunknown6)

	log.Printf("updated database: %v", result)

	if err != nil {
		return fmt.Errorf("Unable to update database: %v", err)
	}
	return nil
}

func getPrefixCountHelper() (*pb.PrefixCountResponse, error) {
	var data pb.PrefixCountResponse

	// Latest data
	sq1 := `SELECT TIME, V4COUNT, V6COUNT FROM INFO ORDER BY TIME DESC LIMIT 1`
	err := db.QueryRow(sq1).Scan(
		&data.Time,
		&data.Active_4,
		&data.Active_6,
	)
	if err != nil {
		return nil, err
	}

	// Six hours ago (last tweeted data)
	sq2 := `SELECT V4COUNT, V6COUNT FROM INFO WHERE TWEET IS NOT NULL
			ORDER BY TIME DESC LIMIT 1`
	err = db.QueryRow(sq2).Scan(
		&data.Sixhoursv4,
		&data.Sixhoursv6,
	)
	if err != nil {
		return nil, err
	}

	// Last weeks numbers
	lastWeek := int32(time.Now().Unix()) - 604800
	sq3 := fmt.Sprintf(`SELECT V4COUNT, V6COUNT FROM INFO WHERE TWEET IS NOT NULL
				AND TIME < '%d' ORDER BY TIME DESC LIMIT 1`, lastWeek)
	err = db.QueryRow(sq3).Scan(
		&data.Weekagov4,
		&data.Weekagov6,
	)
	if err != nil {
		return nil, err
	}

	// /24 and /48 counts
	sq4 := `SELECT V4_24, V6_48 FROM INFO ORDER BY TIME DESC LIMIT 1`
	err = db.QueryRow(sq4).Scan(
		&data.Slash24,
		&data.Slash48,
	)
	if err != nil {
		return nil, err
	}

	return &data, nil
}

func getPieSubnetsHelper() (*pb.PieSubnetsResponse, error) {

	var masks pb.Masks
	var pie pb.PieSubnetsResponse

	err := db.QueryRow(`SELECT V4_08,V4_09,V4_10,V4_11,V4_12,V4_13,V4_14,
        V4_15,V4_16,V4_17,V4_18,V4_19,V4_20,V4_21,V4_22,
        V4_23,V4_24,V4COUNT,V6_48,V6_47,V6_46,V6_45,V6_44,
        V6_43,V6_42,V6_41,V6_40,V6_39,V6_38,V6_37,V6_36,
        V6_35,V6_34,V6_33,V6_32,V6_31,V6_30,V6_29,V6_28,
        V6_27,V6_26,V6_25,V6_24,V6_23,V6_22,V6_21,V6_20,
        V6_19,V6_18,V6_17,V6_16,V6_15,V6_14,V6_13,V6_12,
		V6_11,V6_10,V6_09,V6_08,V6COUNT,
        TIME FROM INFO ORDER BY TIME DESC LIMIT 1`).Scan(
		&masks.V4_08, &masks.V4_09, &masks.V4_10,
		&masks.V4_11, &masks.V4_12, &masks.V4_13,
		&masks.V4_14, &masks.V4_15, &masks.V4_16,
		&masks.V4_17, &masks.V4_18, &masks.V4_19,
		&masks.V4_20, &masks.V4_21, &masks.V4_22,
		&masks.V4_23, &masks.V4_24, &pie.V4Total,
		&masks.V6_48, &masks.V6_47, &masks.V6_46,
		&masks.V6_45, &masks.V6_44, &masks.V6_43,
		&masks.V6_42, &masks.V6_41, &masks.V6_40,
		&masks.V6_39, &masks.V6_38, &masks.V6_37,
		&masks.V6_36, &masks.V6_35, &masks.V6_34,
		&masks.V6_33, &masks.V6_32, &masks.V6_31,
		&masks.V6_30, &masks.V6_29, &masks.V6_28,
		&masks.V6_27, &masks.V6_26, &masks.V6_25,
		&masks.V6_24, &masks.V6_23, &masks.V6_22,
		&masks.V6_21, &masks.V6_20, &masks.V6_19,
		&masks.V6_18, &masks.V6_17, &masks.V6_16,
		&masks.V6_15, &masks.V6_14, &masks.V6_13,
		&masks.V6_12, &masks.V6_11, &masks.V6_10,
		&masks.V6_09, &masks.V6_08, &pie.V6Total,
		&pie.Time,
	)
	if err != nil {
		return nil, err
	}

	// Add masks to the pie response.
	pie.Masks = &masks

	return &pie, nil

}

func getMovementTotalsHelper(m *pb.MovementRequest) (*pb.MovementTotalsResponse, error) {
	// time helpers
	secondsInWeek := 604800
	secondsInMonth := 2628000
	secondsIn6Months := secondsInMonth * 6
	secondsInYear := secondsIn6Months * 2
	end := int(time.Now().Unix() - 66600)

	var start string
	var denomiator int
	switch m.GetPeriod() {
	case pb.MovementRequest_WEEK:
		start = strconv.Itoa(end - secondsInWeek)
		denomiator = 2
	case pb.MovementRequest_MONTH:
		start = strconv.Itoa(end - secondsInMonth)
		denomiator = 7
	case pb.MovementRequest_SIXMONTH:
		start = strconv.Itoa(end - secondsIn6Months)
		denomiator = 30
	case pb.MovementRequest_ANNUAL:
		start = strconv.Itoa(end - secondsInYear)
		denomiator = 60
	}
	sql := fmt.Sprintf(`SELECT TIME, V4COUNT, V6COUNT FROM INFO WHERE TIME >=
						'%s' AND TIME <= '%d'`, start, end)

	var tv []*pb.V4V6Time
	rows, err := db.Query(sql)
	if err != nil {
		return &pb.MovementTotalsResponse{}, err
	}
	defer rows.Close()

	i := 0
	for rows.Next() {
		// We don't need all values. Only each 1/denomiator value
		i++
		if i%denomiator != 0 {
			continue
		}

		var v pb.V4V6Time
		err := rows.Scan(&v.Time, &v.V4Values, &v.V6Values)
		if err != nil {
			return &pb.MovementTotalsResponse{}, err
		}
		tv = append(tv, &v)
	}

	return &pb.MovementTotalsResponse{
		Values: tv,
	}, nil

}

func getRPKIHelper() (*pb.Roas, error) {
	var r pb.Roas
	sql := `select ROAVALIDV4,ROAINVALIDV4,ROAUNKNOWNV4,ROAVALIDV6,ROAINVALIDV6,ROAUNKNOWNV6
	from INFO ORDER by TIME DESC LIMIT 1`
	err := db.QueryRow(sql).Scan(
		&r.V4Valid,
		&r.V4Invalid,
		&r.V4Unknown,
		&r.V6Valid,
		&r.V6Invalid,
		&r.V6Unknown,
	)
	if err != nil {
		return nil, err
	}
	return &r, nil
}

func getAsnameHelper(a *pb.GetAsnameRequest) (*pb.GetAsnameResponse, error) {
	var n pb.GetAsnameResponse
	sql := fmt.Sprintf(`select ASNAME, LOCALE from ASNUMNAME WHERE ASNUMBER = '%d'`,
		a.GetAsNumber())
	err := db.QueryRow(sql).Scan(
		&n.AsName,
		&n.AsLocale,
	)

	if err != nil {
		return nil, err
	}
	return &n, nil
}

func updateASNHelper(asn *pb.AsnamesRequest) (*pb.Result, error) {

	// Create a new temp table to hold new values.
	_, err := db.Exec(`CREATE TABLE ASNUMNAME_NEW LIKE ASNUMNAME`)
	if err != nil {
		return &pb.Result{}, err
	}

	// Dump the new values into the new temp table.
	tx, err := db.Begin()
	if err != nil {
		log.Printf("Error on db.Begin: %v\n", err)
		return &pb.Result{}, err
	}
	stmt, err := tx.Prepare(`INSERT INTO ASNUMNAME_NEW SET ASNUMBER=?, ASNAME=?, LOCALE=?`)
	for _, as := range asn.GetAsnNames() {
		_, err := stmt.Exec(as.GetAsNumber(), as.GetAsName(), as.GetAsLocale())
		if err != nil {
			log.Printf("Error on statement: %v\n", err)
			return &pb.Result{}, err
		}
	}
	if err := tx.Commit(); err != nil {
		return &pb.Result{}, err
	}

	// Now rename and shift in order to only have one table.
	tx, err = db.Begin()
	if err != nil {
		log.Printf("Error on db.Begin: %v\n", err)
		return &pb.Result{}, err
	}
	tx.Exec(`RENAME TABLE ASNUMNAME TO ASNUMNAME_OLD`)
	tx.Exec(`RENAME TABLE ASNUMNAME_NEW TO ASNUMNAME`)
	tx.Exec(`DROP TABLES ASNUMNAME_OLD`)
	if err := tx.Commit(); err != nil {
		return &pb.Result{}, err
	}

	return &pb.Result{
		Success: true,
	}, nil

}

func updateTweetBitHelper(t uint64) (*pb.Result, error) {
	_, err := db.Exec(fmt.Sprintf(`UPDATE INFO SET TWEET = 1 WHERE TIME = %d`, t))
	if err != nil {
		return &pb.Result{}, err
	}
	return &pb.Result{
		Success: true,
	}, nil

}
