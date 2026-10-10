-- Migration: 003_sqlite_schema.sql
-- Description:
--   Pure SQLite schema for bgpsql storage migration (DATA-7).
--   Uses WITHOUT ROWID for compact B-tree storage keyed on (source, TIME),
--   quality check constraints, unix timestamp integer for ingested_at,
--   curated_samples view with source failover, and PRAGMA user_version.

CREATE TABLE IF NOT EXISTS INFO (
    source TEXT NOT NULL,
    TIME INTEGER NOT NULL,
    quality TEXT NOT NULL DEFAULT 'ok' CHECK (quality IN ('ok', 'suspect', 'bad')),
    quality_note TEXT,
    V4COUNT INTEGER NOT NULL,
    V6COUNT INTEGER NOT NULL,
    V4TOTAL INTEGER,
    V6TOTAL INTEGER,
    PEERS_CONFIGURED INTEGER,
    PEERS_UP INTEGER,
    PEERS6_CONFIGURED INTEGER,
    PEERS6_UP INTEGER,
    V4_24 INTEGER,
    V4_23 INTEGER,
    V4_22 INTEGER,
    V4_21 INTEGER,
    V4_20 INTEGER,
    V4_19 INTEGER,
    V4_18 INTEGER,
    V4_17 INTEGER,
    V4_16 INTEGER,
    V4_15 INTEGER,
    V4_14 INTEGER,
    V4_13 INTEGER,
    V4_12 INTEGER,
    V4_11 INTEGER,
    V4_10 INTEGER,
    V4_09 INTEGER,
    V4_08 INTEGER,
    V6_48 INTEGER,
    V6_47 INTEGER,
    V6_46 INTEGER,
    V6_45 INTEGER,
    V6_44 INTEGER,
    V6_43 INTEGER,
    V6_42 INTEGER,
    V6_41 INTEGER,
    V6_40 INTEGER,
    V6_39 INTEGER,
    V6_38 INTEGER,
    V6_37 INTEGER,
    V6_36 INTEGER,
    V6_35 INTEGER,
    V6_34 INTEGER,
    V6_33 INTEGER,
    V6_32 INTEGER,
    V6_31 INTEGER,
    V6_30 INTEGER,
    V6_29 INTEGER,
    V6_28 INTEGER,
    V6_27 INTEGER,
    V6_26 INTEGER,
    V6_25 INTEGER,
    V6_24 INTEGER,
    V6_23 INTEGER,
    V6_22 INTEGER,
    V6_21 INTEGER,
    V6_20 INTEGER,
    V6_19 INTEGER,
    V6_18 INTEGER,
    V6_17 INTEGER,
    V6_16 INTEGER,
    V6_15 INTEGER,
    V6_14 INTEGER,
    V6_13 INTEGER,
    V6_12 INTEGER,
    V6_11 INTEGER,
    V6_10 INTEGER,
    V6_09 INTEGER,
    V6_08 INTEGER,
    AS4_LEN INTEGER,
    AS6_LEN INTEGER,
    AS10_LEN INTEGER,
    AS4_ONLY INTEGER,
    AS6_ONLY INTEGER,
    AS_BOTH INTEGER,
    LARGEC4 INTEGER,
    LARGEC6 INTEGER,
    ROAVALIDV4 INTEGER,
    ROAINVALIDV4 INTEGER,
    ROAUNKNOWNV4 INTEGER,
    ROAVALIDV6 INTEGER,
    ROAINVALIDV6 INTEGER,
    ROAUNKNOWNV6 INTEGER,
    TWEET INTEGER NOT NULL DEFAULT 0,
    ingested_at INTEGER NOT NULL DEFAULT (unixepoch()),
    PRIMARY KEY (source, TIME)
) WITHOUT ROWID;

CREATE INDEX IF NOT EXISTS idx_info_time ON INFO (TIME);

CREATE TABLE IF NOT EXISTS ASNUMNAME (
    ASNUMBER INTEGER PRIMARY KEY,
    ASNAME TEXT NOT NULL,
    LOCALE TEXT
);

DROP VIEW IF EXISTS curated_samples;
CREATE VIEW curated_samples AS
SELECT 
    i.source AS derived_from,
    i.*
FROM INFO i
WHERE i.quality = 'ok'
  AND (
    i.source = 'bgp3'
    OR (
      i.source = 'bgp1'
      AND NOT EXISTS (
        SELECT 1 FROM INFO i2 
        WHERE i2.source = 'bgp3' 
          AND i2.TIME = i.TIME 
          AND i2.quality = 'ok'
      )
    )
  );

PRAGMA user_version = 3;
