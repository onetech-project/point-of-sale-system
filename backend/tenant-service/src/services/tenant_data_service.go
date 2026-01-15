package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/pos/tenant-service/src/models"
	"github.com/pos/tenant-service/src/repository"
)

// TenantDataService handles aggregation of all tenant data for UU PDP compliance (data access rights)
type TenantDataService struct {
	tenantRepo       *repository.TenantRepository
	tenantConfigRepo *repository.TenantConfigRepository
	db               *sql.DB
}

func NewTenantDataService(
	tenantRepo *repository.TenantRepository,
	tenantConfigRepo *repository.TenantConfigRepository,
	db *sql.DB,
) *TenantDataService {
	return &TenantDataService{
		tenantRepo:       tenantRepo,
		tenantConfigRepo: tenantConfigRepo,
		db:               db,
	}
}

// TenantDataResponse aggregates all tenant data for UU PDP Article 3 (data access rights)
type TenantDataResponse struct {
	Tenant       *models.TenantResponse       `json:"tenant"`
	TeamMembers  []TeamMemberData             `json:"team_members"`
	Configuration *TenantConfigurationData    `json:"configuration"`
}

type TeamMemberData struct {
	ID          string  `json:"id"`
	Email       string  `json:"email"`
	Role        string  `json:"role"`
	Status      string  `json:"status"`
	FirstName   *string `json:"first_name,omitempty"`
	LastName    *string `json:"last_name,omitempty"`
	Locale      string  `json:"locale"`
	CreatedAt   string  `json:"created_at"`
}

type TenantConfigurationData struct {
	EnabledDeliveryTypes []string               `json:"enabled_delivery_types"`
	ServiceArea          map[string]interface{} `json:"service_area"`
	DeliveryFeeConfig    map[string]interface{} `json:"delivery_fee_config"`
	AutoCalculateFees    bool                   `json:"auto_calculate_fees"`
	MidtransConfigured   bool                   `json:"midtrans_configured"` // Don't expose actual keys
	MidtransEnvironment  string                 `json:"midtrans_environment"`
}

// GetAllTenantData aggregates all data for a tenant (business profile, team members, configurations)
// Implements UU PDP Article 3 - Right to Access Personal Data
func (s *TenantDataService) GetAllTenantData(ctx context.Context, tenantID string) (*TenantDataResponse, error) {
	// 1. Get tenant business profile
	tenant, err := s.tenantRepo.FindByID(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get tenant: %w", err)
	}
	if tenant == nil {
		return nil, fmt.Errorf("tenant not found")
	}

	// 2. Get team members (from user-service database)
	teamMembers, err := s.getTeamMembers(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get team members: %w", err)
	}

	// 3. Get tenant configuration (masked sensitive data)
	config, err := s.getTenantConfiguration(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get configuration: %w", err)
	}

	return &TenantDataResponse{
		Tenant:       tenant.ToResponse(),
		TeamMembers:  teamMembers,
		Configuration: config,
	}, nil
}

// getTeamMembers fetches all users associated with the tenant
// Note: In production, this should call user-service API via HTTP/gRPC
// For now, we'll query the users table directly (assumes shared database or federation)
func (s *TenantDataService) getTeamMembers(ctx context.Context, tenantID string) ([]TeamMemberData, error) {
	query := `
		SELECT id, email, role, status, first_name, last_name, locale, created_at
		FROM users
		WHERE tenant_id = $1 AND status != 'deleted'
		ORDER BY created_at ASC
	`

	rows, err := s.db.QueryContext(ctx, query, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var members []TeamMemberData
	for rows.Next() {
		var member TeamMemberData
		var firstName, lastName sql.NullString

		err := rows.Scan(
			&member.ID,
			&member.Email,
			&member.Role,
			&member.Status,
			&firstName,
			&lastName,
			&member.Locale,
			&member.CreatedAt,
		)
		if err != nil {
			return nil, err
		}

		if firstName.Valid {
			member.FirstName = &firstName.String
		}
		if lastName.Valid {
			member.LastName = &lastName.String
		}

		members = append(members, member)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return members, nil
}

// getTenantConfiguration fetches tenant configuration with sensitive data masked
func (s *TenantDataService) getTenantConfiguration(ctx context.Context, tenantID string) (*TenantConfigurationData, error) {
	config, err := s.tenantConfigRepo.GetByTenantID(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	// Mask sensitive payment credentials - only indicate if configured
	midtransConfigured := config.MidtransServerKey != "" && config.MidtransClientKey != ""

	// Parse JSON fields
	var serviceArea, deliveryFeeConfig map[string]interface{}
	if len(config.ServiceArea) > 0 {
		// ServiceArea is already a map[string]interface{}
		serviceArea = config.ServiceArea
	} else {
		serviceArea = map[string]interface{}{}
	}

	if len(config.DeliveryFeeConfig) > 0 {
		// DeliveryFeeConfig is already a map[string]interface{}
		deliveryFeeConfig = config.DeliveryFeeConfig
	} else {
		deliveryFeeConfig = map[string]interface{}{}
	}

	return &TenantConfigurationData{
		EnabledDeliveryTypes: config.EnabledDeliveryTypes,
		ServiceArea:          serviceArea,
		DeliveryFeeConfig:    deliveryFeeConfig,
		AutoCalculateFees:    config.AutoCalculateFees,
		MidtransConfigured:   midtransConfigured,
		MidtransEnvironment:  config.MidtransEnvironment,
	}, nil
}

// ExportData returns tenant data in JSON format for export (UU PDP Article 4 - data portability)
func (s *TenantDataService) ExportData(ctx context.Context, tenantID string) ([]byte, error) {
	data, err := s.GetAllTenantData(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal tenant data: %w", err)
	}

	return jsonData, nil
}
