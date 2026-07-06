-- ============================================================================
-- Миграция 001: Инициализация схемы БД «Гончарная мастерская»
-- 
-- Скоуп: клиентское API (BFF). Данные мастеров/программ/слотов — read-only
-- mirror из существующего бэкенда или полная схема для тестового окружения.
-- 
-- Привязка к требованиям ТЗ: FR-13, FR-14, FR-17, R-003 и др.
-- ============================================================================

BEGIN;

-- ============================================================================
-- 1. Справочники (read-only проекция из существующей инфраструктуры)
-- ============================================================================

CREATE TABLE IF NOT EXISTS programs (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name            VARCHAR(255) NOT NULL,
    description     TEXT,
    type            VARCHAR(20) NOT NULL,
    capacity_cap    INT NOT NULL,
    duration_minutes INT NOT NULL DEFAULT 150,

    -- R-003: лепка ≤ 6, гончарный круг ≤ 10
    -- FR-13: потолок вместимости программы участвует в расчёте max_seats
    CONSTRAINT chk_program_type CHECK (type IN ('handbuilding', 'wheel')),
    CONSTRAINT chk_program_capacity_positive CHECK (capacity_cap > 0),
    CONSTRAINT chk_program_capacity_by_type CHECK (
        (type = 'handbuilding' AND capacity_cap <= 6) OR
        (type = 'wheel' AND capacity_cap <= 10)
    ),
    CONSTRAINT chk_program_duration_positive CHECK (duration_minutes > 0)
);

COMMENT ON TABLE programs IS 'Справочник программ занятий (read-only проекция, R-003, FR-9a)';
COMMENT ON COLUMN programs.capacity_cap IS 'Потолок мест: лепка ≤6, круг ≤10 (R-003, FR-13)';
COMMENT ON COLUMN programs.type IS 'handbuilding — лепка, wheel — гончарный круг';

CREATE TABLE IF NOT EXISTS masters (
    id   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL
);

COMMENT ON TABLE masters IS 'Справочник мастеров (read-only проекция, FR-9)';

-- ============================================================================
-- 2. Слоты (read-only проекция из существующей инфраструктуры)
-- ============================================================================

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

    -- FR-9, FR-9a: слот содержит метаданные занятия
    -- FR-13, FR-14: ограничения мест и прокатного фонда
    CONSTRAINT chk_slot_status CHECK (status IN ('active', 'filled', 'cancelled')),
    CONSTRAINT chk_slot_total_seats_positive CHECK (total_seats > 0),
    CONSTRAINT chk_slot_free_seats_not_negative CHECK (free_seats >= 0),
    CONSTRAINT chk_slot_free_seats_lte_total CHECK (free_seats <= total_seats),
    CONSTRAINT chk_slot_free_rental_not_negative CHECK (free_rental_kits >= 0),
    CONSTRAINT chk_slot_price_not_negative CHECK (price >= 0),
    CONSTRAINT chk_slot_rental_price_not_negative CHECK (rental_price >= 0),

    -- R-003, FR-13: всего мест в слоте не может превышать потолок программы
    -- NOTE: cross-table constraint — требует триггер или логику приложения (см. ниже)
    CONSTRAINT chk_slot_total_seats_lte_program_cap CHECK (true) -- placeholder для документации
);

COMMENT ON TABLE slots IS 'Слоты занятий (read-only проекция, FR-9, FR-9a, FR-44)';
COMMENT ON COLUMN slots.free_seats IS 'Свободных мест (вычисляется: total_seats − Σ active/late_cancel bookings)';
COMMENT ON COLUMN slots.free_rental_kits IS 'Свободных прокатных наборов (FR-14)';
COMMENT ON COLUMN slots.status IS 'active — запись открыта, filled — мест нет, cancelled — отменён мастерской (FR-46)';

CREATE INDEX idx_slots_start_at ON slots(start_at);
CREATE INDEX idx_slots_status ON slots(status) WHERE status = 'active';
CREATE INDEX idx_slots_program_master ON slots(program_id, master_id);

-- ============================================================================
-- 3. Триггер: total_seats в слоте не превышает capacity_cap программы
-- ============================================================================

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

-- ============================================================================
-- 4. Клиенты (управляются клиентским API / BFF)
-- ============================================================================

CREATE TABLE IF NOT EXISTS clients (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        VARCHAR(255) NOT NULL,
    phone       VARCHAR(20) NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- FR-1, FR-2: регистрация по имени и телефону
    -- NFR-11: защита персональных данных (phone — sensitive, unique index)
    CONSTRAINT chk_client_phone_format CHECK (phone ~ '^\+[1-9]\d{7,14}$'),
    CONSTRAINT uq_client_phone UNIQUE (phone)
);

COMMENT ON TABLE clients IS 'Клиенты мастерской (FR-1, FR-2, FR-43, FR-47)';
COMMENT ON COLUMN clients.phone IS 'Телефон в формате E.164 — логин (NFR-11)';

CREATE INDEX idx_clients_phone ON clients(phone);

