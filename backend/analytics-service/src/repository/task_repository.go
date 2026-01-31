package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/pos/analytics-service/src/models"
	"github.com/pos/analytics-service/src/utils"
)

// TaskRepository handles operational task queries (delayed orders, low stock)
type TaskRepository struct {
	db          *sql.DB
	vaultClient *utils.VaultClient
}

// NewTaskRepository creates a new task repository instance
func NewTaskRepository(db *sql.DB, vaultClient *utils.VaultClient) *TaskRepository {
	return &TaskRepository{
		db:          db,
		vaultClient: vaultClient,
	}
}

// GetDelayedOrders retrieves orders that have been pending for more than 15 minutes
// Returns orders with decrypted and masked customer PII
func (r *TaskRepository) GetDelayedOrders(ctx context.Context, tenantID string) ([]models.DelayedOrder, error) {
	query := `
		SELECT 
			o.id AS order_id,
			o.order_reference,
			o.customer_phone,
			o.customer_name,
			o.customer_email,
			o.total_amount,
			o.status,
			o.created_at,
			EXTRACT(EPOCH FROM (NOW() - o.created_at)) / 60 AS elapsed_minutes
		FROM guest_orders o
		WHERE o.tenant_id = $1
		  AND o.status = 'PAID'
		  AND o.created_at < NOW() - INTERVAL '15 minutes'
		ORDER BY o.created_at ASC
		LIMIT 50
	`

	rows, err := r.db.QueryContext(ctx, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to query delayed orders: %w", err)
	}
	defer rows.Close()

	var orders []models.DelayedOrder
	encryptedPhones := make(map[uuid.UUID]string)
	encryptedNames := make(map[uuid.UUID]string)
	encryptedEmails := make(map[uuid.UUID]string)

	for rows.Next() {
		var order models.DelayedOrder
		var elapsedMinutes float64

		err := rows.Scan(
			&order.OrderID,
			&order.OrderNumber,
			&order.CustomerID,
			&order.CustomerPhone,
			&order.CustomerName,
			&order.CustomerEmail,
			&order.TotalAmount,
			&order.Status,
			&order.CreatedAt,
			&elapsedMinutes,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan delayed order: %w", err)
		}

		order.ElapsedMinutes = int(elapsedMinutes)
		orders = append(orders, order)

		// Collect encrypted values for batch decryption
		encryptedPhones[order.OrderID] = order.CustomerPhone
		encryptedNames[order.OrderID] = order.CustomerName
		encryptedEmails[order.OrderID] = order.CustomerEmail
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating delayed orders: %w", err)
	}

	// Batch decrypt customer PII
	if err := r.batchDecryptAndMaskCustomerData(ctx, orders, encryptedPhones, encryptedNames, encryptedEmails); err != nil {
		return nil, fmt.Errorf("failed to decrypt customer data: %w", err)
	}

	return orders, nil
}

// GetLowStockProducts retrieves products that are at or below their low stock threshold
func (r *TaskRepository) GetLowStockProducts(ctx context.Context, tenantID string) ([]models.RestockAlert, error) {
	lowStockThreshold := 10 // This could be made configurable per product
	query := `
		SELECT 
			p.id AS product_id,
			p.name AS product_name,
			c.name AS category_name,
			p.sku,
			$2::integer AS low_stock_threshold,
			p.stock_quantity AS current_stock,
			p.selling_price,
			p.cost_price
		FROM products p
		LEFT JOIN categories c ON p.category_id = c.id
		WHERE p.tenant_id = $1
		  AND p.archived_at IS NULL
		  AND p.stock_quantity <= $2::integer
		ORDER BY 
			CASE WHEN p.stock_quantity = 0 THEN 0 ELSE 1 END,  -- Critical (0 stock) first
			p.stock_quantity ASC,
			p.name ASC
		LIMIT 100
	`

	rows, err := r.db.QueryContext(ctx, query, tenantID, lowStockThreshold)
	if err != nil {
		return nil, fmt.Errorf("failed to query low stock products: %w", err)
	}
	defer rows.Close()

	var alerts []models.RestockAlert

	for rows.Next() {
		var alert models.RestockAlert
		var categoryName sql.NullString

		err := rows.Scan(
			&alert.ProductID,
			&alert.ProductName,
			&categoryName,
			&alert.SKU,
			&alert.LowStockThreshold,
			&alert.CurrentStock,
			&alert.SellingPrice,
			&alert.CostPrice,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan restock alert: %w", err)
		}

		if categoryName.Valid {
			alert.CategoryName = categoryName.String
		}

		// Calculate status and recommended reorder
		if alert.IsCritical() {
			alert.Status = "critical"
		} else {
			alert.Status = "low"
		}
		alert.RecommendedReorder = alert.CalculateRecommendedReorder()

		alerts = append(alerts, alert)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating low stock products: %w", err)
	}

	return alerts, nil
}

