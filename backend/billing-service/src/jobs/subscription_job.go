package jobs

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/pos/billing-service/src/models"
	"github.com/pos/billing-service/src/queue"
	"github.com/pos/billing-service/src/repository"
	"github.com/pos/billing-service/src/utils"
)

// JobRunner orchestrates background billing jobs.
type JobRunner struct {
	db               *sql.DB
	repo             *repository.BillingRepository
	publisher        *queue.EventPublisher
	cacheInvalidator SubscriptionCacheInvalidator
}

type SubscriptionCacheInvalidator interface {
	InvalidateSubscriptionStatus(ctx context.Context, tenantID string) error
}

// NewJobRunner creates a new JobRunner.
func NewJobRunner(db *sql.DB, repo *repository.BillingRepository, publisher *queue.EventPublisher) *JobRunner {
	return &JobRunner{db: db, repo: repo, publisher: publisher}
}

// SetSubscriptionCacheInvalidator wires gateway cache invalidation for status-changing jobs.
func (j *JobRunner) SetSubscriptionCacheInvalidator(invalidator SubscriptionCacheInvalidator) {
	j.cacheInvalidator = invalidator
}

// StartAll launches all background jobs as goroutines.
func (j *JobRunner) StartAll() {
	go j.runTrialExpiryJob()
	go j.runGraceEnforcerJob()
	go j.runInvoiceGeneratorJob()
	go j.runTenantRetentionJob()
}

// --- Trial expiry job (every 6 hours) ---

func (j *JobRunner) runTrialExpiryJob() {
	j.processTrialExpiry()
	ticker := time.NewTicker(6 * time.Hour)
	defer ticker.Stop()
	for range ticker.C {
		j.processTrialExpiry()
	}
}

func (j *JobRunner) processTrialExpiry() {
	ctx := context.Background()
	graceDays := utils.GetEnvInt("PLAN_GRACE_PERIOD_DAYS", 7)

	// Notify tenants with trials ending in the next 3 days (D-3 and D-1 alerts).
	expiring, err := j.repo.GetTenantsWithExpiringTrials(ctx, 3)
	if err != nil {
		log.Printf("TrialExpiryJob: error fetching expiring trials: %v", err)
	} else {
		for _, t := range expiring {
			if t.TrialEndsAt == nil {
				continue
			}
			hours := time.Until(*t.TrialEndsAt).Hours()
			var daysRemaining int
			switch {
			case hours >= 48 && hours < 96: // ~3 days
				daysRemaining = 3
			case hours >= 0 && hours < 48: // ~1 day
				daysRemaining = 1
			default:
				continue
			}
			email, _ := j.repo.GetTenantOwnerEmail(ctx, t.ID)
			if err := j.publisher.PublishTrialEnding(ctx, t.ID, email, t.BusinessName, *t.TrialEndsAt, daysRemaining); err != nil {
				log.Printf("TrialExpiryJob: failed to publish trial_ending for %s: %v", t.ID, err)
			}
		}
	}

	// Move expired trials to grace_period.
	expired, err := j.repo.GetExpiredTrialTenants(ctx)
	if err != nil {
		log.Printf("TrialExpiryJob: error fetching expired trials: %v", err)
		return
	}
	for _, t := range expired {
		if err := j.repo.UpdateTenantSubscriptionStatus(ctx, t.ID, "grace_period"); err != nil {
			log.Printf("TrialExpiryJob: failed to update status for %s: %v", t.ID, err)
			continue
		}
		j.invalidateSubscriptionCache(ctx, t.ID)
		graceEndsAt := time.Now()
		if t.TrialEndsAt != nil {
			graceEndsAt = t.TrialEndsAt.Add(time.Duration(graceDays) * 24 * time.Hour)
		}
		email, _ := j.repo.GetTenantOwnerEmail(ctx, t.ID)
		if err := j.publisher.PublishGracePeriodStarted(ctx, t.ID, email, t.BusinessName, graceEndsAt); err != nil {
			log.Printf("TrialExpiryJob: failed to publish grace_period_started for %s: %v", t.ID, err)
		}
	}

	// Also handle active subscriptions that have expired.
	expiredSubs, err := j.repo.GetExpiredSubscriptionTenants(ctx)
	if err != nil {
		log.Printf("TrialExpiryJob: error fetching expired subscriptions: %v", err)
		return
	}
	for _, t := range expiredSubs {
		if err := j.repo.UpdateTenantSubscriptionStatus(ctx, t.ID, "grace_period"); err != nil {
			log.Printf("TrialExpiryJob: failed to update status for %s: %v", t.ID, err)
			continue
		}
		j.invalidateSubscriptionCache(ctx, t.ID)
		graceEndsAt := time.Now()
		if t.SubscriptionEndsAt != nil {
			graceEndsAt = t.SubscriptionEndsAt.Add(time.Duration(graceDays) * 24 * time.Hour)
		}
		email, _ := j.repo.GetTenantOwnerEmail(ctx, t.ID)
		if err := j.publisher.PublishGracePeriodStarted(ctx, t.ID, email, t.BusinessName, graceEndsAt); err != nil {
			log.Printf("TrialExpiryJob: failed to publish grace_period_started for %s: %v", t.ID, err)
		}
	}
}

