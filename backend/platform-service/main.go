package main

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	emw "github.com/labstack/echo/v4/middleware"
	_ "github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

type PlatformService struct {
	db         *sql.DB
	jwtSecret  []byte
	sessionTTL time.Duration
}

type PlatformClaims struct {
	SessionID       string `json:"sessionId"`
	PlatformAdminID string `json:"platformAdminId"`
	Email           string `json:"email"`
	Role            string `json:"role"`
	jwt.RegisteredClaims
}

type PlatformAdmin struct {
	ID        string     `json:"id"`
	Email     string     `json:"email"`
	Name      string     `json:"name"`
	Role      string     `json:"role"`
	Status    string     `json:"status"`
	LastLogin *time.Time `json:"last_login_at,omitempty"`
}

type TenantSummary struct {
	ID                  string     `json:"id"`
	BusinessName        string     `json:"business_name"`
	Slug                string     `json:"slug"`
	Status              string     `json:"status"`
	SubscriptionPlan    string     `json:"subscription_plan"`
	BillingCycle        string     `json:"billing_cycle"`
	SubscriptionStatus  string     `json:"subscription_status"`
	TrialEndsAt         *time.Time `json:"trial_ends_at,omitempty"`
	SubscriptionEndsAt  *time.Time `json:"subscription_ends_at,omitempty"`
	StorageUsedBytes    int64      `json:"storage_used_bytes"`
	StorageQuotaBytes   int64      `json:"storage_quota_bytes"`
	UserCount           int64      `json:"user_count"`
	PaidInvoiceTotalIDR int64      `json:"paid_invoice_total_idr"`
	OpenTicketCount     int64      `json:"open_ticket_count"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
	SuspendedAt         *time.Time `json:"suspended_at,omitempty"`
	DeactivatedAt       *time.Time `json:"deactivated_at,omitempty"`
	ScheduledDeleteAt   *time.Time `json:"scheduled_delete_at,omitempty"`
	StatusReason        *string    `json:"status_reason,omitempty"`
}

type Ticket struct {
	ID             string     `json:"id"`
	TenantID       string     `json:"tenant_id"`
	TenantName     string     `json:"tenant_name,omitempty"`
	Subject        string     `json:"subject"`
	Description    string     `json:"description"`
	Category       string     `json:"category"`
	Priority       string     `json:"priority"`
	Status         string     `json:"status"`
	AssigneeID     *string    `json:"assigned_to_platform_admin_id,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	LastActivityAt time.Time  `json:"last_activity_at"`
	ResolvedAt     *time.Time `json:"resolved_at,omitempty"`
}

type ActivityItem struct {
	Type        string     `json:"type"`
	Action      string     `json:"action"`
	ActorType   string     `json:"actor_type,omitempty"`
	Resource    string     `json:"resource,omitempty"`
	Reason      *string    `json:"reason,omitempty"`
	Before      *string    `json:"before_status,omitempty"`
	After       *string    `json:"after_status,omitempty"`
	OccurredAt  time.Time  `json:"occurred_at"`
	DeleteAfter *time.Time `json:"delete_after,omitempty"`
}

