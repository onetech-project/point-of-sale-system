package repository

import (
	"context"
	"database/sql"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
)

func TestWithTenantTxSetsLocalTenantContext(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer db.Close()

	tenantID := uuid.New()
	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`SELECT set_config('app.current_tenant_id', $1, true)`)).
		WithArgs(tenantID.String()).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT 1`)).
		WillReturnRows(sqlmock.NewRows([]string{"ok"}).AddRow(1))
	mock.ExpectCommit()

	err = WithTenantTx(context.Background(), db, tenantID, nil, func(tx *sql.Tx) error {
		var ok int
		return tx.QueryRowContext(context.Background(), `SELECT 1`).Scan(&ok)
	})
	if err != nil {
		t.Fatalf("WithTenantTx error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}
