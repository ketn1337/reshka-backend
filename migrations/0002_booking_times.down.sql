-- 0002_booking_times.down.sql
ALTER TABLE bookings DROP CONSTRAINT IF EXISTS no_double_book;

ALTER TABLE bookings ADD CONSTRAINT no_double_book
  EXCLUDE USING gist (
    room_id WITH =,
    daterange(check_in, check_out, '[)') WITH &&
  ) WHERE (status IN ('new','confirmed','checked_in'));

ALTER TABLE bookings
  DROP COLUMN IF EXISTS check_out_time,
  DROP COLUMN IF EXISTS check_in_time;
