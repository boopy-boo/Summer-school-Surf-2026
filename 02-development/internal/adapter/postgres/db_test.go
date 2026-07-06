package postgres

import (
	"context"
	"testing"

	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMigrateUpAndDown(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}

	// Используем DATABASE_URL из окружения или дефолт
	dsn := "postgres://postgres:postgres@localhost:5432/pottery?sslmode=disable"
	db, err := NewDB(dsn)
	if err != nil {
		t.Skipf("postgres not available: %v", err)
	}
	defer db.Close()

	// Cleanup before test
	_ = MigrateDown(db)

	// Up
	require.NoError(t, MigrateUp(db))

	// Verify tables exist
	tables := []string{"programs", "masters", "slots", "clients", "bookings", "otp_codes", "push_tokens"}
	for _, table := range tables {
		var name string
		err := db.QueryRowContext(context.Background(),
			`SELECT tablename FROM pg_tables WHERE schemaname='public' AND tablename=$1`, table).Scan(&name)
		require.NoError(t, err, "table %s should exist", table)
		assert.Equal(t, table, name)
	}

	// Verify CHECK constraints
	var constraintName string
	err = db.QueryRow(`
		SELECT conname FROM pg_constraint 
		WHERE conname = 'chk_program_capacity_by_type'`).Scan(&constraintName)
	require.NoError(t, err)
	assert.Equal(t, "chk_program_capacity_by_type", constraintName)

	// Verify views exist
	views := []string{"v_slots_with_program", "v_bookings_full"}
	for _, view := range views {
		var name string
		err := db.QueryRowContext(context.Background(),
			`SELECT viewname FROM pg_views WHERE schemaname='public' AND viewname=$1`, view).Scan(&name)
		require.NoError(t, err, "view %s should exist", view)
	}

	// Verify triggers exist
	var triggerName string
	err = db.QueryRow(`
		SELECT tgname FROM pg_trigger 
		WHERE tgname = 'trg_booking_change'`).Scan(&triggerName)
	require.NoError(t, err)
	assert.Equal(t, "trg_booking_change", triggerName)

	// Verify data constraints
	_, err = db.Exec(`INSERT INTO programs (name, type, capacity_cap) VALUES ('Test', 'handbuilding', 8)`)
	assert.Error(t, err, "should fail: capacity_cap > 6 for handbuilding")

	_, err = db.Exec(`INSERT INTO programs (name, type, capacity_cap) VALUES ('Test', 'wheel', 12)`)
	assert.Error(t, err, "should fail: capacity_cap > 10 for wheel")

	_, err = db.Exec(`INSERT INTO programs (name, type, capacity_cap) VALUES ('Wheel OK', 'wheel', 10)`)
	require.NoError(t, err, "should succeed: capacity_cap = 10 for wheel")

	// Verify client phone E.164
	_, err = db.Exec(`INSERT INTO clients (name, phone) VALUES ('Ivan', '89161234567')`)
	assert.Error(t, err, "should fail: phone without +")

	_, err = db.Exec(`INSERT INTO clients (name, phone) VALUES ('Ivan', '+79161234567')`)
	require.NoError(t, err)

	// Verify booking seats limit
	_, err = db.Exec(`INSERT INTO bookings (slot_id, client_id, seats_count, rental_count, price_total) 
		VALUES ('00000000-0000-0000-0000-000000000001', 
		        (SELECT id FROM clients WHERE phone = '+79161234567'), 
		        4, 0, 100)`)
	assert.Error(t, err, "should fail: seats_count > 3")

	// Down
	require.NoError(t, MigrateDown(db))

	// Verify tables dropped
	for _, table := range tables {
		var count int
		err := db.QueryRowContext(context.Background(),
			`SELECT COUNT(*) FROM pg_tables WHERE schemaname='public' AND tablename=$1`, table).Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, 0, count, "table %s should be dropped", table)
	}
}