// --- Grace period enforcer job (every 6 hours) ---

func (j *JobRunner) runGraceEnforcerJob() {
	j.processGraceExpiry()
	ticker := time.NewTicker(6 * time.Hour)
	defer ticker.Stop()
	for range ticker.C {
		j.processGraceExpiry()
	}
}

func (j *JobRunner) processGraceExpiry() {
	ctx := context.Background()
	graceDays := utils.GetEnvInt("PLAN_GRACE_PERIOD_DAYS", 7)

	tenants, err := j.repo.GetGracePeriodExpiredTenants(ctx, graceDays)
	if err != nil {
		log.Printf("GraceEnforcerJob: error fetching grace-expired tenants: %v", err)
		return
	}
	for _, t := range tenants {
		if err := j.repo.UpdateTenantSubscriptionStatus(ctx, t.ID, "expired"); err != nil {
			log.Printf("GraceEnforcerJob: failed to expire tenant %s: %v", t.ID, err)
			continue
		}
		j.invalidateSubscriptionCache(ctx, t.ID)
		email, _ := j.repo.GetTenantOwnerEmail(ctx, t.ID)
		if err := j.publisher.PublishSubscriptionExpired(ctx, t.ID, email, t.BusinessName); err != nil {
			log.Printf("GraceEnforcerJob: failed to publish expired for %s: %v", t.ID, err)
		}
	}
}

func (j *JobRunner) invalidateSubscriptionCache(ctx context.Context, tenantID string) {
	if j.cacheInvalidator == nil {
		return
	}
	if err := j.cacheInvalidator.InvalidateSubscriptionStatus(ctx, tenantID); err != nil {
		log.Printf("SubscriptionJob: failed to invalidate subscription cache for tenant %s: %v", tenantID, err)
	}
}

// --- Invoice generator job (daily) ---

func (j *JobRunner) runInvoiceGeneratorJob() {
	j.processInvoiceGeneration()
	// Run at approximately midnight each day.
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()
	for range ticker.C {
		j.processInvoiceGeneration()
	}
}

func (j *JobRunner) processInvoiceGeneration() {
	ctx := context.Background()

	// Find active subscribers whose subscription ends in the next 3 days.
	tenants, err := j.repo.GetTenantsNearingRenewal(ctx, 3)
	if err != nil {
		log.Printf("InvoiceGeneratorJob: error fetching tenants nearing renewal: %v", err)
		return
	}
	for _, t := range tenants {
		existing, err := j.repo.GetPendingInvoiceByTenantID(ctx, t.ID)
		if err != nil {
			log.Printf("InvoiceGeneratorJob: error checking pending invoice for %s: %v", t.ID, err)
			continue
		}
		if existing != nil {
			continue // already has a pending invoice
		}

		inv, err := j.buildRenewalInvoice(ctx, t)
		if err != nil {
			log.Printf("InvoiceGeneratorJob: failed to create invoice for %s: %v", t.ID, err)
			continue
		}

		email, _ := j.repo.GetTenantOwnerEmail(ctx, t.ID)
		paymentURL := ""
		if inv.MidtransPaymentURL != nil {
			paymentURL = *inv.MidtransPaymentURL
		}
		if err := j.publisher.PublishInvoiceGenerated(ctx, t.ID, email, t.BusinessName,
			inv.InvoiceNumber, inv.AmountIDR, inv.BillingInterval,
			inv.PeriodStart, inv.PeriodEnd, paymentURL,
		); err != nil {
			log.Printf("InvoiceGeneratorJob: failed to publish invoice.generated for %s: %v", t.ID, err)
		}
	}
}