func main() {
	db, err := sql.Open("postgres", getEnv("DATABASE_URL", "postgresql://pos_user:pos_password@localhost:5432/pos_db?sslmode=disable"))
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()
	if err := db.Ping(); err != nil {
		log.Fatalf("failed to ping database: %v", err)
	}

	sessionTTL := time.Duration(getEnvInt("SESSION_TTL_MINUTES", 720)) * time.Minute
	svc := &PlatformService{
		db:         db,
		jwtSecret:  []byte(getEnv("JWT_SECRET", "change-me")),
		sessionTTL: sessionTTL,
	}
	if err := svc.bootstrapAdmin(context.Background()); err != nil {
		log.Printf("platform admin bootstrap skipped: %v", err)
	}

	e := echo.New()
	e.Use(emw.Recover())
	e.Use(emw.Logger())

	e.GET("/health", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok", "service": getEnv("SERVICE_NAME", "platform-service")})
	})

	api := e.Group("/api/v1/platform")
	api.POST("/auth/login", svc.Login)
	api.POST("/tickets", svc.CreateTicket)

	platform := api.Group("")
	platform.Use(svc.RequirePlatformAdmin)
	platform.GET("/auth/session", svc.Session)
	platform.POST("/auth/logout", svc.Logout)
	platform.GET("/analytics/overview", svc.AnalyticsOverview)
	platform.GET("/tenants", svc.ListTenants)
	platform.GET("/tenants/:tenant_id", svc.GetTenant)
	platform.GET("/tenants/:tenant_id/activity", svc.GetTenantActivity)
	platform.POST("/tenants/:tenant_id/actions/:action", svc.TenantAction)
	platform.GET("/tickets", svc.ListTickets)
	platform.PATCH("/tickets/:ticket_id", svc.UpdateTicket)
	platform.POST("/tickets/:ticket_id/notes", svc.AddTicketNote)

	port := getEnv("PORT", "8091")
	log.Printf("Platform service starting on port %s", port)
	e.Logger.Fatal(e.Start(":" + port))
}

func (s *PlatformService) bootstrapAdmin(ctx context.Context) error {
	var count int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM platform_admins`).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	email := getEnv("PLATFORM_ADMIN_EMAIL", "")
	password := getEnv("PLATFORM_ADMIN_PASSWORD", "")
	name := getEnv("PLATFORM_ADMIN_NAME", "Platform Owner")
	if email == "" || password == "" {
		return errors.New("PLATFORM_ADMIN_EMAIL and PLATFORM_ADMIN_PASSWORD are required when no platform admins exist")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO platform_admins (email, password_hash, name, role, status)
		VALUES ($1, $2, $3, 'platform_owner', 'active')`,
		strings.ToLower(strings.TrimSpace(email)), string(hash), name)
	return err
}

func (s *PlatformService) Login(c echo.Context) error {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := c.Bind(&req); err != nil || req.Email == "" || req.Password == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "email and password are required"})
	}

	var admin PlatformAdmin
	var passwordHash string
	var lastLogin sql.NullTime
	err := s.db.QueryRowContext(c.Request().Context(), `
		SELECT id, email, name, role, status, password_hash, last_login_at
		FROM platform_admins
		WHERE email = $1`,
		strings.ToLower(strings.TrimSpace(req.Email)),
	).Scan(&admin.ID, &admin.Email, &admin.Name, &admin.Role, &admin.Status, &passwordHash, &lastLogin)
	if err == sql.ErrNoRows {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid email or password"})
	}
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to login"})
	}
	if bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(req.Password)) != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid email or password"})
	}
	if admin.Status != "active" {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "platform admin is suspended"})
	}
	if lastLogin.Valid {
		admin.LastLogin = &lastLogin.Time
	}

	sessionID := uuid.NewString()
	expiresAt := time.Now().UTC().Add(s.sessionTTL)
	_, err = s.db.ExecContext(c.Request().Context(), `
		INSERT INTO platform_admin_sessions (session_id, platform_admin_id, ip_address, user_agent, expires_at)
		VALUES ($1, $2, $3, $4, $5)`,
		sessionID, admin.ID, c.RealIP(), c.Request().UserAgent(), expiresAt)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to create session"})
	}

	token, err := s.signToken(sessionID, admin, expiresAt)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to sign session"})
	}
	_, _ = s.db.ExecContext(c.Request().Context(), `UPDATE platform_admins SET last_login_at = NOW(), updated_at = NOW() WHERE id = $1`, admin.ID)

	secureCookie := c.Request().Header.Get("X-Forwarded-Proto") == "https"
	c.SetCookie(&http.Cookie{
		Name:     "platform_auth_token",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   secureCookie,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(s.sessionTTL.Seconds()),
	})
	return c.JSON(http.StatusOK, map[string]interface{}{"admin": admin, "message": "login successful"})
}

