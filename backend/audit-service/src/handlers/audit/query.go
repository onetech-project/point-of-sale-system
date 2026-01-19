package audit

import (
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog/log"

	"github.com/pos/audit-service/src/repository"
)

// QueryHandler handles HTTP requests for audit trail queries
type QueryHandler struct {
	auditRepo   *repository.AuditRepository
	consentRepo *repository.ConsentRepository
}

// NewQueryHandler creates a new audit query handler
func NewQueryHandler(auditRepo *repository.AuditRepository, consentRepo *repository.ConsentRepository) *QueryHandler {
	return &QueryHandler{
		auditRepo:   auditRepo,
		consentRepo: consentRepo,
	}
}

// ListAuditEvents retrieves audit events with filtering and pagination
// GET /api/v1/audit-events?tenant_id=xxx&actor_type=user&limit=50&offset=0
func (h *QueryHandler) ListAuditEvents(c echo.Context) error {
	ctx := c.Request().Context()

	// Required parameter: tenant_id (multi-tenancy isolation)
	tenantID := c.QueryParam("tenant_id")
	if tenantID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "tenant_id is required",
		})
	}

	// Build filter from query parameters
	filter := repository.AuditQueryFilter{
		TenantID: tenantID,
		Limit:    50, // Default limit
		Offset:   0,
	}

	// Optional filters
	if actorType := c.QueryParam("actor_type"); actorType != "" {
		filter.ActorType = &actorType
	}
	if actorID := c.QueryParam("actor_id"); actorID != "" {
		filter.ActorID = &actorID
	}
	if action := c.QueryParam("action"); action != "" {
		filter.Action = &action
	}
	if resourceType := c.QueryParam("resource_type"); resourceType != "" {
		filter.ResourceType = &resourceType
	}
	if resourceID := c.QueryParam("resource_id"); resourceID != "" {
		filter.ResourceID = &resourceID
	}

	// Time range filters
	if startTimeStr := c.QueryParam("start_time"); startTimeStr != "" {
		startTime, err := time.Parse(time.RFC3339, startTimeStr)
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "invalid start_time format (expected RFC3339)",
			})
		}
		filter.StartTime = &startTime
	}
	if endTimeStr := c.QueryParam("end_time"); endTimeStr != "" {
		endTime, err := time.Parse(time.RFC3339, endTimeStr)
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "invalid end_time format (expected RFC3339)",
			})
		}
		filter.EndTime = &endTime
	}

	// Pagination
	if limitStr := c.QueryParam("limit"); limitStr != "" {
		limit, err := strconv.Atoi(limitStr)
		if err != nil || limit < 1 || limit > 1000 {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "invalid limit (must be 1-1000)",
			})
		}
		filter.Limit = limit
	}
	if offsetStr := c.QueryParam("offset"); offsetStr != "" {
		offset, err := strconv.Atoi(offsetStr)
		if err != nil || offset < 0 {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "invalid offset (must be >= 0)",
			})
		}
		filter.Offset = offset
	}

	// Retrieve audit events
	events, err := h.auditRepo.List(ctx, filter)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "failed to retrieve audit events",
		})
	}

	// Get total count for pagination
	total, err := h.auditRepo.Count(ctx, filter)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "failed to count audit events",
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"events": events,
		"pagination": map[string]interface{}{
			"total":  total,
			"limit":  filter.Limit,
			"offset": filter.Offset,
		},
	})
}

// GetAuditEvent retrieves a single audit event by ID
// GET /api/v1/audit-events/:event_id
func (h *QueryHandler) GetAuditEvent(c echo.Context) error {
	ctx := c.Request().Context()

	eventIDStr := c.Param("event_id")
	eventID, err := uuid.Parse(eventIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "invalid event_id format (expected UUID)",
		})
	}

	event, err := h.auditRepo.GetByID(ctx, eventID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "audit event not found",
		})
	}

	return c.JSON(http.StatusOK, event)
}

