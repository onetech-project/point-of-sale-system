package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/pos/analytics-service/src/models"
	"github.com/pos/analytics-service/src/utils"
	"github.com/rs/zerolog/log"
)

// CustomerRepository handles customer analytics queries with encryption
type CustomerRepository struct {
	db        *sql.DB
	encryptor utils.Encryptor
}

// NewCustomerRepository creates a new customer repository
func NewCustomerRepository(db *sql.DB, encryptor utils.Encryptor) *CustomerRepository {
	return &CustomerRepository{
		db:        db,
		encryptor: encryptor,
	}
}

// GetTopCustomersBySpending returns top N customers by total spending
func (r *CustomerRepository) GetTopCustomersBySpending(ctx context.Context, tenantID string, start, end time.Time, limit int) ([]models.CustomerRanking, error) {
	query := `
		SELECT 
			customer_name,
			customer_phone,
			customer_email,
			COUNT(*) as order_count,
			COALESCE(SUM(total_amount), 0) as total_spent,
			COALESCE(AVG(total_amount), 0) as average_order
		FROM guest_orders
		WHERE tenant_id = $1 
			AND status = 'COMPLETE'
			AND created_at BETWEEN $2 AND $3
		GROUP BY customer_name, customer_phone, customer_email
		ORDER BY total_spent DESC
		LIMIT $4
	`

	return r.queryCustomers(ctx, query, tenantID, start, end, limit)
}

// GetTopCustomersByOrders returns top N customers by order count
func (r *CustomerRepository) GetTopCustomersByOrders(ctx context.Context, tenantID string, start, end time.Time, limit int) ([]models.CustomerRanking, error) {
	query := `
		SELECT 
			customer_name,
			customer_phone,
			customer_email,
			COUNT(*) as order_count,
			COALESCE(SUM(total_amount), 0) as total_spent,
			COALESCE(AVG(total_amount), 0) as average_order
		FROM guest_orders
		WHERE tenant_id = $1 
			AND status = 'COMPLETE'
			AND created_at BETWEEN $2 AND $3
		GROUP BY customer_name, customer_phone, customer_email
		ORDER BY order_count DESC
		LIMIT $4
	`

	return r.queryCustomers(ctx, query, tenantID, start, end, limit)
}

// queryCustomers is a helper function to execute customer queries with decryption
func (r *CustomerRepository) queryCustomers(ctx context.Context, query string, tenantID string, start, end time.Time, limit int) ([]models.CustomerRanking, error) {
	rows, err := r.db.QueryContext(ctx, query, tenantID, start, end, limit)
	if err != nil {
		log.Error().Err(err).Str("tenant_id", tenantID).Msg("Failed to query customers")
		return nil, err
	}
	defer rows.Close()

	var customers []models.CustomerRanking
	var encryptedNames []string
	var encryptedPhones []string
	var encryptedEmails []string

	// First pass: collect encrypted data
	for rows.Next() {
		var c models.CustomerRanking
		var name, phone, email sql.NullString

		if err := rows.Scan(&name, &phone, &email, &c.OrderCount, &c.TotalSpent, &c.AverageOrder); err != nil {
			log.Error().Err(err).Msg("Failed to scan customer row")
			continue
		}

		customers = append(customers, c)

		if name.Valid {
			encryptedNames = append(encryptedNames, name.String)
		} else {
			encryptedNames = append(encryptedNames, "")
		}

		if phone.Valid {
			encryptedPhones = append(encryptedPhones, phone.String)
		} else {
			encryptedPhones = append(encryptedPhones, "")
		}

		if email.Valid {
			encryptedEmails = append(encryptedEmails, email.String)
		} else {
			encryptedEmails = append(encryptedEmails, "")
		}
	}

	// Batch decrypt customer PII
	decryptedNames, err := r.encryptor.DecryptBatch(ctx, encryptedNames, "guest_order:customer_name")
	if err != nil {
		log.Error().Err(err).Msg("Failed to decrypt customer names")
		// Continue with encrypted values
		decryptedNames = encryptedNames
	}

	decryptedPhones, err := r.encryptor.DecryptBatch(ctx, encryptedPhones, "guest_order:customer_phone")
	if err != nil {
		log.Error().Err(err).Msg("Failed to decrypt customer phones")
		decryptedPhones = encryptedPhones
	}

	decryptedEmails, err := r.encryptor.DecryptBatch(ctx, encryptedEmails, "guest_order:customer_email")
	if err != nil {
		log.Error().Err(err).Str("emails", encryptedEmails[0]).Msg("Failed to decrypt customer emails")
		decryptedEmails = encryptedEmails
	}

	// Second pass: mask decrypted data for display
	for i := range customers {
		// Mask name: show only first character
		if i < len(decryptedNames) && decryptedNames[i] != "" {
			customers[i].Name = utils.MaskName(decryptedNames[i])
		} else {
			customers[i].Name = "Unknown"
		}

		// Mask phone: show only last 4 digits
		if i < len(decryptedPhones) && decryptedPhones[i] != "" {
			customers[i].Phone = utils.MaskPhone(decryptedPhones[i])
		} else {
			customers[i].Phone = "N/A"
		}

		// Mask email: show first char + domain
		if i < len(decryptedEmails) && decryptedEmails[i] != "" {
			customers[i].Email = utils.MaskEmail(decryptedEmails[i])
		} else {
			customers[i].Email = "N/A"
		}
	}

	return customers, nil
}
