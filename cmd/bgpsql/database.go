package main

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	com "github.com/mellowdrifter/bgp_infrastructure/pkg/common"
	pb "github.com/mellowdrifter/bgp_infrastructure/proto/bgpsql"
	_ "modernc.org/sqlite"
)

//go:embed migrations/003_sqlite_schema.sql
var sqliteSchemaDDL string

var infoMetricColumns = []string{
	"V4COUNT", "V6COUNT", "V4TOTAL", "V6TOTAL", "PEERS_CONFIGURED", "PEERS_UP",
	"PEERS6_CONFIGURED", "PEERS6_UP", "V4_24", "V4_23", "V4_22",
	"V4_21", "V4_20", "V4_19", "V4_18", "V4_17", "V4_16", "V4_15", "V4_14", "V4_13", "V4_12",
	"V4_11", "V4_10", "V4_09", "V4_08", "V6_48", "V6_47", "V6_46",
	"V6_45", "V6_44", "V6_43", "V6_42", "V6_41", "V6_40", "V6_39",
	"V6_38", "V6_37", "V6_36", "V6_35", "V6_34", "V6_33", "V6_32",
	"V6_31", "V6_30", "V6_29", "V6_28", "V6_27", "V6_26", "V6_25",
	"V6_24", "V6_23", "V6_22", "V6_21", "V6_20", "V6_19", "V6_18",
	"V6_17", "V6_16", "V6_15", "V6_14", "V6_13", "V6_12", "V6_11",
	"V6_10", "V6_09", "V6_08", "AS4_LEN", "AS6_LEN", "AS10_LEN",
	"AS4_ONLY", "AS6_ONLY", "AS_BOTH", "LARGEC4", "LARGEC6",
	"ROAVALIDV4", "ROAINVALIDV4", "ROAUNKNOWNV4",
	"ROAVALIDV6", "ROAINVALIDV6", "ROAUNKNOWNV6",
}

var (
	mysqlAddLatestQuery  string
	sqliteAddLatestQuery string
)

func init() {
	cols := append([]string{"source", "TIME", "quality", "quality_note"}, infoMetricColumns...)
	colList := strings.Join(cols, ", ")

	placeholders := make([]string, len(cols))
	for i := range placeholders {
		placeholders[i] = "?"
	}
	valList := strings.Join(placeholders, ", ")

	// MySQL / MariaDB: ON DUPLICATE KEY UPDATE col=VALUES(col)..., ingested_at=CURRENT_TIMESTAMP
	mysqlUpdates := make([]string, 0, len(infoMetricColumns)+3)
	mysqlUpdates = append(mysqlUpdates, "quality = VALUES(quality)", "quality_note = VALUES(quality_note)")
	for _, col := range infoMetricColumns {
		mysqlUpdates = append(mysqlUpdates, fmt.Sprintf("%s = VALUES(%s)", col, col))
	}
	mysqlUpdates = append(mysqlUpdates, "ingested_at = CURRENT_TIMESTAMP")
	mysqlAddLatestQuery = fmt.Sprintf("INSERT INTO INFO (%s) VALUES (%s) ON DUPLICATE KEY UPDATE %s",
		colList, valList, strings.Join(mysqlUpdates, ", "))

	// SQLite: ON CONFLICT(source, TIME) DO UPDATE SET col=excluded.col..., ingested_at=unixepoch()
	sqliteUpdates := make([]string, 0, len(infoMetricColumns)+3)
	sqliteUpdates = append(sqliteUpdates, "quality = excluded.quality", "quality_note = excluded.quality_note")
	for _, col := range infoMetricColumns {
		sqliteUpdates = append(sqliteUpdates, fmt.Sprintf("%s = excluded.%s", col, col))
	}
	sqliteUpdates = append(sqliteUpdates, "ingested_at = unixepoch()")
	sqliteAddLatestQuery = fmt.Sprintf("INSERT INTO INFO (%s) VALUES (%s) ON CONFLICT(source, TIME) DO UPDATE SET %s",
		colList, valList, strings.Join(sqliteUpdates, ", "))
}

func queryContextWithTimeout(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if ctx == nil {
		ctx = context.Background()
	}
	if _, ok := ctx.Deadline(); ok {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, timeout)
}

func getUpsertQuery(driver string) string {
	if strings.EqualFold(driver, "sqlite") {
		return sqliteAddLatestQuery
	}
	return mysqlAddLatestQuery
}