func (s *PlatformService) Session(c echo.Context) error {
	if admin, ok := c.Get("platform_admin").(*PlatformAdmin); ok {
		return c.JSON(http.StatusOK, map[string]interface{}{"admin": admin})
	}
	return c.JSON(http.StatusUnauthorized, map[string]string{"error": "platform session not found"})
}

func (s *PlatformService) Logout(c echo.Context) error {
	sessionID := c.Request().Header.Get("X-Platform-Session-ID")
	if sessionID != "" {
		_, _ = s.db.ExecContext(c.Request().Context(), `UPDATE platform_admin_sessions SET terminated_at = NOW() WHERE session_id = $1`, sessionID)
	}
	c.SetCookie(&http.Cookie{Name: "platform_auth_token", Value: "", Path: "/", HttpOnly: true, SameSite: http.SameSiteLaxMode, MaxAge: -1})
	return c.JSON(http.StatusOK, map[string]string{"message": "logged out"})
}

func (s *PlatformService) RequirePlatformAdmin(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		adminID := c.Request().Header.Get("X-Platform-Admin-ID")
		sessionID := c.Request().Header.Get("X-Platform-Session-ID")
		if adminID == "" || sessionID == "" {
			return c.JSON(http.StatusUnauthorized, map[string]string{"error": "missing platform session"})
		}
		admin, err := s.getAdminForSession(c.Request().Context(), adminID, sessionID)
		if err != nil {
			return c.JSON(http.StatusUnauthorized, map[string]string{"error": "platform session not found"})
		}
		c.Set("platform_admin", admin)
		return next(c)
	}
}

func (s *PlatformService) AnalyticsOverview(c echo.Context) error {
	start, end, err := parseDateRange(c)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	ctx := c.Request().Context()

	overview := map[string]interface{}{
		"start_date": start.Format("2006-01-02"),
		"end_date":   end.Format("2006-01-02"),
	}
	scalars := map[string]string{
		"active_tenants":        `SELECT COUNT(*) FROM tenants WHERE status = 'active'`,
		"active_users":          `SELECT COUNT(*) FROM users WHERE status = 'active'`,
		"recently_active_users": `SELECT COUNT(*) FROM users WHERE status = 'active' AND last_login_at >= NOW() - INTERVAL '7 days'`,
		"grace_period_tenants":  `SELECT COUNT(*) FROM tenants WHERE subscription_status = 'grace_period' AND status != 'deleted'`,
		"expired_tenants":       `SELECT COUNT(*) FROM tenants WHERE subscription_status = 'expired' AND status != 'deleted'`,
		"open_tickets":          `SELECT COUNT(*) FROM support_tickets WHERE status IN ('open', 'in_progress', 'waiting_on_tenant')`,
	}
	for key, query := range scalars {
		var value int64
		if err := s.db.QueryRowContext(ctx, query).Scan(&value); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to load platform metrics"})
		}
		overview[key] = value
	}

	var billingIncome, tenantSalesGMV int64
	if err := s.db.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(amount_idr), 0)
		FROM billing_invoices
		WHERE status = 'paid' AND paid_at BETWEEN $1 AND $2`, start, end).Scan(&billingIncome); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to load billing income"})
	}
	if err := s.db.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(total_amount), 0)
		FROM guest_orders
		WHERE status = 'COMPLETE' AND created_at BETWEEN $1 AND $2`, start, end).Scan(&tenantSalesGMV); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to load tenant sales GMV"})
	}
	overview["billing_income_idr"] = billingIncome
	overview["tenant_sales_gmv_idr"] = tenantSalesGMV

	topRows, err := s.db.QueryContext(ctx, `
		SELECT t.id, t.business_name, COALESCE(SUM(go.total_amount), 0) AS gmv, COUNT(go.id) AS orders
		FROM tenants t
		JOIN guest_orders go ON go.tenant_id = t.id AND go.status = 'COMPLETE' AND go.created_at BETWEEN $1 AND $2
		GROUP BY t.id, t.business_name
		ORDER BY gmv DESC
		LIMIT 10`, start, end)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to load top tenants"})
	}
	defer topRows.Close()
	topTenants := []map[string]interface{}{}
	for topRows.Next() {
		var id, name string
		var gmv, orders int64
		if err := topRows.Scan(&id, &name, &gmv, &orders); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to scan top tenants"})
		}
		topTenants = append(topTenants, map[string]interface{}{"tenant_id": id, "business_name": name, "tenant_sales_gmv_idr": gmv, "orders": orders})
	}
	overview["top_sales_tenants"] = topTenants

	ticketRows, err := s.db.QueryContext(ctx, `SELECT status, COUNT(*) FROM support_tickets GROUP BY status`)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to load ticket metrics"})
	}
	defer ticketRows.Close()
	ticketCounts := map[string]int64{}
	for ticketRows.Next() {
		var status string
		var count int64
		if err := ticketRows.Scan(&status, &count); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to scan ticket metrics"})
		}
		ticketCounts[status] = count
	}
	overview["tickets_by_status"] = ticketCounts

	return c.JSON(http.StatusOK, overview)
}

