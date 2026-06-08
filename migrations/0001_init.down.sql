-- 0001_init.down.sql

DROP TABLE IF EXISTS rates;
DROP TABLE IF EXISTS booking_status_history;
DROP TABLE IF EXISTS bookings;
DROP TABLE IF EXISTS guests;
DROP TABLE IF EXISTS photos;
DROP TABLE IF EXISTS rooms;
DROP TABLE IF EXISTS room_kinds;
DROP TABLE IF EXISTS properties;
DROP TABLE IF EXISTS users;

DROP TYPE IF EXISTS room_orientation;
DROP TYPE IF EXISTS booking_source;
DROP TYPE IF EXISTS booking_status;
DROP TYPE IF EXISTS user_role;
