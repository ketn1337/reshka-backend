-- 0001_init.up.sql
-- PMS «Орёл и Решка»: начальная схема

CREATE EXTENSION IF NOT EXISTS citext;
CREATE EXTENSION IF NOT EXISTS btree_gist;

DO $$ BEGIN
    CREATE TYPE user_role AS ENUM ('admin', 'manager', 'receptionist');
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

DO $$ BEGIN
    CREATE TYPE booking_status AS ENUM (
        'new', 'confirmed', 'checked_in', 'checked_out', 'cancelled', 'no_show'
    );
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

DO $$ BEGIN
    CREATE TYPE booking_source AS ENUM (
        'direct', 'site', 'ota', 'phone', 'max'
    );
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

DO $$ BEGIN
    CREATE TYPE room_orientation AS ENUM ('inner', 'street', 'courtyard');
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

-- =========================
-- users
-- =========================
CREATE TABLE IF NOT EXISTS users (
    id            BIGSERIAL PRIMARY KEY,
    email         CITEXT UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    role          user_role NOT NULL,
    full_name     TEXT NOT NULL,
    is_active     BOOLEAN NOT NULL DEFAULT true,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- =========================
-- properties
-- =========================
CREATE TABLE IF NOT EXISTS properties (
    id          BIGSERIAL PRIMARY KEY,
    slug        TEXT UNIQUE NOT NULL,
    title       TEXT NOT NULL,
    short_title TEXT NOT NULL,
    address     TEXT NOT NULL,
    description TEXT,
    accent      TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- =========================
-- room_kinds
-- =========================
CREATE TABLE IF NOT EXISTS room_kinds (
    id          BIGSERIAL PRIMARY KEY,
    property_id BIGINT NOT NULL REFERENCES properties(id) ON DELETE CASCADE,
    slug        TEXT NOT NULL,
    title       TEXT NOT NULL,
    description TEXT,
    base_rate   NUMERIC(10,2) NOT NULL,
    capacity    INT NOT NULL,
    area        NUMERIC(5,1) NOT NULL,
    beds        TEXT NOT NULL,
    UNIQUE (property_id, slug)
);

-- =========================
-- rooms
-- =========================
CREATE TABLE IF NOT EXISTS rooms (
    id          BIGSERIAL PRIMARY KEY,
    property_id BIGINT NOT NULL REFERENCES properties(id) ON DELETE CASCADE,
    kind_id     BIGINT NOT NULL REFERENCES room_kinds(id) ON DELETE RESTRICT,
    label       TEXT NOT NULL,
    short_label TEXT NOT NULL,
    floor       INT NOT NULL,
    side        TEXT,
    area        NUMERIC(5,1),
    orientation room_orientation,
    is_active   BOOLEAN NOT NULL DEFAULT true,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (property_id, short_label)
);
CREATE INDEX IF NOT EXISTS idx_rooms_property_kind
    ON rooms(property_id, kind_id) WHERE is_active;

-- =========================
-- photos
-- =========================
CREATE TABLE IF NOT EXISTS photos (
    id         BIGSERIAL PRIMARY KEY,
    room_id    BIGINT NOT NULL REFERENCES rooms(id) ON DELETE CASCADE,
    filename   TEXT NOT NULL,
    position   INT NOT NULL DEFAULT 0,
    is_cover   BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (room_id, filename)
);
CREATE INDEX IF NOT EXISTS idx_photos_room ON photos(room_id, position);

-- =========================
-- guests
-- =========================
CREATE TABLE IF NOT EXISTS guests (
    id         BIGSERIAL PRIMARY KEY,
    full_name  TEXT NOT NULL,
    phone      TEXT,
    email      TEXT,
    doc_type   TEXT,
    doc_number TEXT,
    notes      TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_guests_phone ON guests(phone) WHERE phone IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_guests_email ON guests(email) WHERE email IS NOT NULL;

-- =========================
-- bookings
-- =========================
CREATE TABLE IF NOT EXISTS bookings (
    id           BIGSERIAL PRIMARY KEY,
    code         TEXT UNIQUE NOT NULL,
    room_id      BIGINT NOT NULL REFERENCES rooms(id) ON DELETE RESTRICT,
    guest_id     BIGINT REFERENCES guests(id) ON DELETE SET NULL,
    check_in     DATE NOT NULL,
    check_out    DATE NOT NULL,
    adults       INT NOT NULL DEFAULT 1,
    status       booking_status NOT NULL DEFAULT 'new',
    source       booking_source NOT NULL DEFAULT 'site',
    total_amount NUMERIC(10,2) NOT NULL,
    prepayment   NUMERIC(10,2) NOT NULL DEFAULT 0,
    notes        TEXT,
    created_by   BIGINT REFERENCES users(id) ON DELETE SET NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (check_out > check_in),
    CHECK (adults >= 1)
);
CREATE INDEX IF NOT EXISTS idx_bookings_dates ON bookings(check_in, check_out);
CREATE INDEX IF NOT EXISTS idx_bookings_room_dates
    ON bookings(room_id, check_in, check_out);
CREATE INDEX IF NOT EXISTS idx_bookings_status ON bookings(status);

-- Защита от овербукинга: один номер — одна активная бронь в любой момент.
ALTER TABLE bookings DROP CONSTRAINT IF EXISTS no_double_book;
ALTER TABLE bookings ADD CONSTRAINT no_double_book
    EXCLUDE USING gist (
        room_id WITH =,
        daterange(check_in, check_out, '[)') WITH &&
    ) WHERE (status IN ('new', 'confirmed', 'checked_in'));

-- =========================
-- booking_status_history
-- =========================
CREATE TABLE IF NOT EXISTS booking_status_history (
    id          BIGSERIAL PRIMARY KEY,
    booking_id  BIGINT NOT NULL REFERENCES bookings(id) ON DELETE CASCADE,
    from_status booking_status,
    to_status   booking_status NOT NULL,
    changed_by  BIGINT REFERENCES users(id) ON DELETE SET NULL,
    reason      TEXT,
    changed_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_status_hist_booking
    ON booking_status_history(booking_id, changed_at);

-- =========================
-- rates
-- =========================
CREATE TABLE IF NOT EXISTS rates (
    id           BIGSERIAL PRIMARY KEY,
    kind_id      BIGINT NOT NULL REFERENCES room_kinds(id) ON DELETE CASCADE,
    date_from    DATE NOT NULL,
    date_to      DATE NOT NULL,
    weekday_rate NUMERIC(10,2) NOT NULL,
    weekend_rate NUMERIC(10,2) NOT NULL,
    CHECK (date_to >= date_from)
);
CREATE INDEX IF NOT EXISTS idx_rates_kind_dates
    ON rates(kind_id, date_from, date_to);
