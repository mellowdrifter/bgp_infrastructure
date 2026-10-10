package main

import (
	"database/sql"
	"os"
	"testing"

	_ "modernc.org/sqlite"
)

func TestMigrateFixture(t *testing.T) {
	srcDBPath := "./test_src.db"
	dstDBPath := "./test_dst.db"
	defer os.Remove(srcDBPath)
	defer os.Remove(srcDBPath + "-wal")
	defer os.Remove(srcDBPath + "-shm")
	defer os.Remove(dstDBPath)
	defer os.Remove(dstDBPath + "-wal")
	defer os.Remove(dstDBPath + "-shm")

	srcDB, err := sql.Open("sqlite", srcDBPath)
	if err != nil {
		t.Fatalf("Open src failed: %v", err)
	}
	defer srcDB.Close()

	if _, err := srcDB.Exec(sqliteSchemaDDL); err != nil {
		t.Fatalf("Init src schema failed: %v", err)
	}

	// Seed source INFO with 25 samples
	for i := 0; i < 25; i++ {
		src := "bgp1"
		if i%2 == 0 {
			src = "bgp3"
		}
		ts := int64(1710000000 + i*300)
		_, err := srcDB.Exec(`INSERT INTO INFO (source, TIME, quality, quality_note, V4COUNT, V6COUNT, V4TOTAL, V6TOTAL, TWEET, ingested_at)
			VALUES (?, ?, 'ok', 'clean', ?, ?, 1000000, 200000, 1, ?)`,
			src, ts, 1000000+i, 200000+i, ts+10)
		if err != nil {
			t.Fatalf("Insert src row %d failed: %v", i, err)
		}
	}

	// Seed source ASNUMNAME
	asns := []struct {
		num    uint32
		name   string
		locale string
	}{
		{15169, "GOOGLE", "US"},
		{13335, "CLOUDFLARENET", "US"},
		{2914, "NTT", "JP"},
	}
	for _, a := range asns {
		if _, err := srcDB.Exec("INSERT INTO ASNUMNAME (ASNUMBER, ASNAME, LOCALE) VALUES (?, ?, ?)", a.num, a.name, a.locale); err != nil {
			t.Fatalf("Insert AS%d failed: %v", a.num, err)
		}
	}

	// Open destination
	dstDB, err := sql.Open("sqlite", dstDBPath)
	if err != nil {
		t.Fatalf("Open dst failed: %v", err)
	}
	defer dstDB.Close()

	if _, err := dstDB.Exec(sqliteSchemaDDL); err != nil {
		t.Fatalf("Init dst schema failed: %v", err)
	}

	// Run migration (isSrcMySQL=false for fixture test)
	totalRows, err := migrateInfo(srcDB, dstDB, false, false, 10)
	if err != nil {
		t.Fatalf("migrateInfo failed: %v", err)
	}
	if totalRows != 25 {
		t.Errorf("Expected 25 rows migrated, got %d", totalRows)
	}

	asnCount, err := migrateAsnames(srcDB, dstDB)
	if err != nil {
		t.Fatalf("migrateAsnames failed: %v", err)
	}
	if asnCount != 3 {
		t.Errorf("Expected 3 asnames migrated, got %d", asnCount)
	}

	// Verify parity
	if err := verifyParity(srcDB, dstDB); err != nil {
		t.Fatalf("verifyParity failed: %v", err)
	}

	// Test delta mode: add 5 new rows to src
	for i := 25; i < 30; i++ {
		ts := int64(1710000000 + i*300)
		_, err := srcDB.Exec(`INSERT INTO INFO (source, TIME, quality, quality_note, V4COUNT, V6COUNT, V4TOTAL, V6TOTAL, TWEET, ingested_at)
			VALUES ('bgp3', ?, 'ok', 'clean', ?, ?, 1000000, 200000, 0, ?)`,
			ts, 1000000+i, 200000+i, ts+10)
		if err != nil {
			t.Fatalf("Insert delta row %d failed: %v", i, err)
		}
	}

	deltaRows, err := migrateInfo(srcDB, dstDB, false, true, 10)
	if err != nil {
		t.Fatalf("delta migrateInfo failed: %v", err)
	}
	if deltaRows != 5 {
		t.Errorf("Expected 5 delta rows migrated, got %d", deltaRows)
	}

	if err := verifyParity(srcDB, dstDB); err != nil {
		t.Fatalf("verifyParity after delta failed: %v", err)
	}
}
