package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"pottery-api/internal/domain"
	"pottery-api/internal/repository"
)

type ClientRepo struct {
	db *sql.DB
}

func NewClientRepo(db *sql.DB) *ClientRepo {
	return &ClientRepo{db: db}
}

func init() {
	var _ repository.ClientStore = (*ClientRepo)(nil)
}

func (r *ClientRepo) GetByPhone(ctx context.Context, phone string) (*domain.Client, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, name, phone, created_at FROM clients WHERE phone = $1`, phone)
	var c domain.Client
	if err := row.Scan(&c.ID, &c.Name, &c.Phone, &c.CreatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return &c, nil
}

func (r *ClientRepo) GetByID(ctx context.Context, id string) (*domain.Client, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, name, phone, created_at FROM clients WHERE id = $1`, id)
	var c domain.Client
	if err := row.Scan(&c.ID, &c.Name, &c.Phone, &c.CreatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return &c, nil
}

func (r *ClientRepo) Create(ctx context.Context, name, phone string) (*domain.Client, error) {
	id := uuid.New().String()
	now := time.Now()
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO clients (id, name, phone, created_at) VALUES ($1, $2, $3, $4)`,
		id, name, phone, now)
	if err != nil {
		return nil, fmt.Errorf("insert client: %w", err)
	}
	return &domain.Client{ID: id, Name: name, Phone: phone, CreatedAt: now}, nil
}

func (r *ClientRepo) Update(ctx context.Context, id, name string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE clients SET name = $1 WHERE id = $2`, name, id)
	if err != nil {
		return err
	}
	return nil
}

func (r *ClientRepo) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM clients WHERE id = $1`, id)
	return err
}