// ListConsentRecords retrieves consent records with filtering and pagination
// GET /api/v1/consent-records?tenant_id=xxx&subject_type=tenant&subject_id=yyy
func (h *QueryHandler) ListConsentRecords(c echo.Context) error {
	ctx := c.Request().Context()

	// Required parameter: tenant_id
	tenantID := c.QueryParam("tenant_id")
	if tenantID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "tenant_id is required",
		})
	}

	// Build filter from query parameters
	filter := repository.ConsentQueryFilter{
		TenantID: tenantID,
		Limit:    50,
		Offset:   0,
	}

	// Optional filters
	if subjectType := c.QueryParam("subject_type"); subjectType != "" {
		filter.SubjectType = &subjectType
	}
	if subjectID := c.QueryParam("subject_id"); subjectID != "" {
		filter.SubjectID = &subjectID
	}
	if purposeCode := c.QueryParam("purpose_code"); purposeCode != "" {
		filter.PurposeCode = &purposeCode
	}
	if grantedStr := c.QueryParam("granted"); grantedStr != "" {
		granted, err := strconv.ParseBool(grantedStr)
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "invalid granted value (expected boolean)",
			})
		}
		filter.Granted = &granted
	}

	// Pagination
	if limitStr := c.QueryParam("limit"); limitStr != "" {
		limit, err := strconv.Atoi(limitStr)
		if err != nil || limit < 1 || limit > 1000 {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "invalid limit (must be 1-1000)",
			})
		}
		filter.Limit = limit
	}
	if offsetStr := c.QueryParam("offset"); offsetStr != "" {
		offset, err := strconv.Atoi(offsetStr)
		if err != nil || offset < 0 {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "invalid offset (must be >= 0)",
			})
		}
		filter.Offset = offset
	}

	// Retrieve consent records
	records, err := h.consentRepo.ListConsentRecords(ctx, filter)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "failed to retrieve consent records",
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"records": records,
		"pagination": map[string]interface{}{
			"limit":  filter.Limit,
			"offset": filter.Offset,
		},
	})
}

// ListTenantAuditEvents retrieves audit events for the authenticated tenant (T107, T108, T109)
// GET /api/v1/audit/tenant?action=CREATE&resource_type=user&start_time=2026-01-01T00:00:00Z&limit=100
func (h *QueryHandler) ListTenantAuditEvents(c echo.Context) error {
	ctx := c.Request().Context()

	// Extract tenant_id from JWT claims (set by auth middleware)
	tenantIDInterface := c.Get("tenant_id")
	if tenantIDInterface == nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "Missing tenant_id in authentication context",
		})
	}
	tenantID, ok := tenantIDInterface.(string)
	if !ok || tenantID == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "Invalid tenant_id in authentication context",
		})
	}

	// Build filter from query parameters
	filter := repository.AuditQueryFilter{
		TenantID: tenantID, // From JWT - enforces tenant isolation
		Limit:    100,      // Default limit (T108)
		Offset:   0,
	}

	// Optional filters: action_type (action), resource_type, actor_id, date_range (start_time/end_time)
	if action := c.QueryParam("action"); action != "" {
		filter.Action = &action
	}
	if resourceType := c.QueryParam("resource_type"); resourceType != "" {
		filter.ResourceType = &resourceType
	}
	if actorID := c.QueryParam("actor_id"); actorID != "" {
		filter.ActorID = &actorID
	}

	// Date range filters
	if startTimeStr := c.QueryParam("start_time"); startTimeStr != "" {
		startTime, err := time.Parse(time.RFC3339, startTimeStr)
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "invalid start_time format (expected RFC3339)",
			})
		}
		filter.StartTime = &startTime
	}
	if endTimeStr := c.QueryParam("end_time"); endTimeStr != "" {
		endTime, err := time.Parse(time.RFC3339, endTimeStr)
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "invalid end_time format (expected RFC3339)",
			})
		}
		filter.EndTime = &endTime
	}

	// Pagination (T108)
	if limitStr := c.QueryParam("limit"); limitStr != "" {
		limit, err := strconv.Atoi(limitStr)
		if err != nil || limit < 1 || limit > 1000 {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "invalid limit (must be 1-1000)",
			})
		}
		filter.Limit = limit
	}
	if offsetStr := c.QueryParam("offset"); offsetStr != "" {
		offset, err := strconv.Atoi(offsetStr)
		if err != nil || offset < 0 {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "invalid offset (must be >= 0)",
			})
		}
		filter.Offset = offset
	}

	// Retrieve audit events for tenant
	events, err := h.auditRepo.List(ctx, filter)
	if err != nil {
		log.Error().Err(err).Msg("Failed to retrieve tenant audit events")
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "failed to retrieve audit events",
		})
	}

	// Get total count for pagination
	total, err := h.auditRepo.Count(ctx, filter)
	if err != nil {
		log.Error().Err(err).Msg("Failed to count tenant audit events")
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "failed to count audit events",
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"events": events,
		"pagination": map[string]interface{}{
			"total":  total,
			"limit":  filter.Limit,
			"offset": filter.Offset,
		},
	})
}
