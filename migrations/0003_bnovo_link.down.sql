-- 0003_bnovo_link.down.sql
DROP INDEX IF EXISTS idx_bookings_bnovo_id;

ALTER TABLE bookings
  DROP COLUMN IF EXISTS bnovo_number,
  DROP COLUMN IF EXISTS bnovo_id;
