package main

import (
	"crypto/sha256"
	"database/sql"
	_ "embed"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	ini "gopkg.in/ini.v1"
	_ "modernc.org/sqlite"
)

//go:embed 003_sqlite_schema.sql
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

func migrateInfo(srcDB *sql.DB, dstDB *sql.DB, isSrcMySQL bool, delta bool, batchSize int) (int, error) {
	var deltaThreshold int64 = 0
	if delta {
		var maxIngested sql.NullInt64
		err := dstDB.QueryRow("SELECT MAX(ingested_at) FROM INFO;").Scan(&maxIngested)
		if err != nil {
			return 0, fmt.Errorf("failed to query max ingested_at from SQLite: %w", err)
		}
		if maxIngested.Valid {
			deltaThreshold = maxIngested.Int64
		}
		log.Printf("Delta mode active: querying rows newer than %d", deltaThreshold)
	}

	ingestedCol := "ingested_at"
	if isSrcMySQL {
		ingestedCol = "COALESCE(UNIX_TIMESTAMP(ingested_at), 0)"
	}

	allSelectCols := append([]string{"source", "TIME", "quality", "quality_note"}, infoMetricColumns...)
	allSelectCols = append(allSelectCols, "COALESCE(TWEET, 0)", ingestedCol)

	selectQuery := fmt.Sprintf("SELECT %s FROM INFO", strings.Join(allSelectCols, ", "))
	if delta && deltaThreshold > 0 {
		if isSrcMySQL {
			selectQuery += fmt.Sprintf(" WHERE UNIX_TIMESTAMP(ingested_at) > %d", deltaThreshold)
		} else {
			selectQuery += fmt.Sprintf(" WHERE ingested_at > %d", deltaThreshold)
		}
	}
	selectQuery += " ORDER BY TIME ASC"

	rows, err := srcDB.Query(selectQuery)
	if err != nil {
		return 0, fmt.Errorf("failed to query source INFO: %w", err)
	}
	defer rows.Close()

	insertCols := append([]string{"source", "TIME", "quality", "quality_note"}, infoMetricColumns...)
	insertCols = append(insertCols, "TWEET", "ingested_at")

	placeholders := make([]string, len(insertCols))
	for i := range placeholders {
		placeholders[i] = "?"
	}

	updates := make([]string, 0, len(infoMetricColumns)+4)
	updates = append(updates, "quality = excluded.quality", "quality_note = excluded.quality_note")
	for _, col := range infoMetricColumns {
		updates = append(updates, fmt.Sprintf("%s = excluded.%s", col, col))
	}
	updates = append(updates, "TWEET = excluded.TWEET", "ingested_at = excluded.ingested_at")

	sqliteUpsertQuery := fmt.Sprintf("INSERT INTO INFO (%s) VALUES (%s) ON CONFLICT(source, TIME) DO UPDATE SET %s",
		strings.Join(insertCols, ", "),
		strings.Join(placeholders, ", "),
		strings.Join(updates, ", "))

	totalRows := 0
	batchCount := 0

	tx, err := dstDB.Begin()
	if err != nil {
		return 0, fmt.Errorf("failed to begin transaction: %w", err)
	}
	stmt, err := tx.Prepare(sqliteUpsertQuery)
	if err != nil {
		tx.Rollback()
		return 0, fmt.Errorf("failed to prepare SQLite upsert statement: %w", err)
	}

	destVals := make([]interface{}, len(insertCols))
	destPtrs := make([]interface{}, len(insertCols))
	for i := range destVals {
		destPtrs[i] = &destVals[i]
	}

	for rows.Next() {
		if err := rows.Scan(destPtrs...); err != nil {
			stmt.Close()
			tx.Rollback()
			return 0, fmt.Errorf("failed to scan source row: %w", err)
		}

		for i, v := range destVals {
			if b, ok := v.([]byte); ok {
				destVals[i] = string(b)
			}
		}
		if q, ok := destVals[2].(string); !ok || (q != "ok" && q != "suspect" && q != "bad") {
			destVals[2] = "ok"
		}

		if _, err := stmt.Exec(destVals...); err != nil {
			stmt.Close()
			tx.Rollback()
			return 0, fmt.Errorf("failed to execute SQLite insert at row %d: %w", totalRows+1, err)
		}

		totalRows++
		batchCount++

		if batchCount >= batchSize {
			stmt.Close()
			if err := tx.Commit(); err != nil {
				return 0, fmt.Errorf("failed to commit batch: %w", err)
			}

			tx, err = dstDB.Begin()
			if err != nil {
				return 0, fmt.Errorf("failed to begin transaction: %w", err)
			}
			stmt, err = tx.Prepare(sqliteUpsertQuery)
			if err != nil {
				tx.Rollback()
				return 0, fmt.Errorf("failed to prepare statement: %w", err)
			}
			batchCount = 0
		}
	}

	stmt.Close()
	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("failed to commit final batch: %w", err)
	}
	if err := rows.Err(); err != nil {
		return 0, fmt.Errorf("error iterating source rows: %w", err)
	}

	return totalRows, nil
}

