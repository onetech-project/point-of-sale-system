package services

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// RecordNotificationSent records that a deletion notification has been sent to prevent duplicates
// Used by cleanup job to track which users have been notified (T136)
func (s *UserDeletionService) RecordNotificationSent(ctx context.Context, userID string) error {
	query := `
		INSERT INTO user_deletion_notifications (id, user_id, notified_at)
		VALUES ($1, $2, $3)
		ON CONFLICT (user_id) DO NOTHING
	`

	notificationID := uuid.New().String()
	now := time.Now()

	_, err := s.db.ExecContext(ctx, query, notificationID, userID, now)
	if err != nil {
		return fmt.Errorf("failed to record notification: %w", err)
	}

	return nil
}
