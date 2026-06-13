package services

import (
	"regexp"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
)

func expectTenantContext(mock sqlmock.Sqlmock, tenantID uuid.UUID) {
	mock.ExpectExec(regexp.QuoteMeta(`SELECT set_config('app.current_tenant_id', $1, true)`)).
		WithArgs(tenantID.String()).
		WillReturnResult(sqlmock.NewResult(0, 1))
}
