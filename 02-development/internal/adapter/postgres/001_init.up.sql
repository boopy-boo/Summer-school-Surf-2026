-- ============================================================================
-- Миграция 001: Инициализация схемы БД «Гончарная мастерская»
-- ============================================================================

BEGIN;

CREATE TABLE IF NOT EXISTS programs (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name            VARCHAR(255) NOT NULL,
    description     TEXT,
    type            VARCHAR(20) NOT NULL,
    capacity_cap    INT NOT NULL,
    duration_minutes INT NOT NULL DEFAULT 150,
    CONSTRAINT chk_program_type CHECK (type IN ('handbuilding', 'wheel')),
    CONSTRAINT chk_program_capacity_positive CHECK (capacity_cap > 0),
    CONSTRAINT chk_program_capacity_by_type CHECK (
        (type = 'handbuilding' AND capacity_cap <= 6) OR
        (type = 'wheel' AND capacity_cap <= 10)
    ),
    CONSTRAINT chk_program_duration_positive CHECK (duration_minutes > 0)
);

CREATE TABLE IF NOT EXISTS masters (
    id   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL
);

CREATE TABLE IF NOT EXISTS slots (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    program_id        UUID NOT NULL REFERENCES programs(id) ON DELETE RESTRICT,
    master_id         UUID NOT NULL REFERENCES masters(id) ON DELETE RESTRICT,
    start_at          TIMESTAMPTZ NOT NULL,
    total_seats       INT NOT NULL,
    free_seats        INT NOT NULL,
    free_rental_kits  INT NOT NULL DEFAULT 0,
    price             DECIMAL(10,2) NOT NULL,
    rental_price      DECIMAL(10,2) NOT NULL DEFAULT 0.00,
    workshop_address  VARCHAR(500) NOT NULL DEFAULT '',
    workshop_lat      DECIMAL(10, 8),
    workshop_lng      DECIMAL(11, 8),
    status            VARCHAR(20) NOT NULL DEFAULT 'active',
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_slot_status CHECK (status IN ('active', 'filled', 'cancelled')),
    CONSTRAINT chk_slot_total_seats_positive CHECK (total_seats > 0),
    CONSTRAINT chk_slot_free_seats_not_negative CHECK (free_seats >= 0),
    CONSTRAINT chk_slot_free_seats_lte_total CHECK (free_seats <= total_seats),
    CONSTRAINT chk_slot_free_rental_not_negative CHECK (free_rental_kits >= 0),
    CONSTRAINT chk_slot_price_not_negative CHECK (price >= 0),
    CONSTRAINT chk_slot_rental_price_not_negative CHECK (rental_price >= 0)
);

CREATE INDEX idx_slots_start_at ON slots(start_at);
CREATE INDEX idx_slots_status ON slots(status) WHERE status = 'active';
CREATE INDEX idx_slots_program_master ON slots(program_id, master_id);

CREATE OR REPLACE FUNCTION fn_check_slot_capacity_cap()
RETURNS TRIGGER AS $$
DECLARE
    v_cap INT;
BEGIN
    SELECT capacity_cap INTO v_cap FROM programs WHERE id = NEW.program_id;
    IF NEW.total_seats > v_cap THEN
        RAISE EXCEPTION 'total_seats (%) exceeds program capacity_cap (%) (R-003, FR-13)',
            NEW.total_seats, v_cap;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_check_slot_capacity_cap ON slots;
CREATE TRIGGER trg_check_slot_capacity_cap
    BEFORE INSERT OR UPDATE ON slots
    FOR EACH ROW
    EXECUTE FUNCTION fn_check_slot_capacity_cap();

CREATE TABLE IF NOT EXISTS clients (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        VARCHAR(255) NOT NULL,
    phone       VARCHAR(20) NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_client_phone_format CHECK (phone ~ '^\+[1-9]\d{7,14}$'),
    CONSTRAINT uq_client_phone UNIQUE (phone)
);

CREATE INDEX idx_clients_phone ON clients(phone);

