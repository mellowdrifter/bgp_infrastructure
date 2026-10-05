-- Migration: 001_add_source_quality.sql
-- Description:
--   1. Deduplicate legacy 5-minute bucket collisions (retains latest sample per bucket).
--   2. Truncate sample timestamps to 5-minute boundaries.
--   3. Add source, quality, quality_note, and ingested_at columns.
--   4. Switch primary key to (source, TIME) and add secondary index idx_info_time (TIME DESC).
--
-- Usage:
--   mysql -u root -p bgp_statistics -e "SET @source_node = 'bgp3'; SOURCE /path/to/001_add_source_quality.sql;"
--   (If @source_node is not explicitly set, it defaults to 'bgp1')

SET @source_node = COALESCE(@source_node, 'bgp1');

-- 1. Deduplicate legacy 5-minute bucket collisions
CREATE TEMPORARY TABLE IF NOT EXISTS duplicate_buckets AS
SELECT (TIME - (TIME % 300)) AS bucket
FROM INFO
GROUP BY (TIME - (TIME % 300))
HAVING COUNT(*) > 1;

CREATE TEMPORARY TABLE IF NOT EXISTS to_delete AS
SELECT i.TIME
FROM INFO i
JOIN duplicate_buckets d ON i.TIME >= d.bucket AND i.TIME < d.bucket + 300
WHERE i.TIME NOT IN (
    SELECT MAX(i2.TIME)
    FROM INFO i2
    JOIN duplicate_buckets d2 ON i2.TIME >= d2.bucket AND i2.TIME < d2.bucket + 300
    GROUP BY d2.bucket
);

DELETE FROM INFO WHERE TIME IN (SELECT TIME FROM to_delete);
DROP TEMPORARY TABLE IF EXISTS to_delete;
DROP TEMPORARY TABLE IF EXISTS duplicate_buckets;

-- 2. Truncate timestamps to 5-minute boundaries
UPDATE INFO SET TIME = TIME - (TIME % 300) WHERE (TIME % 300) != 0;

-- 3. Add columns and update primary key if 'source' does not exist yet
SET @col_exists = (
    SELECT COUNT(*) 
    FROM information_schema.columns 
    WHERE table_schema = DATABASE() 
      AND table_name = 'INFO' 
      AND column_name = 'source'
);

SET @sql_alter = IF(@col_exists = 0,
    CONCAT(
        "ALTER TABLE INFO ",
        "ADD COLUMN source VARCHAR(16) NOT NULL DEFAULT '", @source_node, "', ",
        "ADD COLUMN quality ENUM('ok', 'suspect', 'bad') NOT NULL DEFAULT 'ok', ",
        "ADD COLUMN quality_note VARCHAR(255) DEFAULT NULL, ",
        "ADD COLUMN ingested_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP, ",
        "DROP PRIMARY KEY, ",
        "ADD PRIMARY KEY (source, TIME), ",
        "ADD INDEX idx_info_time (TIME DESC);"
    ),
    "SELECT 'Column source already exists; skipping ALTER TABLE.' AS status;"
);

PREPARE stmt FROM @sql_alter;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;
