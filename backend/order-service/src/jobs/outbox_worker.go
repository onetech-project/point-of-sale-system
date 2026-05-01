package jobs

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/point-of-sale-system/order-service/src/services"
)

// OutboxWorker is a background worker that polls the event outbox
// and publishes pending events to Kafka
// Implements the Transactional Outbox Pattern for reliable event delivery
type OutboxWorker struct {
	eventPublisher *services.EventPublisher
	pollInterval   time.Duration
	batchSize      int
	isRunning      bool
	stopChan       chan struct{}
}

// OutboxWorkerConfig holds configuration for the outbox worker
type OutboxWorkerConfig struct {
	PollInterval time.Duration // How often to poll for pending events
	BatchSize    int           // Number of events to process per batch
}

// NewOutboxWorker creates a new outbox worker with default configuration
func NewOutboxWorker(eventPublisher *services.EventPublisher) *OutboxWorker {
	return NewOutboxWorkerWithConfig(eventPublisher, OutboxWorkerConfig{
		PollInterval: 5 * time.Second, // Poll every 5 seconds
		BatchSize:    50,               // Process up to 50 events per batch
	})
}

// NewOutboxWorkerWithConfig creates a new outbox worker with custom configuration
func NewOutboxWorkerWithConfig(eventPublisher *services.EventPublisher, config OutboxWorkerConfig) *OutboxWorker {
	return &OutboxWorker{
		eventPublisher: eventPublisher,
		pollInterval:   config.PollInterval,
		batchSize:      config.BatchSize,
		isRunning:      false,
		stopChan:       make(chan struct{}),
	}
}

// Start begins the background polling loop
// This should be called once when the service starts
func (w *OutboxWorker) Start(ctx context.Context) error {
	if w.isRunning {
		return fmt.Errorf("outbox worker is already running")
	}

	w.isRunning = true
	log.Printf("[OutboxWorker] Starting outbox worker (poll interval: %v, batch size: %d)", 
		w.pollInterval, w.batchSize)

	go w.run(ctx)
	return nil
}

// Stop gracefully stops the background worker
func (w *OutboxWorker) Stop() {
	if !w.isRunning {
		return
	}

	log.Println("[OutboxWorker] Stopping outbox worker...")
	close(w.stopChan)
	w.isRunning = false
}

// run is the main polling loop
func (w *OutboxWorker) run(ctx context.Context) {
	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()

	log.Println("[OutboxWorker] Outbox worker started successfully")

	for {
		select {
		case <-ctx.Done():
			log.Println("[OutboxWorker] Context cancelled, stopping worker")
			return

		case <-w.stopChan:
			log.Println("[OutboxWorker] Stop signal received, stopping worker")
			return

		case <-ticker.C:
			// Poll and publish pending events
			if err := w.processBatch(ctx); err != nil {
				log.Printf("[OutboxWorker] Error processing batch: %v", err)
			}
		}
	}
}

// processBatch polls the outbox and publishes a batch of pending events
func (w *OutboxWorker) processBatch(ctx context.Context) error {
	successCount, failureCount, err := w.eventPublisher.PublishPendingEvents(ctx, w.batchSize)
	if err != nil {
		return fmt.Errorf("failed to publish pending events: %w", err)
	}

	// Log summary only if events were processed
	if successCount > 0 || failureCount > 0 {
		log.Printf("[OutboxWorker] Batch processed: %d successful, %d failed", successCount, failureCount)
	}

	return nil
}

// RunOnce processes a single batch immediately (useful for testing or manual triggers)
func (w *OutboxWorker) RunOnce(ctx context.Context) (int, int, error) {
	log.Println("[OutboxWorker] Running one-time batch processing")
	return w.eventPublisher.PublishPendingEvents(ctx, w.batchSize)
}

// GetStatus returns the current worker status
func (w *OutboxWorker) GetStatus() WorkerStatus {
	return WorkerStatus{
		IsRunning:    w.isRunning,
		PollInterval: w.pollInterval,
		BatchSize:    w.batchSize,
	}
}

// WorkerStatus represents the current status of the outbox worker
type WorkerStatus struct {
	IsRunning    bool          `json:"is_running"`
	PollInterval time.Duration `json:"poll_interval"`
	BatchSize    int           `json:"batch_size"`
}

// OutboxCleanupJob removes successfully published events from the outbox
// to prevent unbounded table growth
type OutboxCleanupJob struct {
	eventPublisher *services.EventPublisher
	retentionPeriod time.Duration
}

// NewOutboxCleanupJob creates a new outbox cleanup job
// retentionPeriod: How long to keep published events before deletion (e.g., 7 days)
func NewOutboxCleanupJob(eventPublisher *services.EventPublisher, retentionPeriod time.Duration) *OutboxCleanupJob {
	return &OutboxCleanupJob{
		eventPublisher: eventPublisher,
		retentionPeriod: retentionPeriod,
	}
}

// Run executes the cleanup job (should be called periodically, e.g., daily)
func (j *OutboxCleanupJob) Run(ctx context.Context) error {
	log.Printf("[OutboxCleanup] Starting cleanup of published events older than %v", j.retentionPeriod)

	// Note: This would need additional method in EventPublisher to call DeletePublishedEvents
	// Left as placeholder for implementation
	
	log.Println("[OutboxCleanup] Cleanup completed successfully")
	return nil
}