func isDBDriverSQLite(db *sql.DB) bool {
	if db != nil && db.Driver() != nil {
		drvName := fmt.Sprintf("%T", db.Driver())
		return strings.Contains(strings.ToLower(drvName), "sqlite")
	}
	return false
}

func openSQLitePools(dbPath string) (*sql.DB, *sql.DB, error) {
	dsn := fmt.Sprintf("file:%s?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)&_pragma=foreign_keys(ON)&_pragma=cache_size(-16000)", dbPath)

	writeDB, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open write db: %w", err)
	}
	writeDB.SetMaxOpenConns(1)

	// Ensure WAL mode is active
	if _, err := writeDB.Exec("PRAGMA journal_mode=WAL;"); err != nil {
		writeDB.Close()
		return nil, nil, fmt.Errorf("failed to set WAL mode: %w", err)
	}

	readDB, err := sql.Open("sqlite", dsn)
	if err != nil {
		writeDB.Close()
		return nil, nil, fmt.Errorf("failed to open read db: %w", err)
	}
	readDB.SetMaxOpenConns(4)
	readDB.SetMaxIdleConns(4)

	return writeDB, readDB, nil
}

func initSQLiteSchema(db *sql.DB) error {
	var userVersion int
	err := db.QueryRow("PRAGMA user_version;").Scan(&userVersion)
	if err != nil {
		return fmt.Errorf("failed to check user_version: %w", err)
	}
	if userVersion < 3 {
		if _, err := db.Exec(sqliteSchemaDDL); err != nil {
			return fmt.Errorf("failed to execute sqlite schema migration: %w", err)
		}
	}
	return nil
}

func execAddSample(ctx context.Context, stmt *sql.Stmt, b *com.BgpUpdate) error {
	source := b.Source
	if source == "" {
		source = "bgp1"
	}

	sampleTime := b.SampleTime
	if sampleTime == 0 {
		sampleTime = b.Time
	}
	sampleTime = sampleTime - (sampleTime % 300)

	quality := b.Quality
	if quality == "" {
		quality = "ok"
	}
	var qualityNote sql.NullString
	if b.QualityNote != "" {
		qualityNote = sql.NullString{String: b.QualityNote, Valid: true}
	}

	_, err := stmt.ExecContext(ctx,
		source, sampleTime, quality, qualityNote,
		b.V4Count, b.V6Count, b.V4Total, b.V6Total, b.PeersConfigured,
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
	return err
}

// add latest BGP update information to database
func addLatestHelper(ctx context.Context, b *com.BgpUpdate, db *sql.DB, driver ...string) error {
	if db == nil {
		return fmt.Errorf("db object is nil")
	}

	ctx, cancel := queryContextWithTimeout(ctx, 10*time.Second)
	defer cancel()

	drv := "mysql"
	if len(driver) > 0 && driver[0] != "" {
		drv = driver[0]
	} else if isDBDriverSQLite(db) {
		drv = "sqlite"
	}
	query := getUpsertQuery(drv)

	stmt, err := db.PrepareContext(ctx, query)
	if err != nil {
		return fmt.Errorf("unable to prepare statement: %w", err)
	}
	defer stmt.Close()

	return execAddSample(ctx, stmt, b)
}

func getPrefixCountHelper(ctx context.Context, db *sql.DB) (*pb.PrefixCountResponse, error) {
	if db == nil {
		return nil, fmt.Errorf("db object is nil")
	}
	ctx, cancel := queryContextWithTimeout(ctx, 10*time.Second)
	defer cancel()

	var data pb.PrefixCountResponse

	// Latest data
	sq1 := `SELECT TIME, V4COUNT, V6COUNT FROM curated_samples ORDER BY TIME DESC LIMIT 1`
	err := db.QueryRowContext(ctx, sq1).Scan(
		&data.Time,
		&data.Active_4,
		&data.Active_6,
	)
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve data: %w", err)
	}

	// Six hours ago (last tweeted data)
	sq2 := `SELECT V4COUNT, V6COUNT FROM curated_samples WHERE TWEET IS NOT NULL
			ORDER BY TIME DESC LIMIT 1`
	err = db.QueryRowContext(ctx, sq2).Scan(
		&data.Sixhoursv4,
		&data.Sixhoursv6,
	)
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve data: %w", err)
	}

	// Last weeks numbers
	lastWeek := int32(time.Now().Unix()) - 604800
	sq3 := `SELECT V4COUNT, V6COUNT FROM curated_samples WHERE TWEET IS NOT NULL
			AND TIME < ? ORDER BY TIME DESC LIMIT 1`
	err = db.QueryRowContext(ctx, sq3, lastWeek).Scan(
		&data.Weekagov4,
		&data.Weekagov6,
	)
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve data: %w", err)
	}

	// /24 and /48 counts
	sq4 := `SELECT V4_24, V6_48 FROM curated_samples ORDER BY TIME DESC LIMIT 1`
	err = db.QueryRowContext(ctx, sq4).Scan(
		&data.Slash24,
		&data.Slash48,
	)
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve data: %w", err)
	}

	return &data, nil
}

