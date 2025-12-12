package services

import (
	"database/sql"

	"github.com/pos/notification-service/src/repository"
)

// NotificationConfigService handles notification configuration business logic
type NotificationConfigService struct {
	configRepo *repository.NotificationConfigRepository
}

// NewNotificationConfigService creates a new notification config service
func NewNotificationConfigService(db *sql.DB) *NotificationConfigService {
	return &NotificationConfigService{
		configRepo: repository.NewNotificationConfigRepository(db),
	}
}

// GetNotificationConfig retrieves notification configuration for a tenant
func (s *NotificationConfigService) GetNotificationConfig(tenantID string) (map[string]interface{}, error) {
	return s.configRepo.GetNotificationConfig(tenantID)
}

// UpdateNotificationConfig updates notification configuration for a tenant
func (s *NotificationConfigService) UpdateNotificationConfig(tenantID string, config map[string]interface{}) error {
	return s.configRepo.UpdateNotificationConfig(tenantID, config)
}
