package repository

import (
	"context"
	"database/sql"
	"fmt"
)

type BaseRepository struct {
	db *sql.DB
}

func NewBaseRepository(db *sql.DB) *BaseRepository {
	return &BaseRepository{db: db}
}

func (r *BaseRepository) DB() *sql.DB {
	return r.db
}

func (r *BaseRepository) QueryRowTenantScoped(ctx context.Context, tenantID, query string, args ...interface{}) *sql.Row {
	if tenantID == "" {
		panic("tenant ID is required for tenant-scoped queries")
	}
	scopedQuery := injectTenantFilter(query, tenantID)
	return r.db.QueryRowContext(ctx, scopedQuery, args...)
}

func (r *BaseRepository) QueryTenantScoped(ctx context.Context, tenantID, query string, args ...interface{}) (*sql.Rows, error) {
	if tenantID == "" {
		return nil, fmt.Errorf("tenant ID is required for tenant-scoped queries")
	}
	scopedQuery := injectTenantFilter(query, tenantID)
	return r.db.QueryContext(ctx, scopedQuery, args...)
}

func (r *BaseRepository) ExecTenantScoped(ctx context.Context, tenantID, query string, args ...interface{}) (sql.Result, error) {
	if tenantID == "" {
		return nil, fmt.Errorf("tenant ID is required for tenant-scoped queries")
	}
	scopedQuery := injectTenantFilter(query, tenantID)
	return r.db.ExecContext(ctx, scopedQuery, args...)
}

func injectTenantFilter(query, tenantID string) string {
	if tenantID == "" {
		panic("tenant ID cannot be empty for tenant-scoped operations")
	}
	// Set the session variable for Row-Level Security
	return fmt.Sprintf("SET app.current_tenant_id = '%s'; %s", tenantID, query)
}

func (r *BaseRepository) BeginTx(ctx context.Context) (*sql.Tx, error) {
	return r.db.BeginTx(ctx, nil)
}

func (r *BaseRepository) WithTransaction(ctx context.Context, fn func(*sql.Tx) error) error {
	tx, err := r.BeginTx(ctx)
	if err != nil {
		return err
	}

	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		}
	}()

	if err := fn(tx); err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}
