package models

import (
	"time"
)

// RetentionPolicy defines automated data retention rules per table/record type
type RetentionPolicy struct {
	ID                     string    `json:"id" db:"id"`
	TableName              string    `json:"table_name" db:"table_name"`
	RecordType             *string   `json:"record_type" db:"record_type"`                             // Optional subtype
	RetentionPeriodDays    int       `json:"retention_period_days" db:"retention_period_days"`         // Days to retain
	RetentionField         string    `json:"retention_field" db:"retention_field"`                     // Timestamp field to check
	GracePeriodDays        *int      `json:"grace_period_days" db:"grace_period_days"`                 // Soft delete grace period
	LegalMinimumDays       *int      `json:"legal_minimum_days" db:"legal_minimum_days"`               // Minimum by law
	CleanupMethod          string    `json:"cleanup_method" db:"cleanup_method"`                       // 'soft_delete', 'hard_delete', 'anonymize'
	NotificationDaysBefore *int      `json:"notification_days_before" db:"notification_days_before"`   // Notification window
	IsActive               bool      `json:"is_active" db:"is_active"`
	CreatedAt              time.Time `json:"created_at" db:"created_at"`
	UpdatedAt              time.Time `json:"updated_at" db:"updated_at"`
}

// IsExpired checks if a record timestamp is beyond the retention period
func (rp *RetentionPolicy) IsExpired(recordTimestamp time.Time) bool {
	expiryDate := recordTimestamp.AddDate(0, 0, rp.RetentionPeriodDays)
	return time.Now().After(expiryDate)
}

// ShouldNotify checks if notification should be sent (within notification window)
func (rp *RetentionPolicy) ShouldNotify(recordTimestamp time.Time) bool {
	if rp.NotificationDaysBefore == nil {
		return false
	}
	
	expiryDate := recordTimestamp.AddDate(0, 0, rp.RetentionPeriodDays)
	notificationDate := expiryDate.AddDate(0, 0, -*rp.NotificationDaysBefore)
	now := time.Now()
	
	return now.After(notificationDate) && now.Before(expiryDate)
}
