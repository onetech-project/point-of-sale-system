package services

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"
)

// PartitionService manages monthly partitions for audit_events table
type PartitionService struct {
	db *sql.DB
}

// NewPartitionService creates a new partition service
func NewPartitionService(db *sql.DB) *PartitionService {
	return &PartitionService{db: db}
}

// StartMonitor starts monitoring and creating monthly partitions
// Runs daily and creates next month's partition 7 days before month end (T115)
func (s *PartitionService) StartMonitor(ctx context.Context) {
	log.Info().Msg("Partition manager started - checks daily, creates partitions 7 days before month end")

	// Create initial partitions on startup
	if err := s.EnsurePartitions(ctx); err != nil {
		log.Error().Err(err).Msg("Failed to create initial partitions")
	}

	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Info().Msg("Partition manager stopped")
			return
		case <-ticker.C:
			if err := s.EnsurePartitions(ctx); err != nil {
				log.Error().Err(err).Msg("Failed to ensure partitions")
			}
		}
	}
}

// EnsurePartitions creates partitions proactively (T115: 7 days before month end)
// Creates partitions for current month, next month, and month after
// This ensures partitions exist well before they're needed
func (s *PartitionService) EnsurePartitions(ctx context.Context) error {
	now := time.Now().UTC()

	// T115: Check if we're within 7 days of month end to create next month's partition
	daysInMonth := time.Date(now.Year(), now.Month()+1, 0, 0, 0, 0, 0, time.UTC).Day()
	daysUntilMonthEnd := daysInMonth - now.Day()

	if daysUntilMonthEnd <= 7 {
		log.Info().
			Int("days_until_month_end", daysUntilMonthEnd).
			Msg("Within 7 days of month end - ensuring next month partition exists")
	}

	// Create partitions for current month, next month, and month after
	// This ensures we're always prepared 2 months ahead
	for i := 0; i < 3; i++ {
		targetMonth := now.AddDate(0, i, 0)
		partitionName := fmt.Sprintf("audit_events_%s", targetMonth.Format("2006_01"))

		exists, err := s.PartitionExists(ctx, partitionName)
		if err != nil {
			return fmt.Errorf("failed to check partition existence: %w", err)
		}

		if !exists {
			if err := s.CreatePartition(ctx, targetMonth); err != nil {
				return fmt.Errorf("failed to create partition %s: %w", partitionName, err)
			}
			log.Info().Str("partition", partitionName).Msg("Created monthly partition")
		} else {
			log.Debug().Str("partition", partitionName).Msg("Partition already exists")
		}
	}

	return nil
}

// PartitionExists checks if a partition table exists
func (s *PartitionService) PartitionExists(ctx context.Context, partitionName string) (bool, error) {
	query := `
		SELECT EXISTS (
			SELECT 1
			FROM pg_class c
			JOIN pg_namespace n ON n.oid = c.relnamespace
			WHERE c.relname = $1
			AND n.nspname = 'public'
		)
	`

	var exists bool
	err := s.db.QueryRowContext(ctx, query, partitionName).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to query partition existence: %w", err)
	}

	return exists, nil
}

// CreatePartition creates a new monthly partition for audit_events
// Partition range: [start_of_month, start_of_next_month)
func (s *PartitionService) CreatePartition(ctx context.Context, month time.Time) error {
	// Normalize to start of month
	startOfMonth := time.Date(month.Year(), month.Month(), 1, 0, 0, 0, 0, time.UTC)
	startOfNextMonth := startOfMonth.AddDate(0, 1, 0)

	partitionName := fmt.Sprintf("audit_events_%s", month.Format("2006_01"))

	// Create partition table for this month's range
	query := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s PARTITION OF audit_events
		FOR VALUES FROM ('%s') TO ('%s')
	`, partitionName, startOfMonth.Format(time.RFC3339), startOfNextMonth.Format(time.RFC3339))

	_, err := s.db.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to create partition table: %w", err)
	}

	// Create indexes on the partition for better query performance
	indexes := []string{
		fmt.Sprintf("CREATE INDEX IF NOT EXISTS %s_tenant_id_idx ON %s (tenant_id)", partitionName, partitionName),
		fmt.Sprintf("CREATE INDEX IF NOT EXISTS %s_timestamp_idx ON %s (timestamp DESC)", partitionName, partitionName),
		fmt.Sprintf("CREATE INDEX IF NOT EXISTS %s_resource_idx ON %s (resource_type, resource_id)", partitionName, partitionName),
		fmt.Sprintf("CREATE INDEX IF NOT EXISTS %s_actor_idx ON %s (actor_type, actor_id)", partitionName, partitionName),
	}

	for _, indexQuery := range indexes {
		if _, err := s.db.ExecContext(ctx, indexQuery); err != nil {
			log.Warn().Err(err).Str("partition", partitionName).Msg("Failed to create index on partition")
		}
	}

	return nil
}

// DropOldPartitions removes partitions older than retention period (e.g., 7 years per UU PDP Article 56)
// This should be called periodically (e.g., monthly) as part of data retention policy
func (s *PartitionService) DropOldPartitions(ctx context.Context, retentionMonths int) error {
	cutoffDate := time.Now().UTC().AddDate(0, -retentionMonths, 0)
	cutoffPartition := fmt.Sprintf("audit_events_%s", cutoffDate.Format("2006_01"))

	// Find all audit_events partitions older than cutoff
	query := `
		SELECT c.relname
		FROM pg_class c
		JOIN pg_namespace n ON n.oid = c.relnamespace
		WHERE c.relname LIKE 'audit_events_%'
		AND n.nspname = 'public'
		AND c.relname < $1
		ORDER BY c.relname
	`

	rows, err := s.db.QueryContext(ctx, query, cutoffPartition)
	if err != nil {
		return fmt.Errorf("failed to query old partitions: %w", err)
	}
	defer rows.Close()

	var droppedCount int
	for rows.Next() {
		var partitionName string
		if err := rows.Scan(&partitionName); err != nil {
			log.Error().Err(err).Msg("Failed to scan partition name")
			continue
		}

		// Drop the partition table
		dropQuery := fmt.Sprintf("DROP TABLE IF EXISTS %s", partitionName)
		if _, err := s.db.ExecContext(ctx, dropQuery); err != nil {
			log.Error().Err(err).Str("partition", partitionName).Msg("Failed to drop partition")
			continue
		}

		log.Info().Str("partition", partitionName).Msg("Dropped old partition")
		droppedCount++
	}

	if droppedCount > 0 {
		log.Info().Int("count", droppedCount).Msg("Dropped old partitions")
	}

	return rows.Err()
}
