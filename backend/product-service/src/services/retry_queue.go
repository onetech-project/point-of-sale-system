package services

import (
	"context"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

// RetryOperation represents an operation that can be retried
type RetryOperation struct {
	ID            string
	TenantID      string
	StorageKey    string
	Attempt       int
	MaxAttempts   int
	NextRetryTime time.Time
	CreatedAt     time.Time
	LastError     string
}

// RetryQueue manages background retry operations for failed S3 deletions
type RetryQueue struct {
	operations    map[string]*RetryOperation
	mu            sync.RWMutex
	storageClient *StorageService
	ticker        *time.Ticker
	stopChan      chan struct{}
	wg            sync.WaitGroup
}

// NewRetryQueue creates a new retry queue
func NewRetryQueue(storageClient *StorageService, checkInterval time.Duration) *RetryQueue {
	return &RetryQueue{
		operations:    make(map[string]*RetryOperation),
		storageClient: storageClient,
		ticker:        time.NewTicker(checkInterval),
		stopChan:      make(chan struct{}),
	}
}

// Start begins processing the retry queue
func (q *RetryQueue) Start(ctx context.Context) {
	q.wg.Add(1)
	go q.processQueue(ctx)
	log.Info().Msg("Retry queue started for S3 deletion operations")
}

// Stop gracefully shuts down the retry queue
func (q *RetryQueue) Stop() {
	close(q.stopChan)
	q.ticker.Stop()
	q.wg.Wait()
	log.Info().Msg("Retry queue stopped")
}

// Enqueue adds a failed deletion to the retry queue
func (q *RetryQueue) Enqueue(tenantID, storageKey string, maxAttempts int) {
	q.mu.Lock()
	defer q.mu.Unlock()

	// Check if operation already exists
	if existing, found := q.operations[storageKey]; found {
		log.Debug().
			Str("storage_key", storageKey).
			Int("current_attempt", existing.Attempt).
			Msg("S3 deletion already in retry queue")
		return
	}

	operation := &RetryOperation{
		ID:            storageKey, // Use storage key as unique ID
		TenantID:      tenantID,
		StorageKey:    storageKey,
		Attempt:       0,
		MaxAttempts:   maxAttempts,
		NextRetryTime: time.Now().Add(calculateBackoff(0)),
		CreatedAt:     time.Now(),
	}

	q.operations[storageKey] = operation

	log.Info().
		Str("tenant_id", tenantID).
		Str("storage_key", storageKey).
		Int("max_attempts", maxAttempts).
		Time("next_retry", operation.NextRetryTime).
		Msg("S3 deletion enqueued for retry")
}

// processQueue continuously processes retry operations
func (q *RetryQueue) processQueue(ctx context.Context) {
	defer q.wg.Done()

	for {
		select {
		case <-ctx.Done():
			log.Info().Msg("Retry queue context cancelled")
			return
		case <-q.stopChan:
			log.Info().Msg("Retry queue stop signal received")
			return
		case <-q.ticker.C:
			q.processPendingRetries(ctx)
		}
	}
}

// processPendingRetries processes operations ready for retry
func (q *RetryQueue) processPendingRetries(ctx context.Context) {
	q.mu.Lock()
	defer q.mu.Unlock()

	now := time.Now()
	for key, op := range q.operations {
		// Check if operation is ready for retry
		if now.Before(op.NextRetryTime) {
			continue
		}

		// Increment attempt counter
		op.Attempt++

		// Try to delete from S3
		err := q.storageClient.DeletePhoto(ctx, op.StorageKey)
		if err != nil {
			op.LastError = err.Error()

			// Check if we've exceeded max attempts
			if op.Attempt >= op.MaxAttempts {
				log.Error().
					Err(err).
					Str("tenant_id", op.TenantID).
					Str("storage_key", op.StorageKey).
					Int("attempts", op.Attempt).
					Dur("total_duration", time.Since(op.CreatedAt)).
					Msg("S3 deletion permanently failed after max retry attempts")

				// Remove from queue - give up
				delete(q.operations, key)
				continue
			}

			// Schedule next retry with exponential backoff
			op.NextRetryTime = now.Add(calculateBackoff(op.Attempt))

			log.Warn().
				Err(err).
				Str("tenant_id", op.TenantID).
				Str("storage_key", op.StorageKey).
				Int("attempt", op.Attempt).
				Int("max_attempts", op.MaxAttempts).
				Time("next_retry", op.NextRetryTime).
				Msg("S3 deletion retry failed, will retry again")
		} else {
			// Success - remove from queue
			log.Info().
				Str("tenant_id", op.TenantID).
				Str("storage_key", op.StorageKey).
				Int("attempts", op.Attempt).
				Dur("total_duration", time.Since(op.CreatedAt)).
				Msg("S3 deletion succeeded after retry")

			delete(q.operations, key)
		}
	}
}

// GetQueueStats returns statistics about the retry queue
func (q *RetryQueue) GetQueueStats() map[string]interface{} {
	q.mu.RLock()
	defer q.mu.RUnlock()

	stats := map[string]interface{}{
		"total_operations": len(q.operations),
		"operations":       []map[string]interface{}{},
	}

	for _, op := range q.operations {
		stats["operations"] = append(stats["operations"].([]map[string]interface{}), map[string]interface{}{
			"tenant_id":    op.TenantID,
			"storage_key":  op.StorageKey,
			"attempt":      op.Attempt,
			"max_attempts": op.MaxAttempts,
			"next_retry":   op.NextRetryTime,
			"created_at":   op.CreatedAt,
			"last_error":   op.LastError,
			"age_seconds":  time.Since(op.CreatedAt).Seconds(),
		})
	}

	return stats
}

// calculateBackoff returns exponential backoff duration
// Attempt 0: 30s, 1: 2m, 2: 8m, 3: 32m, 4+: 2h
func calculateBackoff(attempt int) time.Duration {
	switch attempt {
	case 0:
		return 30 * time.Second
	case 1:
		return 2 * time.Minute
	case 2:
		return 8 * time.Minute
	case 3:
		return 32 * time.Minute
	default:
		return 2 * time.Hour
	}
}