func migrateAsnames(srcDB *sql.DB, dstDB *sql.DB) (int, error) {
	asnRows, err := srcDB.Query("SELECT ASNUMBER, ASNAME, LOCALE FROM ASNUMNAME;")
	if err != nil {
		return 0, fmt.Errorf("failed to query source ASNUMNAME: %w", err)
	}
	defer asnRows.Close()

	asnTx, err := dstDB.Begin()
	if err != nil {
		return 0, fmt.Errorf("failed to begin ASNUMNAME transaction: %w", err)
	}
	if _, err := asnTx.Exec("DELETE FROM ASNUMNAME;"); err != nil {
		asnTx.Rollback()
		return 0, fmt.Errorf("failed to clear SQLite ASNUMNAME: %w", err)
	}
	asnStmt, err := asnTx.Prepare("INSERT INTO ASNUMNAME (ASNUMBER, ASNAME, LOCALE) VALUES (?, ?, ?);")
	if err != nil {
		asnTx.Rollback()
		return 0, fmt.Errorf("failed to prepare ASNUMNAME insert: %w", err)
	}

	asnCount := 0
	for asnRows.Next() {
		var asNum uint32
		var asName string
		var locale sql.NullString
		if err := asnRows.Scan(&asNum, &asName, &locale); err != nil {
			asnStmt.Close()
			asnTx.Rollback()
			return 0, fmt.Errorf("failed to scan ASNUMNAME row: %w", err)
		}
		var locVal interface{}
		if locale.Valid {
			locVal = locale.String
		}
		if _, err := asnStmt.Exec(asNum, asName, locVal); err != nil {
			asnStmt.Close()
			asnTx.Rollback()
			return 0, fmt.Errorf("failed to insert ASNUMNAME row: %w", err)
		}
		asnCount++
	}
	asnStmt.Close()
	if err := asnTx.Commit(); err != nil {
		return 0, fmt.Errorf("failed to commit ASNUMNAME transaction: %w", err)
	}

	return asnCount, nil
}

