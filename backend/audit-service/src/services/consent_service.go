package services

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/pos/audit-service/src/events"
	"github.com/pos/audit-service/src/models"
	"github.com/pos/audit-service/src/queue"
	"github.com/pos/audit-service/src/repository"
	"github.com/rs/zerolog/log"
)

// ConsentService handles business logic for consent management
type ConsentService struct {
	consentRepo *repository.ConsentRepository
	producer    *queue.KafkaProducer
}

// NewConsentService creates a new consent service
func NewConsentService(consentRepo *repository.ConsentRepository, producer *queue.KafkaProducer) *ConsentService {
	return &ConsentService{
		consentRepo: consentRepo,
		producer:    producer,
	}
}

// ConsentGrantRequest represents a request to grant consent
type ConsentGrantRequest struct {
	TenantID      string
	SubjectType   string // "tenant" or "guest"
	SubjectID     string
	PurposeCodes  []string
	PolicyVersion string
	ConsentMethod string
	IPAddress     string
	UserAgent     string
}

// ValidateConsents checks if all required consent purposes are included for a given context
func (s *ConsentService) ValidateConsents(ctx context.Context, purposeCodes []string, contextFilter string) error {
	// Get consent purposes for the specified context (tenant or guest)
	purposes, err := s.consentRepo.ListConsentPurposes(ctx, "en", contextFilter)
	if err != nil {
		return fmt.Errorf("failed to list consent purposes: %w", err)
	}

	// Create map of provided purposes
	provided := make(map[string]bool)
	for _, code := range purposeCodes {
		provided[code] = true
	}

	// Check all required purposes are included
	var missing []string
	for _, purpose := range purposes {
		if purpose.IsRequired && !provided[purpose.PurposeCode] {
			missing = append(missing, purpose.PurposeCode)
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("missing required consent purposes: %v", missing)
	}

	return nil
}

// GrantConsents creates consent records for the provided purposes
func (s *ConsentService) GrantConsents(ctx context.Context, req ConsentGrantRequest) error {
	// Determine context from subject type
	contextFilter := req.SubjectType
	if contextFilter != "tenant" && contextFilter != "guest" {
		return fmt.Errorf("invalid subject_type: must be 'tenant' or 'guest'")
	}

	// Validate that all required purposes are included for the given context
	if err := s.ValidateConsents(ctx, req.PurposeCodes, contextFilter); err != nil {
		return err
	}

	// Get current privacy policy if version not specified
	policyVersion := req.PolicyVersion
	if policyVersion == "" {
		policy, err := s.consentRepo.GetCurrentPrivacyPolicy(ctx, "en")
		if err != nil {
			return fmt.Errorf("failed to get current privacy policy: %w", err)
		}
		policyVersion = policy.Version
	}

	// Create consent record for each purpose
	for _, purposeCode := range req.PurposeCodes {
		// Verify purpose exists
		_, err := s.consentRepo.GetConsentPurposeByCode(ctx, purposeCode)
		if err != nil {
			return fmt.Errorf("invalid purpose code %s: %w", purposeCode, err)
		}

		// Prepare pointer fields
		subjectID := req.SubjectID
		ipAddr := req.IPAddress
		userAgent := req.UserAgent

		record := &models.ConsentRecord{
			RecordID:      uuid.New(),
			TenantID:      req.TenantID,
			SubjectType:   req.SubjectType,
			SubjectID:     &subjectID,
			PurposeCode:   purposeCode,
			Granted:       true,
			PolicyVersion: policyVersion,
			ConsentMethod: req.ConsentMethod,
			IPAddress:     &ipAddr,
			UserAgent:     &userAgent,
		}

		if err := s.consentRepo.CreateConsentRecord(ctx, record); err != nil {
			return fmt.Errorf("failed to create consent record for %s: %w", purposeCode, err)
		}
	}

	return nil
}

// RevokeConsentRequest represents a request to revoke consent
type RevokeConsentRequest struct {
	TenantID    string
	SubjectType string
	SubjectID   string
	PurposeCode string
	IPAddress   string
	UserAgent   string
}

// RevokeConsent revokes consent for a specific purpose
func (s *ConsentService) RevokeConsent(ctx context.Context, req RevokeConsentRequest) error {
	// Check if purpose is required (cannot revoke)
	purpose, err := s.consentRepo.GetConsentPurposeByCode(ctx, req.PurposeCode)
	if err != nil {
		return fmt.Errorf("invalid purpose code: %w", err)
	}

	if purpose.IsRequired {
		return fmt.Errorf("cannot revoke required consent purpose: %s", req.PurposeCode)
	}

	// Find active consent record for this purpose
	activeConsents, err := s.consentRepo.GetActiveConsents(ctx, req.TenantID, req.SubjectType, req.SubjectID)
	if err != nil {
		return fmt.Errorf("failed to get active consents: %w", err)
	}

	var targetRecord *models.ConsentRecord
	for _, consent := range activeConsents {
		if consent.PurposeCode == req.PurposeCode {
			targetRecord = consent
			break
		}
	}

	if targetRecord == nil {
		return fmt.Errorf("no active consent found for purpose: %s", req.PurposeCode)
	}

	// Revoke the consent
	if err := s.consentRepo.RevokeConsent(ctx, targetRecord.RecordID); err != nil {
		return fmt.Errorf("failed to revoke consent: %w", err)
	}

	// Publish ConsentRevokedEvent to audit topic for compliance trail
	event := events.ConsentRevokedEvent{
		EventID:       uuid.New().String(),
		EventType:     "consent.revoked",
		TenantID:      req.TenantID,
		SubjectType:   req.SubjectType,
		SubjectID:     req.SubjectID,
		PurposeCode:   req.PurposeCode,
		PurposeName:   purpose.DisplayNameEN, // Use English name for event
		RevokedAt:     time.Now(),
		IPAddress:     req.IPAddress,
		UserAgent:     req.UserAgent,
		Timestamp:     time.Now(),
		ComplianceTag: "UU_PDP_Article_21", // Right to revoke consent
	}

	if err := s.producer.Publish(ctx, req.TenantID, event); err != nil {
		// Log error but don't fail the operation - event publishing is async
		log.Error().
			Err(err).
			Str("tenant_id", req.TenantID).
			Str("purpose_code", req.PurposeCode).
			Msg("Failed to publish ConsentRevokedEvent")
	}

	return nil
}

// GetConsentStatus retrieves current consent status for a subject
func (s *ConsentService) GetConsentStatus(ctx context.Context, tenantID, subjectType, subjectID string) (map[string]bool, error) {
	// Get all active consents
	activeConsents, err := s.consentRepo.GetActiveConsents(ctx, tenantID, subjectType, subjectID)
	if err != nil {
		return nil, fmt.Errorf("failed to get active consents: %w", err)
	}

	// Create map of active consent purposes
	status := make(map[string]bool)
	for _, consent := range activeConsents {
		status[consent.PurposeCode] = true
	}

	return status, nil
}

// GetConsentHistory retrieves full consent history for a subject
func (s *ConsentService) GetConsentHistory(ctx context.Context, tenantID, subjectType, subjectID string) ([]*models.ConsentRecord, error) {
	history, err := s.consentRepo.GetConsentHistory(ctx, tenantID, subjectType, subjectID)
	if err != nil {
		return nil, fmt.Errorf("failed to get consent history: %w", err)
	}
	return history, nil
}

// HasActiveConsent checks if a subject has active consent for a specific purpose
func (s *ConsentService) HasActiveConsent(ctx context.Context, tenantID, subjectType, subjectID, purposeCode string) (bool, error) {
	status, err := s.GetConsentStatus(ctx, tenantID, subjectType, subjectID)
	if err != nil {
		return false, err
	}
	return status[purposeCode], nil
}

// CheckConsentForPurpose checks if subject has active consent, returns error if not
// This is a convenience method for enforcement in services that require consent
func (s *ConsentService) CheckConsentForPurpose(ctx context.Context, tenantID, subjectType, subjectID, purposeCode string) error {
	hasConsent, err := s.HasActiveConsent(ctx, tenantID, subjectType, subjectID, purposeCode)
	if err != nil {
		return fmt.Errorf("failed to check consent: %w", err)
	}
	
	if !hasConsent {
		return fmt.Errorf("user does not have active consent for purpose: %s", purposeCode)
	}
	
	return nil
}

