-- Migration: 002_create_curated_view.sql
-- Description:
--   Creates or replaces the curated_samples SQL view for DATA-6.
--   Curates exactly one canonical row per timestamp:
--     1. Prefers BGP3 if quality = 'ok'
--     2. Falls back to BGP1 if quality = 'ok'
--     3. Leaves a gap if neither source has quality = 'ok'
--   Includes derived_from column identifying which router provided the record.

CREATE OR REPLACE VIEW curated_samples AS
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