func verifyParity(srcDB *sql.DB, dstDB *sql.DB) error {
	type sourceStats struct {
		source  string
		count   int64
		minTime int64
		maxTime int64
	}

	getStats := func(db *sql.DB, query string) (map[string]sourceStats, error) {
		res := make(map[string]sourceStats)
		r, err := db.Query(query)
		if err != nil {
			return nil, err
		}
		defer r.Close()
		for r.Next() {
			var s sourceStats
			if err := r.Scan(&s.source, &s.count, &s.minTime, &s.maxTime); err != nil {
				return nil, err
			}
			res[s.source] = s
		}
		return res, r.Err()
	}

	statQuery := "SELECT source, COUNT(*), MIN(TIME), MAX(TIME) FROM INFO GROUP BY source ORDER BY source"
	sStats, err := getStats(srcDB, statQuery)
	if err != nil {
		return fmt.Errorf("failed to get source stats: %w", err)
	}
	dStats, err := getStats(dstDB, statQuery)
	if err != nil {
		return fmt.Errorf("failed to get destination stats: %w", err)
	}

	for src, ms := range sStats {
		ss, exists := dStats[src]
		if !exists {
			return fmt.Errorf("source %s missing from destination", src)
		}
		if ms.count != ss.count {
			return fmt.Errorf("source %s count mismatch (source=%d, dest=%d)", src, ms.count, ss.count)
		}
		if ms.minTime != ss.minTime {
			return fmt.Errorf("source %s min(TIME) mismatch (source=%d, dest=%d)", src, ms.minTime, ss.minTime)
		}
		if ms.maxTime != ss.maxTime {
			return fmt.Errorf("source %s max(TIME) mismatch (source=%d, dest=%d)", src, ms.maxTime, ss.maxTime)
		}
		log.Printf("Source %s parity verified: count=%d, min_time=%d, max_time=%d", src, ms.count, ms.minTime, ms.maxTime)
	}

	computeChecksum := func(db *sql.DB) (string, error) {
		h := sha256.New()
		r, err := db.Query("SELECT source, TIME, V4COUNT, V6COUNT, COALESCE(TWEET, 0) FROM INFO ORDER BY source, TIME")
		if err != nil {
			return "", err
		}
		defer r.Close()

		var src string
		var t, v4, v6, tweet int64
		for r.Next() {
			if err := r.Scan(&src, &t, &v4, &v6, &tweet); err != nil {
				return "", err
			}
			fmt.Fprintf(h, "%s|%d|%d|%d|%d\n", src, t, v4, v6, tweet)
		}
		return fmt.Sprintf("%x", h.Sum(nil)), r.Err()
	}

	srcHash, err := computeChecksum(srcDB)
	if err != nil {
		return fmt.Errorf("failed to compute source checksum: %w", err)
	}
	dstHash, err := computeChecksum(dstDB)
	if err != nil {
		return fmt.Errorf("failed to compute destination checksum: %w", err)
	}

	if srcHash != dstHash {
		return fmt.Errorf("checksum mismatch! source=%s, dest=%s", srcHash, dstHash)
	}
	log.Printf("100%% Row Checksum Match: %s", srcHash)

	var sAsnCount, dAsnCount int
	if err := srcDB.QueryRow("SELECT COUNT(*) FROM ASNUMNAME;").Scan(&sAsnCount); err != nil {
		return fmt.Errorf("failed to count source ASNUMNAME: %w", err)
	}
	if err := dstDB.QueryRow("SELECT COUNT(*) FROM ASNUMNAME;").Scan(&dAsnCount); err != nil {
		return fmt.Errorf("failed to count destination ASNUMNAME: %w", err)
	}
	if sAsnCount != dAsnCount {
		return fmt.Errorf("ASNUMNAME count mismatch (source=%d, dest=%d)", sAsnCount, dAsnCount)
	}
	log.Printf("ASNUMNAME parity verified: %d records", dAsnCount)

	return nil
}

