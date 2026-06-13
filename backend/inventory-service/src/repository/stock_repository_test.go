package repository

import (
	"context"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

func TestStockRepositoryValuation(t *testing.T) {
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
	mock.ExpectQuery(`SELECT COALESCE\(SUM\(current_stock_base \* average_cost_per_base_unit\), 0\)`).
		WithArgs(tenantID).
		WillReturnRows(sqlmock.NewRows([]string{"total"}).AddRow("125.50"))
	mock.ExpectCommit()

	repo := NewStockRepository(db)
	got, err := repo.Valuation(context.Background(), tenantID)
	if err != nil {
		t.Fatalf("Valuation error: %v", err)
	}
	want := decimal.RequireFromString("125.50")
	if !got.Equal(want) {
		t.Fatalf("Valuation = %s, want %s", got, want)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}
