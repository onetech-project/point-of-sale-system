package unit

import (
"context"
"database/sql"
"testing"
"time"

"github.com/DATA-DOG/go-sqlmock"
"github.com/google/uuid"
"github.com/pos/backend/product-service/src/models"
"github.com/pos/backend/product-service/src/repository"
"github.com/stretchr/testify/assert"
"github.com/stretchr/testify/require"
)

// T072: Unit test for StockRepository.CreateAdjustment
func TestStockRepositoryCreateAdjustment(t *testing.T) {
db, mock, err := sqlmock.New()
require.NoError(t, err)
defer db.Close()

repo := repository.NewStockRepository(db)
ctx := context.Background()

tests := []struct {
name       string
adjustment *models.StockAdjustment
mockSetup  func(sqlmock.Sqlmock, *models.StockAdjustment)
wantErr    bool
}{
{
name: "successful adjustment creation",
adjustment: &models.StockAdjustment{
TenantID:         uuid.New(),
ProductID:        uuid.New(),
UserID:           uuid.New(),
PreviousQuantity: 100,
NewQuantity:      150,
QuantityDelta:    50,
Reason:           "supplier_delivery",
Notes:            "Received shipment",
},
mockSetup: func(mock sqlmock.Sqlmock, adj *models.StockAdjustment) {
rows := sqlmock.NewRows([]string{"id", "created_at"}).
AddRow(uuid.New(), time.Now())

mock.ExpectQuery(`INSERT INTO stock_adjustments`).
WithArgs(adj.TenantID, adj.ProductID, adj.UserID,
adj.PreviousQuantity, adj.NewQuantity, adj.QuantityDelta,
adj.Reason, adj.Notes).
WillReturnRows(rows)
},
wantErr: false,
},
{
name: "adjustment with negative delta",
adjustment: &models.StockAdjustment{
TenantID:         uuid.New(),
ProductID:        uuid.New(),
UserID:           uuid.New(),
PreviousQuantity: 100,
NewQuantity:      80,
QuantityDelta:    -20,
Reason:           "shrinkage",
Notes:            "Damaged items",
},
mockSetup: func(mock sqlmock.Sqlmock, adj *models.StockAdjustment) {
rows := sqlmock.NewRows([]string{"id", "created_at"}).
AddRow(uuid.New(), time.Now())

mock.ExpectQuery(`INSERT INTO stock_adjustments`).
WithArgs(adj.TenantID, adj.ProductID, adj.UserID,
adj.PreviousQuantity, adj.NewQuantity, adj.QuantityDelta,
adj.Reason, adj.Notes).
WillReturnRows(rows)
},
wantErr: false,
},
{
name: "database error",
adjustment: &models.StockAdjustment{
TenantID:         uuid.New(),
ProductID:        uuid.New(),
UserID:           uuid.New(),
PreviousQuantity: 100,
NewQuantity:      150,
Reason:           "correction",
},
mockSetup: func(mock sqlmock.Sqlmock, adj *models.StockAdjustment) {
mock.ExpectQuery(`INSERT INTO stock_adjustments`).
WillReturnError(sql.ErrConnDone)
},
wantErr: true,
},
}

for _, tt := range tests {
t.Run(tt.name, func(t *testing.T) {
tt.mockSetup(mock, tt.adjustment)

err := repo.CreateAdjustment(ctx, tt.adjustment)

if tt.wantErr {
assert.Error(t, err)
} else {
assert.NoError(t, err)
assert.NotEqual(t, uuid.Nil, tt.adjustment.ID)
}

assert.NoError(t, mock.ExpectationsWereMet())
})
}
}
