package main

import (
	"database/sql"
	"fmt"
	"log"
	"strconv"
	"time"

	_ "github.com/go-sql-driver/mysql"
	com "github.com/mellowdrifter/bgp_infrastructure/common"
	pb "github.com/mellowdrifter/bgp_infrastructure/proto/bgpsql"
)

// add latest BGP update information to database
func addLatestHelper(b *com.BgpUpdate, db *sql.DB) error {
	if db == nil {
		log.Fatalf("db object is nil")
	}
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
	res, err := stmt.Exec(b.Time, b.V4Count, b.V6Count, b.V4Total, b.V6Total, b.PeersConfigured,
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
		return fmt.Errorf("Unable to update database: %w", err)
	}
	log.Printf("updated database: %v", res)
	return nil

}

func getPrefixCountHelper(db *sql.DB) (*pb.PrefixCountResponse, error) {
	if db == nil {
		log.Fatalf("db object is nil")
	}
	var data pb.PrefixCountResponse

	// Latest data
	sq1 := `SELECT TIME, V4COUNT, V6COUNT FROM INFO ORDER BY TIME DESC LIMIT 1`
	err := db.QueryRow(sq1).Scan(
		&data.Time,
		&data.Active_4,
		&data.Active_6,
	)
	if err != nil {
		return nil, fmt.Errorf("Unable to retrieve data: %w", err)
	}

	// Six hours ago (last tweeted data)
	sq2 := `SELECT V4COUNT, V6COUNT FROM INFO WHERE TWEET IS NOT NULL
			ORDER BY TIME DESC LIMIT 1`
	err = db.QueryRow(sq2).Scan(
		&data.Sixhoursv4,
		&data.Sixhoursv6,
	)
	if err != nil {
		return nil, fmt.Errorf("Unable to retrieve data: %w", err)
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
		return nil, fmt.Errorf("Unable to retrieve data: %w", err)
	}

	// /24 and /48 counts
	sq4 := `SELECT V4_24, V6_48 FROM INFO ORDER BY TIME DESC LIMIT 1`
	err = db.QueryRow(sq4).Scan(
		&data.Slash24,
		&data.Slash48,
	)
	if err != nil {
		return nil, fmt.Errorf("Unable to retrieve data: %w", err)
	}

	return &data, nil
}

func getPieSubnetsHelper(db *sql.DB) (*pb.PieSubnetsResponse, error) {

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

func getMovementTotalsHelper(m *pb.MovementRequest, db *sql.DB) (*pb.MovementTotalsResponse, error) {
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
	query := fmt.Sprintf(`SELECT TIME, V4COUNT, V6COUNT FROM INFO WHERE TIME >=
						'%s' AND TIME <= '%d'`, start, end)

	var tv []*pb.V4V6Time
	rows, err := db.Query(query)
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

func getRPKIHelper(db *sql.DB) (*pb.Roas, error) {
	var r pb.Roas
	query := `select ROAVALIDV4,ROAINVALIDV4,ROAUNKNOWNV4,ROAVALIDV6,ROAINVALIDV6,ROAUNKNOWNV6
	from INFO ORDER by TIME DESC LIMIT 1`
	err := db.QueryRow(query).Scan(
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

func getAsnameHelper(a *pb.GetAsnameRequest, db *sql.DB) (*pb.GetAsnameResponse, error) {
	var n pb.GetAsnameResponse
	query := fmt.Sprintf(`select ASNAME, LOCALE from ASNUMNAME WHERE ASNUMBER = '%d'`,
		a.GetAsNumber())
	err := db.QueryRow(query).Scan(
		&n.AsName,
		&n.AsLocale,
	)

	switch {
	// No result returned, so does not exist.
	case err == sql.ErrNoRows:
		n.Exists = false
		return &n, nil
	case err != nil:
		return nil, err
	default:
		// Else it exists and we can return
		n.Exists = true
		return &n, nil
	}

}

func updateASNHelper(asn *pb.AsnamesRequest, db *sql.DB) (*pb.Result, error) {
	// Temp table may be sitting around from a failed attempt.
	stmt, _ := db.Prepare(`DROP TABLE IF EXISTS ASNUMNAME_NEW`)
	stmt.Exec()

	// Create temporary holding table.
	stmt, _ = db.Prepare(`CREATE TABLE ASNUMNAME_NEW (
  				ASNUMBER INTEGER NOT NULL,
  				ASNAME TEXT NOT NULL,
  				LOCALE TEXT DEFAULT NULL)`)

	_, err := stmt.Exec()
	if err != nil {
		return &pb.Result{
			Success: false,
		}, fmt.Errorf("unable to create temp database: %w", err)
	}

	// Dump the new values into the new temp table.
	tx, _ := db.Begin()
	stmt, _ = tx.Prepare(`INSERT INTO ASNUMNAME_NEW (
		ASNUMBER, ASNAME, LOCALE) VALUES (?, ?, ?)`)
	for _, as := range asn.GetAsnNames() {
		_, err := stmt.Exec(as.GetAsNumber(), as.GetAsName(), as.GetAsLocale())
		if err != nil {
			return &pb.Result{
				Success: false,
			}, fmt.Errorf("error on statement execute: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return &pb.Result{
			Success: false,
		}, fmt.Errorf("unable to complete transaction: %w", err)
	}

	// Now rename and shift in order to only have one table.
	tx, _ = db.Begin()
	tx.Exec(`DROP TABLE IF EXISTS ASNUMNAME`)
	tx.Exec(`ALTER TABLE ASNUMNAME_NEW RENAME TO ASNUMNAME`)
	if err := tx.Commit(); err != nil {
		return &pb.Result{
			Success: false,
		}, fmt.Errorf("unable to complete transaction: %w", err)
	}

	return &pb.Result{
		Success: true,
	}, nil

}

func updateTweetBitHelper(t uint64, db *sql.DB) (*pb.Result, error) {
	if db == nil {
		log.Fatalf("db object is nil")
	}
	_, err := db.Exec(fmt.Sprintf(`UPDATE INFO SET TWEET = 1 WHERE TIME = %d`, t))
	if err != nil {
		return &pb.Result{
			Success: false,
		}, err
	}
	return &pb.Result{
		Success: true,
	}, nil

}