-- ============================================================================
-- 5. Бронирования (Booking)
-- ============================================================================

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

    -- FR-10, FR-11, FR-12: запись на занятие (1–3 места, выбор инструментов)
    -- FR-13: макс. мест на бронь ≤ 3 (себя + до 2 гостей)
    -- FR-14: rental_count не превышает свободный прокатный фонд
    -- FR-15: запрет овербукинга (проверяется на уровне приложения + БД)
    -- FR-17/FR-18: ранняя / поздняя отмена (логика в приложении; cancelled_at фиксирует время)
    -- FR-46: отмена мастерской с причиной
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

    -- FR-46: cancellation_reason обязателен только при workshop_cancelled
    CONSTRAINT chk_booking_cancel_reason CHECK (
        (status = 'workshop_cancelled' AND cancellation_reason IS NOT NULL AND cancellation_reason <> '') OR
        (status != 'workshop_cancelled')
    )
);

COMMENT ON TABLE bookings IS 'Записи клиентов на занятия (FR-10…FR-18, FR-35a, FR-46)';
COMMENT ON COLUMN bookings.seats_count IS 'Общее число мест в брони: 1–3 (FR-12, FR-13)';
COMMENT ON COLUMN bookings.rental_count IS 'Число прокатных наборов (FR-14); 0 = все со своими инструментами';
COMMENT ON COLUMN bookings.price_total IS 'Итоговая стоимость от сервера (FR-45): price×seats + rental_price×rental_count';
COMMENT ON COLUMN bookings.status IS 'active/cancelled/late_cancel/workshop_cancelled (FR-17, FR-18, FR-46)';
COMMENT ON COLUMN bookings.cancellation_reason IS 'Причина отмены мастерской (FR-46)';

CREATE INDEX idx_bookings_client ON bookings(client_id, created_at DESC);
CREATE INDEX idx_bookings_slot ON bookings(slot_id);
CREATE INDEX idx_bookings_status ON bookings(status) WHERE status = 'active';

-- ============================================================================
-- 6. Триггеры для поддержания денормализованных полей слотов (free_seats, free_rental_kits)
-- 
-- NOTE: В архитектуре BFF слоты — read-only mirror из бэкенда. 
-- Триггеры ниже применимы только если БД является primary storage 
-- (тестовое окружение / локальная разработка).
-- ============================================================================

CREATE OR REPLACE FUNCTION fn_update_slot_free_seats_on_booking()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'INSERT' AND NEW.status = 'active' THEN
        UPDATE slots SET
            free_seats = free_seats - NEW.seats_count,
            free_rental_kits = free_rental_kits - NEW.rental_count
        WHERE id = NEW.slot_id;
    ELSIF TG_OP = 'UPDATE' AND OLD.status = 'active' AND NEW.status IN ('cancelled', 'late_cancel', 'workshop_cancelled') THEN
        -- FR-17: ранняя отмена (cancelled) возвращает места и наборы
        -- FR-18: поздняя отмена (late_cancel) — места НЕ возвращаются
        -- FR-46: отмена мастерской (workshop_cancelled) — места НЕ возвращаются
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

-- ============================================================================
-- 7. OTP / Сессии (управляются BFF, хранятся в Redis по умолчанию; 
--    PostgreSQL — fallback для аудита и durability)
-- ============================================================================

CREATE TABLE IF NOT EXISTS otp_codes (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    phone       VARCHAR(20) NOT NULL,
    code        VARCHAR(10) NOT NULL,
    attempts    INT NOT NULL DEFAULT 0,
    max_attempts INT NOT NULL DEFAULT 3,
    verified    BOOLEAN NOT NULL DEFAULT FALSE,
    expires_at  TIMESTAMPTZ NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- NFR-11: OTP TTL, лимиты попыток
    CONSTRAINT chk_otp_attempts_not_negative CHECK (attempts >= 0),
    CONSTRAINT chk_otp_max_attempts_positive CHECK (max_attempts > 0)
);

COMMENT ON TABLE otp_codes IS 'OTP-коды для авторизации (FR-43, NFR-11)';
COMMENT ON COLUMN otp_codes.attempts IS 'Число попыток ввода (NFR-11)';
COMMENT ON COLUMN otp_codes.expires_at IS 'TTL кода (NFR-11)';

CREATE INDEX idx_otp_phone ON otp_codes(phone, created_at DESC);
CREATE INDEX idx_otp_expires ON otp_codes(expires_at) WHERE verified = FALSE;

-- ============================================================================
-- 8. Push-токены (FR-33: push-уведомления)
-- ============================================================================

CREATE TABLE IF NOT EXISTS push_tokens (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    client_id   UUID NOT NULL REFERENCES clients(id) ON DELETE CASCADE,
    token       VARCHAR(500) NOT NULL,
    platform    VARCHAR(20) NOT NULL DEFAULT 'unknown',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_push_platform CHECK (platform IN ('ios', 'android', 'unknown')),
    CONSTRAINT uq_push_token_per_client UNIQUE (client_id, token)
);

COMMENT ON TABLE push_tokens IS 'Push-токены устройств клиентов (FR-33)';

-- ============================================================================
-- 9. Views: удобные проекции для запросов API
-- ============================================================================

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

COMMENT ON VIEW v_slots_with_program IS 'Агрегированное представление слота для API /slots (FR-9, FR-9a)';

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

COMMENT ON VIEW v_bookings_full IS 'Агрегированное представление брони для API /bookings (FR-35a, FR-16)';

COMMIT;