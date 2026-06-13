package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
)

func SetTenantContextTx(ctx context.Context, tx *sql.Tx, tenantID uuid.UUID) error {
	if _, err := tx.ExecContext(ctx, `SELECT set_config('app.current_tenant_id', $1, true)`, tenantID.String()); err != nil {
		return fmt.Errorf("failed to set tenant context: %w", err)
	}
	return nil
}

func WithTenantTx(ctx context.Context, db *sql.DB, tenantID uuid.UUID, opts *sql.TxOptions, fn func(*sql.Tx) error) error {
	tx, err := db.BeginTx(ctx, opts)
	if err != nil {
		return fmt.Errorf("failed to begin tenant transaction: %w", err)
	}

	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	if err := SetTenantContextTx(ctx, tx, tenantID); err != nil {
		return err
	}
	if err := fn(tx); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit tenant transaction: %w", err)
	}
	committed = true
	return nil
}