func (s *PlatformService) ListTenants(c echo.Context) error {
	limit := clampInt(c.QueryParam("limit"), 50, 1, 200)
	offset := clampInt(c.QueryParam("offset"), 0, 0, 1000000)
	status := c.QueryParam("status")
	subscriptionStatus := c.QueryParam("subscription_status")
	q := strings.TrimSpace(c.QueryParam("q"))

	rows, err := s.db.QueryContext(c.Request().Context(), `
		SELECT t.id, t.business_name, t.slug, t.status,
		       t.subscription_plan, t.billing_cycle, t.subscription_status,
		       t.trial_ends_at, t.subscription_ends_at,
		       t.storage_used_bytes, t.storage_quota_bytes,
		       COALESCE((SELECT COUNT(*) FROM users u WHERE u.tenant_id = t.id AND u.status != 'deleted'), 0) AS user_count,
		       COALESCE((SELECT SUM(amount_idr) FROM billing_invoices bi WHERE bi.tenant_id = t.id AND bi.status = 'paid'), 0) AS paid_invoice_total_idr,
		       COALESCE((SELECT COUNT(*) FROM support_tickets st WHERE st.tenant_id = t.id AND st.status IN ('open','in_progress','waiting_on_tenant')), 0) AS open_ticket_count,
		       t.created_at, t.updated_at,
		       t.suspended_at, t.deactivated_at, t.scheduled_delete_at, t.status_reason
		FROM tenants t
		WHERE ($1 = '' OR t.status = $1)
		  AND ($2 = '' OR t.subscription_status = $2)
		  AND ($3 = '' OR t.business_name ILIKE '%' || $3 || '%' OR t.slug ILIKE '%' || $3 || '%')
		ORDER BY t.created_at DESC
		LIMIT $4 OFFSET $5`, status, subscriptionStatus, q, limit, offset)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to list tenants"})
	}
	defer rows.Close()

	tenants := []TenantSummary{}
	for rows.Next() {
		tenant, err := scanTenantSummary(rows)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to scan tenant"})
		}
		tenants = append(tenants, tenant)
	}
	return c.JSON(http.StatusOK, map[string]interface{}{"tenants": tenants, "limit": limit, "offset": offset})
}

func (s *PlatformService) GetTenant(c echo.Context) error {
	tenant, err := s.getTenantSummary(c.Request().Context(), c.Param("tenant_id"))
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "tenant not found"})
	}
	activity, _ := s.loadTenantActivity(c.Request().Context(), tenant.ID, 20)
	return c.JSON(http.StatusOK, map[string]interface{}{"tenant": tenant, "activity": activity})
}

func (s *PlatformService) GetTenantActivity(c echo.Context) error {
	activity, err := s.loadTenantActivity(c.Request().Context(), c.Param("tenant_id"), 50)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to load activity"})
	}
	return c.JSON(http.StatusOK, map[string]interface{}{"activity": activity})
}