CREATE TABLE IF NOT EXISTS bookings (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    slot_id             UUID NOT NULL REFERENCES slots(id) ON DELETE RESTRICT,
    client_id           UUID NOT NULL REFERENCES clients(id) ON DELETE CASCADE,
    seats_count         INT NOT NULL,
    rental_count        INT NOT NULL DEFAULT 0,
    price_total         DECIMAL(10,2) NOT NULL,
    status              VARCHAR(30) NOT NULL DEFAULT 'active',
    cancellation_reason VARCHAR(500),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    cancelled_at        TIMESTAMPTZ,
    CONSTRAINT chk_booking_status CHECK (
        status IN ('active', 'cancelled', 'late_cancel', 'workshop_cancelled')
    ),
    CONSTRAINT chk_booking_seats_count CHECK (seats_count BETWEEN 1 AND 3),
    CONSTRAINT chk_booking_rental_count_not_negative CHECK (rental_count >= 0),
    CONSTRAINT chk_booking_rental_lte_seats CHECK (rental_count <= seats_count),
    CONSTRAINT chk_booking_price_total_not_negative CHECK (price_total >= 0),
    CONSTRAINT chk_booking_cancelled_at CHECK (
        (status IN ('cancelled', 'late_cancel', 'workshop_cancelled') AND cancelled_at IS NOT NULL) OR
        (status = 'active' AND cancelled_at IS NULL)
    ),
    CONSTRAINT chk_booking_cancel_reason CHECK (
        (status = 'workshop_cancelled' AND cancellation_reason IS NOT NULL AND cancellation_reason <> '') OR
        (status != 'workshop_cancelled')
    )
);

CREATE INDEX idx_bookings_client ON bookings(client_id, created_at DESC);
CREATE INDEX idx_bookings_slot ON bookings(slot_id);
CREATE INDEX idx_bookings_status ON bookings(status) WHERE status = 'active';

CREATE OR REPLACE FUNCTION fn_update_slot_free_seats_on_booking()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'INSERT' AND NEW.status = 'active' THEN
        UPDATE slots SET
            free_seats = free_seats - NEW.seats_count,
            free_rental_kits = free_rental_kits - NEW.rental_count
        WHERE id = NEW.slot_id;
    ELSIF TG_OP = 'UPDATE' AND OLD.status = 'active' AND NEW.status IN ('cancelled', 'late_cancel', 'workshop_cancelled') THEN
        IF NEW.status = 'cancelled' THEN
            UPDATE slots SET
                free_seats = free_seats + OLD.seats_count,
                free_rental_kits = free_rental_kits + OLD.rental_count
            WHERE id = OLD.slot_id;
        END IF;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_booking_change ON bookings;
CREATE TRIGGER trg_booking_change
    AFTER INSERT OR UPDATE ON bookings
    FOR EACH ROW
    EXECUTE FUNCTION fn_update_slot_free_seats_on_booking();

CREATE TABLE IF NOT EXISTS otp_codes (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    phone       VARCHAR(20) NOT NULL,
    code        VARCHAR(10) NOT NULL,
    attempts    INT NOT NULL DEFAULT 0,
    max_attempts INT NOT NULL DEFAULT 3,
    verified    BOOLEAN NOT NULL DEFAULT FALSE,
    expires_at  TIMESTAMPTZ NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_otp_attempts_not_negative CHECK (attempts >= 0),
    CONSTRAINT chk_otp_max_attempts_positive CHECK (max_attempts > 0)
);

CREATE INDEX idx_otp_phone ON otp_codes(phone, created_at DESC);
CREATE INDEX idx_otp_expires ON otp_codes(expires_at) WHERE verified = FALSE;

CREATE TABLE IF NOT EXISTS push_tokens (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    client_id   UUID NOT NULL REFERENCES clients(id) ON DELETE CASCADE,
    token       VARCHAR(500) NOT NULL,
    platform    VARCHAR(20) NOT NULL DEFAULT 'unknown',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_push_platform CHECK (platform IN ('ios', 'android', 'unknown')),
    CONSTRAINT uq_push_token_per_client UNIQUE (client_id, token)
);

CREATE OR REPLACE VIEW v_slots_with_program AS
SELECT
    s.*,
    p.name AS program_name,
    p.type AS program_type,
    p.description AS program_description,
    p.capacity_cap AS program_capacity_cap,
    p.duration_minutes AS program_duration_minutes,
    m.name AS master_name
FROM slots s
JOIN programs p ON s.program_id = p.id
JOIN masters m ON s.master_id = m.id;

CREATE OR REPLACE VIEW v_bookings_full AS
SELECT
    b.*,
    s.start_at AS slot_start_at,
    s.workshop_address,
    s.price AS slot_price,
    s.rental_price AS slot_rental_price,
    p.name AS program_name,
    p.type AS program_type,
    m.name AS master_name
FROM bookings b
JOIN slots s ON b.slot_id = s.id
JOIN programs p ON s.program_id = p.id
JOIN masters m ON s.master_id = m.id;

COMMIT;