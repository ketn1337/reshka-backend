-- 0002_booking_times.up.sql
-- Добавляем время заезда/выезда в бронирования и переводим EXCLUDE-constraint
-- с daterange на tsrange, чтобы turnover в один день (выезд 12:00 + заезд 14:00)
-- не блокировался.

ALTER TABLE bookings
  ADD COLUMN IF NOT EXISTS check_in_time  TIME NOT NULL DEFAULT '14:00:00',
  ADD COLUMN IF NOT EXISTS check_out_time TIME NOT NULL DEFAULT '12:00:00';

-- старый date-констрейнт снимаем
ALTER TABLE bookings DROP CONSTRAINT IF EXISTS no_double_book;

-- новый timestamp-констрейнт: одна активная бронь на комнату в любой момент
ALTER TABLE bookings ADD CONSTRAINT no_double_book
  EXCLUDE USING gist (
    room_id WITH =,
    tsrange(
      (check_in  + check_in_time),
      (check_out + check_out_time),
      '[)'
    ) WITH &&
  ) WHERE (status IN ('new','confirmed','checked_in'));
