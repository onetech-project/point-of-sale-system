package services

import (
	"context"
	"time"

	"github.com/rs/zerolog/log"
)

type ReservationCleanupJob struct {
	inventoryService *InventoryService
	interval         time.Duration
	stopChan         chan struct{}
}

func NewReservationCleanupJob(inventoryService *InventoryService) *ReservationCleanupJob {
	return &ReservationCleanupJob{
		inventoryService: inventoryService,
		interval:         1 * time.Minute, // Run every minute
		stopChan:         make(chan struct{}),
	}
}

// Start begins the cleanup job in a goroutine
func (j *ReservationCleanupJob) Start(ctx context.Context) {
	log.Info().Msg("Starting reservation cleanup job")

	ticker := time.NewTicker(j.interval)
	defer ticker.Stop()

	// Run immediately on start
	j.cleanupExpiredReservations(ctx)

	for {
		select {
		case <-ticker.C:
			j.cleanupExpiredReservations(ctx)
		case <-j.stopChan:
			log.Info().Msg("Stopping reservation cleanup job")
			return
		case <-ctx.Done():
			log.Info().Msg("Context cancelled, stopping reservation cleanup job")
			return
		}
	}
}

// Stop gracefully stops the cleanup job
func (j *ReservationCleanupJob) Stop() {
	close(j.stopChan)
}

func (j *ReservationCleanupJob) cleanupExpiredReservations(ctx context.Context) {
	log.Debug().Msg("Running expired reservation cleanup")

	// Get expired reservations
	reservations, err := j.inventoryService.reservationRepo.GetExpiredReservations(ctx)
	if err != nil {
		log.Error().Err(err).Msg("Failed to get expired reservations")
		return
	}

	if len(reservations) == 0 {
		log.Debug().Msg("No expired reservations found")
		return
	}

	releasedCount := 0
	failedCount := 0

	for _, reservation := range reservations {
		// Release the reservation
		err := j.inventoryService.reservationRepo.ReleaseReservation(ctx, nil, reservation.ID)
		if err != nil {
			log.Error().Err(err).
				Str("reservation_id", reservation.ID).
				Str("order_id", reservation.OrderID).
				Str("product_id", reservation.ProductID).
				Msg("Failed to release expired reservation")
			failedCount++
			continue
		}

		log.Info().
			Str("reservation_id", reservation.ID).
			Str("order_id", reservation.OrderID).
			Str("product_id", reservation.ProductID).
			Int("quantity", reservation.Quantity).
			Time("expired_at", reservation.ExpiresAt).
			Msg("Expired reservation released")

		releasedCount++
	}

	log.Info().
		Int("total", len(reservations)).
		Int("released", releasedCount).
		Int("failed", failedCount).
		Msg("Completed expired reservation cleanup")
}
