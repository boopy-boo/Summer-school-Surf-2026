-- ============================================================================
-- Откат миграции 001: удаление схемы БД
-- ============================================================================

BEGIN;

DROP VIEW IF EXISTS v_bookings_full;
DROP VIEW IF EXISTS v_slots_with_program;

DROP TABLE IF EXISTS push_tokens;
DROP TABLE IF EXISTS otp_codes;

DROP TRIGGER IF EXISTS trg_booking_change ON bookings;
DROP FUNCTION IF EXISTS fn_update_slot_free_seats_on_booking();

DROP TABLE IF EXISTS bookings;

DROP TABLE IF EXISTS clients;

DROP TRIGGER IF EXISTS trg_check_slot_capacity_cap ON slots;
DROP FUNCTION IF EXISTS fn_check_slot_capacity_cap();

DROP TABLE IF EXISTS slots;

DROP TABLE IF EXISTS masters;
DROP TABLE IF EXISTS programs;

COMMIT;