func (s *PlatformService) TenantAction(c echo.Context) error {
	adminID := c.Request().Header.Get("X-Platform-Admin-ID")
	if adminID == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "missing platform admin"})
	}
	tenantID := c.Param("tenant_id")
	action := c.Param("action")
	var req struct {
		Reason string `json:"reason"`
	}
	_ = c.Bind(&req)

	tx, err := s.db.BeginTx(c.Request().Context(), nil)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to start transaction"})
	}
	defer tx.Rollback()

	var beforeStatus string
	if err := tx.QueryRowContext(c.Request().Context(), `SELECT status FROM tenants WHERE id = $1 AND status != 'deleted' FOR UPDATE`, tenantID).Scan(&beforeStatus); err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "tenant not found"})
	}

	now := time.Now().UTC()
	afterStatus := beforeStatus
	eventAction := action
	var deleteAfter *time.Time
	var updateQuery string
	var args []interface{}
	switch action {
	case "suspend":
		afterStatus = "suspended"
		updateQuery = `UPDATE tenants SET status = $2, suspended_at = $3, status_reason = $4, updated_at = NOW() WHERE id = $1`
		args = []interface{}{tenantID, afterStatus, now, req.Reason}
	case "deactivate":
		afterStatus = "inactive"
		updateQuery = `UPDATE tenants SET status = $2, deactivated_at = $3, status_reason = $4, updated_at = NOW() WHERE id = $1`
		args = []interface{}{tenantID, afterStatus, now, req.Reason}
	case "delete":
		eventAction = "schedule_delete"
		afterStatus = "inactive"
		scheduled := now.AddDate(0, 0, getEnvInt("TENANT_DELETE_GRACE_DAYS", 30))
		deleteAfter = &scheduled
		updateQuery = `UPDATE tenants SET status = $2, deletion_requested_at = $3, scheduled_delete_at = $4, status_reason = $5, updated_at = NOW() WHERE id = $1`
		args = []interface{}{tenantID, afterStatus, now, scheduled, req.Reason}
	case "cancel-delete":
		eventAction = "cancel_delete"
		afterStatus = "active"
		updateQuery = `UPDATE tenants SET status = $2, deletion_requested_at = NULL, scheduled_delete_at = NULL, status_reason = NULL, updated_at = NOW() WHERE id = $1`
		args = []interface{}{tenantID, afterStatus}
	case "reactivate":
		afterStatus = "active"
		updateQuery = `UPDATE tenants SET status = $2, suspended_at = NULL, deactivated_at = NULL, deletion_requested_at = NULL, scheduled_delete_at = NULL, status_reason = NULL, updated_at = NOW() WHERE id = $1`
		args = []interface{}{tenantID, afterStatus}
	default:
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "unsupported tenant action"})
	}

	if _, err := tx.ExecContext(c.Request().Context(), updateQuery, args...); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to update tenant"})
	}
	if afterStatus != "active" {
		_, _ = tx.ExecContext(c.Request().Context(), `
			UPDATE sessions SET terminated_at = COALESCE(terminated_at, NOW()) WHERE tenant_id = $1 AND terminated_at IS NULL`, tenantID)
	}
	if _, err := tx.ExecContext(c.Request().Context(), `
		INSERT INTO tenant_lifecycle_events
		  (tenant_id, platform_admin_id, action, reason, before_status, after_status, effective_at, delete_after)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		tenantID, adminID, eventAction, nullString(req.Reason), beforeStatus, afterStatus, now, deleteAfter); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to write lifecycle event"})
	}
	if err := tx.Commit(); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to commit tenant action"})
	}
	return c.JSON(http.StatusOK, map[string]interface{}{"tenant_id": tenantID, "status": afterStatus, "delete_after": deleteAfter})
}

func (s *PlatformService) ListTickets(c echo.Context) error {
	status := c.QueryParam("status")
	tenantID := c.QueryParam("tenant_id")
	rows, err := s.db.QueryContext(c.Request().Context(), `
		SELECT st.id, st.tenant_id, t.business_name, st.subject, st.description,
		       st.category, st.priority, st.status, st.assigned_to_platform_admin_id,
		       st.created_at, st.updated_at, st.last_activity_at, st.resolved_at
		FROM support_tickets st
		JOIN tenants t ON t.id = st.tenant_id
		WHERE ($1 = '' OR st.status = $1)
		  AND ($2 = '' OR st.tenant_id = NULLIF($2, '')::uuid)
		ORDER BY st.last_activity_at DESC
		LIMIT 200`, status, tenantID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to list tickets"})
	}
	defer rows.Close()

	tickets := []Ticket{}
	for rows.Next() {
		t, err := scanTicket(rows)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to scan ticket"})
		}
		tickets = append(tickets, t)
	}
	return c.JSON(http.StatusOK, map[string]interface{}{"tickets": tickets})
}

func (s *PlatformService) CreateTicket(c echo.Context) error {
	tenantID := c.Request().Header.Get("X-Tenant-ID")
	if tenantID == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "tenant context required"})
	}
	var req struct {
		Subject     string `json:"subject"`
		Description string `json:"description"`
		Category    string `json:"category"`
		Priority    string `json:"priority"`
	}
	if err := c.Bind(&req); err != nil || strings.TrimSpace(req.Subject) == "" || strings.TrimSpace(req.Description) == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "subject and description are required"})
	}
	if req.Category == "" {
		req.Category = "billing"
	}
	if req.Priority == "" {
		req.Priority = "normal"
	}
	userID := c.Request().Header.Get("X-User-ID")
	email := c.Request().Header.Get("X-User-Email")

	var id string
	err := s.db.QueryRowContext(c.Request().Context(), `
		INSERT INTO support_tickets
		  (tenant_id, subject, description, category, priority, created_by_user_id, created_by_email)
		VALUES ($1, $2, $3, $4, $5, NULLIF($6, '')::uuid, NULLIF($7, ''))
		RETURNING id`,
		tenantID, req.Subject, req.Description, req.Category, req.Priority, userID, email,
	).Scan(&id)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to create ticket"})
	}
	_, _ = s.db.ExecContext(c.Request().Context(), `
		INSERT INTO support_ticket_notes (ticket_id, author_type, author_id, body)
		VALUES ($1, 'tenant_user', NULLIF($2, '')::uuid, $3)`,
		id, userID, req.Description)
	return c.JSON(http.StatusCreated, map[string]string{"id": id})
}

func (s *PlatformService) UpdateTicket(c echo.Context) error {
	var req struct {
		Status     string  `json:"status"`
		Priority   string  `json:"priority"`
		AssigneeID *string `json:"assigned_to_platform_admin_id"`
	}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}
	var resolvedAt interface{}
	if req.Status == "resolved" || req.Status == "closed" {
		resolvedAt = time.Now().UTC()
	}
	_, err := s.db.ExecContext(c.Request().Context(), `
		UPDATE support_tickets
		SET status = COALESCE(NULLIF($2, ''), status),
		    priority = COALESCE(NULLIF($3, ''), priority),
		    assigned_to_platform_admin_id = COALESCE($4, assigned_to_platform_admin_id),
		    resolved_at = COALESCE($5, resolved_at),
		    last_activity_at = NOW(),
		    updated_at = NOW()
		WHERE id = $1`,
		c.Param("ticket_id"), req.Status, req.Priority, req.AssigneeID, resolvedAt)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to update ticket"})
	}
	return c.JSON(http.StatusOK, map[string]string{"status": "updated"})
}

func (s *PlatformService) AddTicketNote(c echo.Context) error {
	adminID := c.Request().Header.Get("X-Platform-Admin-ID")
	if adminID == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "missing platform admin"})
	}
	var req struct {
		Body string `json:"body"`
	}
	if err := c.Bind(&req); err != nil || strings.TrimSpace(req.Body) == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "note body is required"})
	}
	tx, err := s.db.BeginTx(c.Request().Context(), nil)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to start transaction"})
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(c.Request().Context(), `
		INSERT INTO support_ticket_notes (ticket_id, author_type, author_id, body)
		VALUES ($1, 'platform_admin', $2, $3)`,
		c.Param("ticket_id"), adminID, req.Body); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to add note"})
	}
	if _, err := tx.ExecContext(c.Request().Context(), `
		UPDATE support_tickets SET last_activity_at = NOW(), updated_at = NOW() WHERE id = $1`,
		c.Param("ticket_id")); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to update ticket activity"})
	}
	if err := tx.Commit(); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to commit note"})
	}
	return c.JSON(http.StatusCreated, map[string]string{"status": "created"})
}

func (s *PlatformService) signToken(sessionID string, admin PlatformAdmin, expiresAt time.Time) (string, error) {
	claims := PlatformClaims{
		SessionID:       sessionID,
		PlatformAdminID: admin.ID,
		Email:           admin.Email,
		Role:            admin.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now().UTC()),
			Subject:   admin.ID,
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(s.jwtSecret)
}

func (s *PlatformService) getAdmin(ctx context.Context, id string) (*PlatformAdmin, error) {
	admin := &PlatformAdmin{}
	var lastLogin sql.NullTime
	err := s.db.QueryRowContext(ctx, `
		SELECT id, email, name, role, status, last_login_at
		FROM platform_admins
		WHERE id = $1 AND status = 'active'`, id).
		Scan(&admin.ID, &admin.Email, &admin.Name, &admin.Role, &admin.Status, &lastLogin)
	if err != nil {
		return nil, err
	}
	if lastLogin.Valid {
		admin.LastLogin = &lastLogin.Time
	}
	return admin, nil
}

func (s *PlatformService) getAdminForSession(ctx context.Context, adminID, sessionID string) (*PlatformAdmin, error) {
	admin := &PlatformAdmin{}
	var lastLogin sql.NullTime
	err := s.db.QueryRowContext(ctx, `
		SELECT pa.id, pa.email, pa.name, pa.role, pa.status, pa.last_login_at
		FROM platform_admins pa
		JOIN platform_admin_sessions pas ON pas.platform_admin_id = pa.id
		WHERE pa.id = $1
		  AND pas.session_id = $2
		  AND pa.status = 'active'
		  AND pas.terminated_at IS NULL
		  AND pas.expires_at > NOW()`,
		adminID, sessionID,
	).Scan(&admin.ID, &admin.Email, &admin.Name, &admin.Role, &admin.Status, &lastLogin)
	if err != nil {
		return nil, err
	}
	if lastLogin.Valid {
		admin.LastLogin = &lastLogin.Time
	}
	return admin, nil
}

func (s *PlatformService) getTenantSummary(ctx context.Context, tenantID string) (*TenantSummary, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT t.id, t.business_name, t.slug, t.status,
		       t.subscription_plan, t.billing_cycle, t.subscription_status,
		       t.trial_ends_at, t.subscription_ends_at,
		       t.storage_used_bytes, t.storage_quota_bytes,
		       COALESCE((SELECT COUNT(*) FROM users u WHERE u.tenant_id = t.id AND u.status != 'deleted'), 0),
		       COALESCE((SELECT SUM(amount_idr) FROM billing_invoices bi WHERE bi.tenant_id = t.id AND bi.status = 'paid'), 0),
		       COALESCE((SELECT COUNT(*) FROM support_tickets st WHERE st.tenant_id = t.id AND st.status IN ('open','in_progress','waiting_on_tenant')), 0),
		       t.created_at, t.updated_at,
		       t.suspended_at, t.deactivated_at, t.scheduled_delete_at, t.status_reason
		FROM tenants t
		WHERE t.id = $1`, tenantID)
	tenant, err := scanTenantSummary(row)
	return &tenant, err
}