func (j *JobRunner) buildRenewalInvoice(ctx context.Context, t *models.Tenant) (*models.BillingInvoice, error) {
	billingInterval := t.BillingCycle
	monthly := utils.GetEnvInt("PLAN_MONTHLY_PRICE_IDR", 299000)
	var amount int
	if billingInterval == "annual" {
		discountPct := utils.GetEnvInt("PLAN_ANNUAL_DISCOUNT_PCT", 20)
		annual := monthly * 12
		amount = annual - (annual * discountPct / 100)
	} else {
		amount = monthly
	}

	var periodStart time.Time
	if t.SubscriptionEndsAt != nil {
		periodStart = *t.SubscriptionEndsAt
	} else {
		periodStart = time.Now().UTC()
	}
	var periodEnd time.Time
	if billingInterval == "annual" {
		periodEnd = periodStart.AddDate(1, 0, 0)
	} else {
		periodEnd = periodStart.AddDate(0, 1, 0)
	}

	count, err := j.repo.CountInvoicesThisMonth(ctx)
	if err != nil {
		return nil, err
	}
	invoiceNumber := fmt.Sprintf("INV-%s-%06d", time.Now().UTC().Format("200601"), count+1)

	inv := &models.BillingInvoice{
		TenantID:        t.ID,
		InvoiceNumber:   invoiceNumber,
		AmountIDR:       amount,
		BillingInterval: billingInterval,
		PeriodStart:     periodStart,
		PeriodEnd:       periodEnd,
		Status:          "pending",
	}
	if err := j.repo.CreateInvoice(ctx, inv); err != nil {
		return nil, err
	}
	return inv, nil
}

// --- Tenant retention cleanup job (daily) ---

func (j *JobRunner) runTenantRetentionJob() {
	j.processTenantRetention()
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()
	for range ticker.C {
		j.processTenantRetention()
	}
}

func (j *JobRunner) processTenantRetention() {
	ctx := context.Background()
	retentionDays := utils.GetEnvInt("PLAN_RETENTION_DAYS", 30)

	tenants, err := j.repo.GetTenantsDueForSubscriptionRetention(ctx, retentionDays)
	if err != nil {
		log.Printf("TenantRetentionJob: error fetching tenants due for cleanup: %v", err)
		return
	}

	for _, t := range tenants {
		j.deleteTenantPhotoObjects(ctx, t.ID)

		anonymizedAt, err := j.repo.AnonymizeTenantOperationalData(ctx, t.ID)
		if err != nil {
			log.Printf("TenantRetentionJob: failed to anonymize operational data for tenant %s: %v", t.ID, err)
			continue
		}
		if j.publisher != nil {
			if err := j.publisher.PublishTenantSubscriptionDataAnonymized(ctx, t.ID, t.RetentionStartedAt, anonymizedAt); err != nil {
				log.Printf("TenantRetentionJob: failed to publish anonymization audit event for tenant %s: %v", t.ID, err)
			}
		}
	}
}

func (j *JobRunner) deleteTenantPhotoObjects(ctx context.Context, tenantID string) {
	endpoint := os.Getenv("S3_ENDPOINT")
	accessKey := os.Getenv("S3_ACCESS_KEY")
	secretKey := os.Getenv("S3_SECRET_KEY")
	bucket := os.Getenv("S3_BUCKET_NAME")
	if endpoint == "" || accessKey == "" || secretKey == "" || bucket == "" {
		return
	}

	keys, err := j.repo.GetTenantPhotoStorageKeys(ctx, tenantID)
	if err != nil {
		log.Printf("TenantRetentionJob: failed to list photo objects for tenant %s: %v", tenantID, err)
		return
	}
	if len(keys) == 0 {
		return
	}

	useSSL := os.Getenv("S3_USE_SSL") == "true"
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		log.Printf("TenantRetentionJob: failed to initialize object storage client for tenant %s: %v", tenantID, err)
		return
	}

	for _, key := range keys {
		if err := client.RemoveObject(ctx, bucket, key, minio.RemoveObjectOptions{}); err != nil {
			log.Printf("TenantRetentionJob: failed to delete photo object %s for tenant %s: %v", key, tenantID, err)
		}
	}
}