func getPieSubnetsHelper(ctx context.Context, db *sql.DB) (*pb.PieSubnetsResponse, error) {
	if db == nil {
		return nil, fmt.Errorf("db object is nil")
	}
	ctx, cancel := queryContextWithTimeout(ctx, 10*time.Second)
	defer cancel()

	var masks pb.Masks
	var pie pb.PieSubnetsResponse

	err := db.QueryRowContext(ctx, `SELECT V4_08,V4_09,V4_10,V4_11,V4_12,V4_13,V4_14,
        V4_15,V4_16,V4_17,V4_18,V4_19,V4_20,V4_21,V4_22,
        V4_23,V4_24,V4COUNT,V6_48,V6_47,V6_46,V6_45,V6_44,
        V6_43,V6_42,V6_41,V6_40,V6_39,V6_38,V6_37,V6_36,
        V6_35,V6_34,V6_33,V6_32,V6_31,V6_30,V6_29,V6_28,
        V6_27,V6_26,V6_25,V6_24,V6_23,V6_22,V6_21,V6_20,
        V6_19,V6_18,V6_17,V6_16,V6_15,V6_14,V6_13,V6_12,
        V6_11,V6_10,V6_09,V6_08,V6COUNT,
        TIME FROM curated_samples ORDER BY TIME DESC LIMIT 1`).Scan(
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

func getMovementTotalsHelper(ctx context.Context, m *pb.MovementRequest, db *sql.DB) (*pb.MovementTotalsResponse, error) {
	if db == nil {
		return &pb.MovementTotalsResponse{}, fmt.Errorf("db object is nil")
	}
	ctx, cancel := queryContextWithTimeout(ctx, 10*time.Second)
	defer cancel()

	// time helpers
	secondsInWeek := 604800
	secondsInMonth := 2628000
	secondsIn6Months := secondsInMonth * 6
	secondsInYear := secondsIn6Months * 2
	end := int(time.Now().Unix() - 66600)

	var start int
	var denominator int
	switch m.GetPeriod() {
	case pb.MovementRequest_WEEK:
		start = end - secondsInWeek
		denominator = 2
	case pb.MovementRequest_MONTH:
		start = end - secondsInMonth
		denominator = 7
	case pb.MovementRequest_SIXMONTH:
		start = end - secondsIn6Months
		denominator = 30
	case pb.MovementRequest_ANNUAL:
		start = end - secondsInYear
		denominator = 60
	default:
		start = end - secondsInWeek
		denominator = 2
	}

	query := `SELECT TIME, V4COUNT, V6COUNT FROM curated_samples WHERE TIME >= ? AND TIME <= ? ORDER BY TIME ASC`

	var tv []*pb.V4V6Time
	rows, err := db.QueryContext(ctx, query, start, end)
	if err != nil {
		return &pb.MovementTotalsResponse{}, err
	}
	defer rows.Close()

	i := 0
	for rows.Next() {
		// We don't need all values. Only each 1/denominator value
		i++
		if i%denominator != 0 {
			continue
		}

		var v pb.V4V6Time
		err := rows.Scan(&v.Time, &v.V4Values, &v.V6Values)
		if err != nil {
			return &pb.MovementTotalsResponse{}, err
		}
		tv = append(tv, &v)
	}
	if err := rows.Err(); err != nil {
		return &pb.MovementTotalsResponse{}, err
	}

	return &pb.MovementTotalsResponse{
		Values: tv,
	}, nil
}

func getRPKIHelper(ctx context.Context, db *sql.DB) (*pb.Roas, error) {
	if db == nil {
		return nil, fmt.Errorf("db object is nil")
	}
	ctx, cancel := queryContextWithTimeout(ctx, 10*time.Second)
	defer cancel()

	var r pb.Roas
	query := `SELECT ROAVALIDV4, ROAINVALIDV4, ROAUNKNOWNV4, ROAVALIDV6, ROAINVALIDV6, ROAUNKNOWNV6
		FROM curated_samples ORDER BY TIME DESC LIMIT 1`
	err := db.QueryRowContext(ctx, query).Scan(
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

func getAsnameHelper(ctx context.Context, a *pb.GetAsnameRequest, db *sql.DB) (*pb.GetAsnameResponse, error) {
	if db == nil {
		return nil, fmt.Errorf("db object is nil")
	}
	ctx, cancel := queryContextWithTimeout(ctx, 10*time.Second)
	defer cancel()

	var n pb.GetAsnameResponse
	query := `SELECT ASNAME, LOCALE FROM ASNUMNAME WHERE ASNUMBER = ?`
	err := db.QueryRowContext(ctx, query, a.GetAsNumber()).Scan(
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

func getAsnamesHelper(ctx context.Context, db *sql.DB) (*pb.GetAsnamesResponse, error) {
	if db == nil {
		return nil, fmt.Errorf("db object is nil")
	}
	ctx, cancel := queryContextWithTimeout(ctx, 10*time.Second)
	defer cancel()

	var n pb.GetAsnamesResponse
	query := `SELECT ASNUMBER, ASNAME, LOCALE FROM ASNUMNAME`
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return &n, err
	}
	defer rows.Close()

	for rows.Next() {
		var a pb.AsnumberAsnames
		err = rows.Scan(&a.AsNumber, &a.AsName, &a.AsLocale)
		if err != nil {
			return nil, err
		}
		n.Asnumnames = append(n.Asnumnames, &a)
	}
	err = rows.Err()
	if err != nil {
		return nil, err
	}

	return &n, nil
}

// updateASNHelper atomically replaces the ASNUMNAME table in a single transaction.
func updateASNHelper(ctx context.Context, asn *pb.AsnamesRequest, db *sql.DB) (*pb.Result, error) {
	if db == nil {
		return &pb.Result{Success: false}, fmt.Errorf("db object is nil")
	}
	ctx, cancel := queryContextWithTimeout(ctx, 30*time.Second)
	defer cancel()

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return &pb.Result{Success: false}, fmt.Errorf("unable to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Clear table atomically within the transaction
	if _, err := tx.ExecContext(ctx, `DELETE FROM ASNUMNAME`); err != nil {
		return &pb.Result{Success: false}, fmt.Errorf("unable to clear ASNUMNAME: %w", err)
	}

	stmt, err := tx.PrepareContext(ctx, `INSERT INTO ASNUMNAME (ASNUMBER, ASNAME, LOCALE) VALUES (?, ?, ?)`)
	if err != nil {
		return &pb.Result{Success: false}, fmt.Errorf("unable to prepare insert statement: %w", err)
	}
	defer stmt.Close()

	for _, as := range asn.GetAsnNames() {
		var locale interface{}
		if as.GetAsLocale() != "" {
			locale = as.GetAsLocale()
		}
		if _, err := stmt.ExecContext(ctx, as.GetAsNumber(), as.GetAsName(), locale); err != nil {
			return &pb.Result{Success: false}, fmt.Errorf("error inserting AS%d: %w", as.GetAsNumber(), err)
		}
	}

	if err := tx.Commit(); err != nil {
		return &pb.Result{Success: false}, fmt.Errorf("unable to commit ASNUMNAME transaction: %w", err)
	}

	return &pb.Result{
		Success: true,
	}, nil
}

func updateTweetBitHelper(ctx context.Context, t uint64, db *sql.DB) (*pb.Result, error) {
	if db == nil {
		return nil, fmt.Errorf("db object is nil")
	}
	ctx, cancel := queryContextWithTimeout(ctx, 10*time.Second)
	defer cancel()

	_, err := db.ExecContext(ctx, `UPDATE INFO SET TWEET = 1 WHERE TIME = ?`, t)
	if err != nil {
		return &pb.Result{
			Success: false,
		}, err
	}
	return &pb.Result{
		Success: true,
	}, nil
}

func getSampleIndexHelper(ctx context.Context, since, until uint64, db *sql.DB) (*pb.SampleIndexResponse, error) {
	if db == nil {
		return nil, fmt.Errorf("db object is nil")
	}
	ctx, cancel := queryContextWithTimeout(ctx, 30*time.Second)
	defer cancel()

	query := `SELECT source, TIME, quality FROM INFO WHERE TIME >= ?`
	args := []interface{}{since}
	if until > 0 {
		query += ` AND TIME <= ?`
		args = append(args, until)
	}
	query += ` ORDER BY TIME ASC`

	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query sample index: %w", err)
	}
	defer rows.Close()

	var keys []*pb.SampleKey
	for rows.Next() {
		var k pb.SampleKey
		if err := rows.Scan(&k.Source, &k.Time, &k.Quality); err != nil {
			return nil, fmt.Errorf("failed to scan sample key: %w", err)
		}
		keys = append(keys, &k)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return &pb.SampleIndexResponse{Keys: keys}, nil
}

const sampleSelectCols = `source, TIME, quality, quality_note, V4COUNT, V6COUNT, V4TOTAL, V6TOTAL,
	PEERS_CONFIGURED, PEERS_UP, PEERS6_CONFIGURED, PEERS6_UP, V4_24, V4_23, V4_22, V4_21,
	V4_20, V4_19, V4_18, V4_17, V4_16, V4_15, V4_14, V4_13, V4_12, V4_11, V4_10, V4_09,
	V4_08, V6_48, V6_47, V6_46, V6_45, V6_44, V6_43, V6_42, V6_41, V6_40, V6_39, V6_38,
	V6_37, V6_36, V6_35, V6_34, V6_33, V6_32, V6_31, V6_30, V6_29, V6_28, V6_27, V6_26,
	V6_25, V6_24, V6_23, V6_22, V6_21, V6_20, V6_19, V6_18, V6_17, V6_16, V6_15, V6_14,
	V6_13, V6_12, V6_11, V6_10, V6_09, V6_08, AS4_LEN, AS6_LEN, AS10_LEN, AS4_ONLY,
	AS6_ONLY, AS_BOTH, LARGEC4, LARGEC6, ROAVALIDV4, ROAINVALIDV4, ROAUNKNOWNV4,
	ROAVALIDV6, ROAINVALIDV6, ROAUNKNOWNV6`

type nullUint32 uint32

func (n *nullUint32) Scan(src interface{}) error {
	if src == nil {
		*n = 0
		return nil
	}
	var ni sql.NullInt64
	if err := ni.Scan(src); err != nil {
		return err
	}
	if ni.Valid {
		*n = nullUint32(ni.Int64)
	} else {
		*n = 0
	}
	return nil
}

func scanFullBgpUpdate(scanner interface{ Scan(...interface{}) error }) (*com.BgpUpdate, error) {
	var b com.BgpUpdate
	var qualityNote sql.NullString
	err := scanner.Scan(
		&b.Source, &b.Time, &b.Quality, &qualityNote,
		&b.V4Count, &b.V6Count, (*nullUint32)(&b.V4Total), (*nullUint32)(&b.V6Total),
		(*nullUint32)(&b.PeersConfigured), (*nullUint32)(&b.PeersUp), (*nullUint32)(&b.Peers6Configured), (*nullUint32)(&b.Peers6Up),
		(*nullUint32)(&b.V4_24), (*nullUint32)(&b.V4_23), (*nullUint32)(&b.V4_22), (*nullUint32)(&b.V4_21), (*nullUint32)(&b.V4_20), (*nullUint32)(&b.V4_19),
		(*nullUint32)(&b.V4_18), (*nullUint32)(&b.V4_17), (*nullUint32)(&b.V4_16), (*nullUint32)(&b.V4_15), (*nullUint32)(&b.V4_14), (*nullUint32)(&b.V4_13),
		(*nullUint32)(&b.V4_12), (*nullUint32)(&b.V4_11), (*nullUint32)(&b.V4_10), (*nullUint32)(&b.V4_09), (*nullUint32)(&b.V4_08),
		(*nullUint32)(&b.V6_48), (*nullUint32)(&b.V6_47), (*nullUint32)(&b.V6_46), (*nullUint32)(&b.V6_45), (*nullUint32)(&b.V6_44), (*nullUint32)(&b.V6_43),
		(*nullUint32)(&b.V6_42), (*nullUint32)(&b.V6_41), (*nullUint32)(&b.V6_40), (*nullUint32)(&b.V6_39), (*nullUint32)(&b.V6_38), (*nullUint32)(&b.V6_37),
		(*nullUint32)(&b.V6_36), (*nullUint32)(&b.V6_35), (*nullUint32)(&b.V6_34), (*nullUint32)(&b.V6_33), (*nullUint32)(&b.V6_32), (*nullUint32)(&b.V6_31),
		(*nullUint32)(&b.V6_30), (*nullUint32)(&b.V6_29), (*nullUint32)(&b.V6_28), (*nullUint32)(&b.V6_27), (*nullUint32)(&b.V6_26), (*nullUint32)(&b.V6_25),
		(*nullUint32)(&b.V6_24), (*nullUint32)(&b.V6_23), (*nullUint32)(&b.V6_22), (*nullUint32)(&b.V6_21), (*nullUint32)(&b.V6_20), (*nullUint32)(&b.V6_19),
		(*nullUint32)(&b.V6_18), (*nullUint32)(&b.V6_17), (*nullUint32)(&b.V6_16), (*nullUint32)(&b.V6_15), (*nullUint32)(&b.V6_14), (*nullUint32)(&b.V6_13),
		(*nullUint32)(&b.V6_12), (*nullUint32)(&b.V6_11), (*nullUint32)(&b.V6_10), (*nullUint32)(&b.V6_09), (*nullUint32)(&b.V6_08),
		(*nullUint32)(&b.As4), (*nullUint32)(&b.As6), (*nullUint32)(&b.As10), (*nullUint32)(&b.As4Only), (*nullUint32)(&b.As6Only), (*nullUint32)(&b.AsBoth),
		(*nullUint32)(&b.LargeC4), (*nullUint32)(&b.LargeC6),
		(*nullUint32)(&b.Roavalid4), (*nullUint32)(&b.Roainvalid4), (*nullUint32)(&b.Roaunknown4),
		(*nullUint32)(&b.Roavalid6), (*nullUint32)(&b.Roainvalid6), (*nullUint32)(&b.Roaunknown6),
	)
	if err != nil {
		return nil, err
	}
	if qualityNote.Valid {
		b.QualityNote = qualityNote.String
	}
	b.SampleTime = b.Time
	return &b, nil
}

func getSampleBatchHelper(ctx context.Context, keys []*pb.SampleKey, db *sql.DB) (*pb.SampleBatchResponse, error) {
	if db == nil {
		return nil, fmt.Errorf("db object is nil")
	}
	ctx, cancel := queryContextWithTimeout(ctx, 30*time.Second)
	defer cancel()

	stmt, err := db.PrepareContext(ctx, fmt.Sprintf("SELECT %s FROM INFO WHERE source = ? AND TIME = ?", sampleSelectCols))
	if err != nil {
		return nil, fmt.Errorf("failed to prepare select statement: %w", err)
	}
	defer stmt.Close()

	var samples []*pb.Values
	for _, k := range keys {
		row := stmt.QueryRowContext(ctx, k.GetSource(), k.GetTime())
		update, err := scanFullBgpUpdate(row)
		if err == sql.ErrNoRows {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("failed to scan row for %s@%d: %w", k.GetSource(), k.GetTime(), err)
		}
		samples = append(samples, com.StructToProto(update))
	}

	return &pb.SampleBatchResponse{Samples: samples}, nil
}

func addSampleBatchHelper(ctx context.Context, samples []*pb.Values, db *sql.DB, driver ...string) (*pb.Result, error) {
	if db == nil {
		return nil, fmt.Errorf("db object is nil")
	}

	if len(samples) == 0 {
		return &pb.Result{Success: true}, nil
	}

	ctx, cancel := queryContextWithTimeout(ctx, 30*time.Second)
	defer cancel()

	drv := "mysql"
	if len(driver) > 0 && driver[0] != "" {
		drv = driver[0]
	} else if isDBDriverSQLite(db) {
		drv = "sqlite"
	}
	query := getUpsertQuery(drv)

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("unable to prepare statement: %w", err)
	}
	defer stmt.Close()

	for _, s := range samples {
		update := com.ProtoToStruct(s)
		if err := execAddSample(ctx, stmt, update); err != nil {
			return &pb.Result{Success: false}, fmt.Errorf("failed to insert reconciled sample %s@%d: %w", update.Source, update.Time, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit batch transaction: %w", err)
	}

	return &pb.Result{Success: true}, nil
}