// batchDecryptAndMaskCustomerData decrypts customer PII in batch and applies masking
func (r *TaskRepository) batchDecryptAndMaskCustomerData(
	ctx context.Context,
	orders []models.DelayedOrder,
	encryptedPhones, encryptedNames, encryptedEmails map[uuid.UUID]string,
) error {
	if len(orders) == 0 {
		return nil
	}

	// Batch decrypt phones
	phoneBatch := make([]string, 0, len(encryptedPhones))
	phoneIndexMap := make(map[string]uuid.UUID)
	for orderID, phone := range encryptedPhones {
		phoneBatch = append(phoneBatch, phone)
		phoneIndexMap[phone] = orderID
	}

	decryptedPhones, err := r.vaultClient.DecryptBatch(ctx, phoneBatch, []string{"guest_order:customer_phone"})
	if err != nil {
		return fmt.Errorf("failed to batch decrypt phones: %w", err)
	}

	// Batch decrypt names
	nameBatch := make([]string, 0, len(encryptedNames))
	nameIndexMap := make(map[string]uuid.UUID)
	for orderID, name := range encryptedNames {
		nameBatch = append(nameBatch, name)
		nameIndexMap[name] = orderID
	}

	decryptedNames, err := r.vaultClient.DecryptBatch(ctx, nameBatch, []string{"guest_order:customer_name"})
	if err != nil {
		return fmt.Errorf("failed to batch decrypt names: %w", err)
	}

	// Batch decrypt emails
	emailBatch := make([]string, 0, len(encryptedEmails))
	emailIndexMap := make(map[string]uuid.UUID)
	for orderID, email := range encryptedEmails {
		emailBatch = append(emailBatch, email)
		emailIndexMap[email] = orderID
	}

	decryptedEmails, err := r.vaultClient.DecryptBatch(ctx, emailBatch, []string{"guest_order:customer_email"})
	if err != nil {
		return fmt.Errorf("failed to batch decrypt emails: %w", err)
	}

	// Map decrypted values back to orders and apply masking
	phoneMap := make(map[uuid.UUID]string)
	for i, plaintext := range decryptedPhones {
		encrypted := phoneBatch[i]
		orderID := phoneIndexMap[encrypted]
		phoneMap[orderID] = plaintext
	}

	nameMap := make(map[uuid.UUID]string)
	for i, plaintext := range decryptedNames {
		encrypted := nameBatch[i]
		orderID := nameIndexMap[encrypted]
		nameMap[orderID] = plaintext
	}

	emailMap := make(map[uuid.UUID]string)
	for i, plaintext := range decryptedEmails {
		encrypted := emailBatch[i]
		orderID := emailIndexMap[encrypted]
		emailMap[orderID] = plaintext
	}

	// Apply masked values to orders
	for i := range orders {
		orderID := orders[i].OrderID

		if phone, ok := phoneMap[orderID]; ok {
			orders[i].MaskedPhone = utils.MaskPhone(phone)
		}

		if name, ok := nameMap[orderID]; ok {
			orders[i].MaskedName = utils.MaskName(name)
		}

		if email, ok := emailMap[orderID]; ok {
			orders[i].MaskedEmail = utils.MaskEmail(email)
		}
	}

	return nil
}
