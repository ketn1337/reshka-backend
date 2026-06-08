-- 0003_bnovo_link.up.sql
-- Связь с Bnovo PMS: храним исходный bnovo_id (для группировки мульти-номерных
-- бронирований) и человекочитаемый bnovo_number.
-- Колонки nullable, чтобы старые ручные брони жили без них.

ALTER TABLE bookings
  ADD COLUMN IF NOT EXISTS bnovo_id     TEXT,
  ADD COLUMN IF NOT EXISTS bnovo_number TEXT;

-- Частичный индекс: быстрый поиск «все мои брони, импортированные из этой Bnovo-брони».
CREATE INDEX IF NOT EXISTS idx_bookings_bnovo_id
    ON bookings(bnovo_id) WHERE bnovo_id IS NOT NULL;