func main() {
	configPath := flag.String("config", "/home/bgp/bgpsql/config.ini", "Path to bgpsql config.ini containing MySQL credentials")
	outPath := flag.String("out", "/var/lib/bgpsql/bgp.db", "Destination path for SQLite database file")
	delta := flag.Bool("delta", false, "Delta mode: only copy rows newer than newest in SQLite file")
	batchSize := flag.Int("batch-size", 10000, "Transaction batch size for inserts")
	flag.Parse()

	startTime := time.Now()
	log.Printf("Starting bgpsql-migrate (delta=%v, batch_size=%d)", *delta, *batchSize)
	log.Printf("Source config: %s", *configPath)
	log.Printf("Target SQLite: %s", *outPath)

	cf, err := ini.Load(*configPath)
	if err != nil {
		log.Fatalf("Failed to read config file %s: %v", *configPath, err)
	}
	myUser := cf.Section("sql").Key("username").String()
	myPass := cf.Section("sql").Key("password").String()
	myDB := cf.Section("sql").Key("database").String()
	if myUser == "" || myDB == "" {
		log.Fatalf("Missing MySQL username or database in config %s", *configPath)
	}

	mysqlDSN := fmt.Sprintf("%s:%s@tcp(127.0.0.1:3306)/%s", myUser, myPass, myDB)
	mysqlDB, err := sql.Open("mysql", mysqlDSN)
	if err != nil {
		log.Fatalf("Failed to open MariaDB connection: %v", err)
	}
	defer mysqlDB.Close()
	if err := mysqlDB.Ping(); err != nil {
		log.Fatalf("Failed to ping MariaDB: %v", err)
	}
	log.Println("Connected to MariaDB successfully")

	if err := os.MkdirAll(filepath.Dir(*outPath), 0750); err != nil {
		log.Fatalf("Failed to create destination directory: %v", err)
	}

	sqliteDSN := fmt.Sprintf("file:%s?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)&_pragma=foreign_keys(ON)&_pragma=cache_size(-16000)", *outPath)
	sqliteDB, err := sql.Open("sqlite", sqliteDSN)
	if err != nil {
		log.Fatalf("Failed to open SQLite database: %v", err)
	}
	defer sqliteDB.Close()
	sqliteDB.SetMaxOpenConns(1)

	if _, err := sqliteDB.Exec("PRAGMA journal_mode=WAL;"); err != nil {
		log.Fatalf("Failed to set WAL mode: %v", err)
	}

	var userVer int
	if err := sqliteDB.QueryRow("PRAGMA user_version;").Scan(&userVer); err != nil {
		log.Fatalf("Failed to check user_version: %v", err)
	}
	if userVer < 3 {
		log.Println("Applying SQLite schema migration 003...")
		if _, err := sqliteDB.Exec(sqliteSchemaDDL); err != nil {
			log.Fatalf("Failed to apply SQLite schema: %v", err)
		}
		log.Println("SQLite schema applied successfully")
	}

	totalRows, err := migrateInfo(mysqlDB, sqliteDB, true, *delta, *batchSize)
	if err != nil {
		log.Fatalf("Migration of INFO failed: %v", err)
	}
	log.Printf("Successfully migrated %d rows to SQLite INFO", totalRows)

	asnCount, err := migrateAsnames(mysqlDB, sqliteDB)
	if err != nil {
		log.Fatalf("Migration of ASNUMNAME failed: %v", err)
	}
	log.Printf("Successfully migrated %d ASNUMNAME records", asnCount)

	log.Println("Running automatic parity verification...")
	if err := verifyParity(mysqlDB, sqliteDB); err != nil {
		log.Fatalf("VERIFICATION FAILURE: %v", err)
	}

	log.Println("Running PRAGMA integrity_check...")
	var checkResult string
	if err := sqliteDB.QueryRow("PRAGMA integrity_check;").Scan(&checkResult); err != nil {
		log.Fatalf("PRAGMA integrity_check query failed: %v", err)
	}
	if checkResult != "ok" {
		log.Fatalf("PRAGMA integrity_check returned: %s", checkResult)
	}
	log.Println("PRAGMA integrity_check: ok")

	log.Println("Running SQLite ANALYZE...")
	if _, err := sqliteDB.Exec("ANALYZE;"); err != nil {
		log.Fatalf("ANALYZE failed: %v", err)
	}
	log.Println("ANALYZE completed successfully")

	fi, err := os.Stat(*outPath)
	var fileSize int64 = 0
	if err == nil {
		fileSize = fi.Size()
	}

	elapsed := time.Since(startTime)
	log.Printf("=== MIGRATION COMPLETE ===")
	log.Printf("Wall time: %s", elapsed)
	log.Printf("Total rows in INFO: %d", totalRows)
	log.Printf("SQLite file size: %.2f MB (%d bytes)", float64(fileSize)/(1024*1024), fileSize)
}