func (s *PlatformService) loadTenantActivity(ctx context.Context, tenantID string, limit int) ([]ActivityItem, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT 'lifecycle' AS type, action, '' AS actor_type, 'tenant' AS resource,
		       reason, before_status, after_status, created_at, delete_after
		FROM tenant_lifecycle_events
		WHERE tenant_id = $1
		ORDER BY created_at DESC
		LIMIT $2`, tenantID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []ActivityItem{}
	for rows.Next() {
		var item ActivityItem
		var reason, before, after sql.NullString
		var deleteAfter sql.NullTime
		if err := rows.Scan(&item.Type, &item.Action, &item.ActorType, &item.Resource, &reason, &before, &after, &item.OccurredAt, &deleteAfter); err != nil {
			return nil, err
		}
		if reason.Valid {
			item.Reason = &reason.String
		}
		if before.Valid {
			item.Before = &before.String
		}
		if after.Valid {
			item.After = &after.String
		}
		if deleteAfter.Valid {
			item.DeleteAfter = &deleteAfter.Time
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func scanTenantSummary(row interface {
	Scan(dest ...interface{}) error
}) (TenantSummary, error) {
	var tenant TenantSummary
	var trialEndsAt, subscriptionEndsAt, suspendedAt, deactivatedAt, scheduledDeleteAt sql.NullTime
	var statusReason sql.NullString
	err := row.Scan(
		&tenant.ID, &tenant.BusinessName, &tenant.Slug, &tenant.Status,
		&tenant.SubscriptionPlan, &tenant.BillingCycle, &tenant.SubscriptionStatus,
		&trialEndsAt, &subscriptionEndsAt,
		&tenant.StorageUsedBytes, &tenant.StorageQuotaBytes,
		&tenant.UserCount, &tenant.PaidInvoiceTotalIDR, &tenant.OpenTicketCount,
		&tenant.CreatedAt, &tenant.UpdatedAt,
		&suspendedAt, &deactivatedAt, &scheduledDeleteAt, &statusReason,
	)
	if trialEndsAt.Valid {
		tenant.TrialEndsAt = &trialEndsAt.Time
	}
	if subscriptionEndsAt.Valid {
		tenant.SubscriptionEndsAt = &subscriptionEndsAt.Time
	}
	if suspendedAt.Valid {
		tenant.SuspendedAt = &suspendedAt.Time
	}
	if deactivatedAt.Valid {
		tenant.DeactivatedAt = &deactivatedAt.Time
	}
	if scheduledDeleteAt.Valid {
		tenant.ScheduledDeleteAt = &scheduledDeleteAt.Time
	}
	if statusReason.Valid {
		tenant.StatusReason = &statusReason.String
	}
	return tenant, err
}

func scanTicket(row interface {
	Scan(dest ...interface{}) error
}) (Ticket, error) {
	var ticket Ticket
	var assignee sql.NullString
	var resolvedAt sql.NullTime
	err := row.Scan(
		&ticket.ID, &ticket.TenantID, &ticket.TenantName, &ticket.Subject, &ticket.Description,
		&ticket.Category, &ticket.Priority, &ticket.Status, &assignee,
		&ticket.CreatedAt, &ticket.UpdatedAt, &ticket.LastActivityAt, &resolvedAt,
	)
	if assignee.Valid {
		ticket.AssigneeID = &assignee.String
	}
	if resolvedAt.Valid {
		ticket.ResolvedAt = &resolvedAt.Time
	}
	return ticket, err
}

func parseDateRange(c echo.Context) (time.Time, time.Time, error) {
	startStr := c.QueryParam("start_date")
	endStr := c.QueryParam("end_date")
	now := time.Now().UTC()
	if startStr == "" || endStr == "" {
		return time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC), now, nil
	}
	start, err := time.Parse("2006-01-02", startStr)
	if err != nil {
		return time.Time{}, time.Time{}, errors.New("invalid start_date, expected YYYY-MM-DD")
	}
	end, err := time.Parse("2006-01-02", endStr)
	if err != nil {
		return time.Time{}, time.Time{}, errors.New("invalid end_date, expected YYYY-MM-DD")
	}
	return start, end.Add(24*time.Hour - time.Nanosecond), nil
}

func nullString(value string) interface{} {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return value
}

func clampInt(value string, fallback, min, max int) int {
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	if parsed < min {
		return min
	}
	if parsed > max {
		return max
	}
	return parsed
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	value, err := strconv.Atoi(os.Getenv(key))
	if err != nil {
		return fallback
	}
	return value
}
