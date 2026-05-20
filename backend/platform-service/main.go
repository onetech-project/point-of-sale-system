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
	OwnerEmail          string     `json:"owner_email,omitempty"`
	SubscriptionPlan    string     `json:"subscription_plan"`
	BillingCycle        string     `json:"billing_cycle"`
	SubscriptionStatus  string     `json:"subscription_status"`
	TrialEndsAt         *time.Time `json:"trial_ends_at,omitempty"`
	SubscriptionEndsAt  *time.Time `json:"subscription_ends_at,omitempty"`
	ExpiresAt           *time.Time `json:"expires_at,omitempty"`
	StorageUsedBytes    int64      `json:"storage_used_bytes"`
	StorageQuotaBytes   int64      `json:"storage_quota_bytes"`
	UserCount           int64      `json:"user_count"`
	ActiveUserCount     int64      `json:"active_user_count"`
	PaidInvoiceTotalIDR int64      `json:"paid_invoice_total_idr"`
	OpenTicketCount     int64      `json:"open_ticket_count"`
	LastActiveAt        *time.Time `json:"last_active_at,omitempty"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
	SuspendedAt         *time.Time `json:"suspended_at,omitempty"`
	DeactivatedAt       *time.Time `json:"deactivated_at,omitempty"`
	ScheduledDeleteAt   *time.Time `json:"scheduled_delete_at,omitempty"`
	StatusReason        *string    `json:"status_reason,omitempty"`
}

type TenantHealthSummary struct {
	RegisteredTenants  int64 `json:"registered_tenants"`
	ActiveTenants      int64 `json:"active_tenants"`
	TrialTenants       int64 `json:"trial_tenants"`
	GracePeriodTenants int64 `json:"grace_period_tenants"`
	WillExpire7        int64 `json:"will_expire_7"`
	WillExpire14       int64 `json:"will_expire_14"`
	WillExpire30       int64 `json:"will_expire_30"`
	ExpiredTenants     int64 `json:"expired_tenants"`
	CancelledTenants   int64 `json:"cancelled_tenants"`
	SuspendedTenants   int64 `json:"suspended_tenants"`
	InactiveTenants    int64 `json:"inactive_tenants"`
	ScheduledDeletes   int64 `json:"scheduled_deletes"`
}

type RevenuePoint struct {
	Date          string `json:"date"`
	PaidIncomeIDR int64  `json:"paid_income_idr"`
	InvoiceCount  int64  `json:"invoice_count"`
}

type TopPayingTenant struct {
	TenantID      string `json:"tenant_id"`
	BusinessName  string `json:"business_name"`
	PaidIncomeIDR int64  `json:"paid_income_idr"`
	InvoiceCount  int64  `json:"invoice_count"`
}

type RevenueSummary struct {
	StartDate             string            `json:"start_date"`
	EndDate               string            `json:"end_date"`
	PaidIncomeIDR         int64             `json:"paid_income_idr"`
	PreviousPaidIncomeIDR int64             `json:"previous_paid_income_idr"`
	PeriodDeltaPercent    *float64          `json:"period_delta_percent,omitempty"`
	PendingIncomeIDR      int64             `json:"pending_income_idr"`
	ExpiredIncomeIDR      int64             `json:"expired_income_idr"`
	OverdueIncomeIDR      int64             `json:"overdue_income_idr"`
	InvoiceCountByStatus  map[string]int64  `json:"invoice_count_by_status"`
	BillingCycleMix       map[string]int64  `json:"billing_cycle_mix"`
	Timeseries            []RevenuePoint    `json:"revenue_timeseries"`
	TopPayingTenants      []TopPayingTenant `json:"top_paying_tenants"`
	OverdueInvoices       []TenantInvoice   `json:"overdue_invoices"`
}

type UrgentItem struct {
	Type         string     `json:"type"`
	TenantID     string     `json:"tenant_id"`
	BusinessName string     `json:"business_name"`
	Label        string     `json:"label"`
	Severity     string     `json:"severity"`
	DueAt        *time.Time `json:"due_at,omitempty"`
	AmountIDR    int64      `json:"amount_idr,omitempty"`
}

type TenantInvoice struct {
	ID                   string     `json:"id"`
	TenantID             string     `json:"tenant_id"`
	TenantName           string     `json:"tenant_name,omitempty"`
	InvoiceNumber        string     `json:"invoice_number"`
	AmountIDR            int64      `json:"amount_idr"`
	BillingInterval      string     `json:"billing_interval"`
	PeriodStart          time.Time  `json:"period_start"`
	PeriodEnd            time.Time  `json:"period_end"`
	DueAt                time.Time  `json:"due_at"`
	Status               string     `json:"status"`
	PaidAt               *time.Time `json:"paid_at,omitempty"`
	MidtransOrderID      *string    `json:"midtrans_order_id,omitempty"`
	PaymentURL           *string    `json:"payment_url,omitempty"`
	PaymentLinkExpiresAt *time.Time `json:"payment_link_expires_at,omitempty"`
	CreatedAt            time.Time  `json:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at"`
}

type PaymentAttempt struct {
	ID              string    `json:"id"`
	InvoiceID       string    `json:"invoice_id"`
	TenantID        string    `json:"tenant_id"`
	AmountIDR       int64     `json:"amount_idr"`
	MidtransOrderID *string   `json:"midtrans_order_id,omitempty"`
	PaymentMethod   *string   `json:"payment_method,omitempty"`
	Status          string    `json:"status"`
	ErrorMsg        *string   `json:"error_msg,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
}

type TenantBillingDetail struct {
	Invoices        []TenantInvoice  `json:"invoices"`
	PaymentAttempts []PaymentAttempt `json:"payment_attempts"`
}

type TenantNote struct {
	ID        string    `json:"id"`
	TenantID  string    `json:"tenant_id"`
	AdminID   *string   `json:"platform_admin_id,omitempty"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
}

type PlatformTenantDetail struct {
	Tenant   *TenantSummary      `json:"tenant"`
	Billing  TenantBillingDetail `json:"billing"`
	Activity []ActivityItem      `json:"activity"`
	Tickets  []Ticket            `json:"tickets"`
	Notes    []TenantNote        `json:"notes"`
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
	platform.GET("/analytics/revenue", svc.AnalyticsRevenue)
	platform.GET("/analytics/tenant-health", svc.TenantHealth)
	platform.GET("/tenants", svc.ListTenants)
	platform.GET("/tenants/:tenant_id", svc.GetTenant)
	platform.GET("/tenants/:tenant_id/activity", svc.GetTenantActivity)
	platform.GET("/tenants/:tenant_id/billing", svc.GetTenantBilling)
	platform.POST("/tenants/:tenant_id/actions/:action", svc.TenantAction)
	platform.GET("/audit-events", svc.ListAuditEvents)
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
		"registered_tenants":    `SELECT COUNT(*) FROM tenants WHERE status != 'deleted'`,
		"active_tenants":        `SELECT COUNT(*) FROM tenants WHERE status = 'active'`,
		"active_users":          `SELECT COUNT(*) FROM users WHERE status = 'active'`,
		"recently_active_users": `SELECT COUNT(*) FROM users WHERE status = 'active' AND last_login_at >= NOW() - INTERVAL '7 days'`,
		"open_tickets":          `SELECT COUNT(*) FROM support_tickets WHERE status IN ('open', 'in_progress', 'waiting_on_tenant')`,
		"high_priority_tickets": `SELECT COUNT(*) FROM support_tickets WHERE priority IN ('high', 'urgent') AND status IN ('open', 'in_progress', 'waiting_on_tenant')`,
		"storage_used_bytes":    `SELECT COALESCE(SUM(storage_used_bytes), 0) FROM tenants WHERE status != 'deleted'`,
		"storage_quota_bytes":   `SELECT COALESCE(SUM(storage_quota_bytes), 0) FROM tenants WHERE status != 'deleted'`,
	}
	for key, query := range scalars {
		var value int64
		if err := s.db.QueryRowContext(ctx, query).Scan(&value); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to load platform metrics"})
		}
		overview[key] = value
	}

	revenue, err := s.loadRevenueSummary(ctx, start, end)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to load revenue metrics"})
	}
	health, err := s.loadTenantHealth(ctx)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to load tenant health"})
	}
	urgent, err := s.loadUrgentItems(ctx)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to load urgent items"})
	}

	var tenantSalesGMV int64
	if err := s.db.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(total_amount), 0)
		FROM guest_orders
		WHERE status = 'COMPLETE' AND created_at BETWEEN $1 AND $2`, start, end).Scan(&tenantSalesGMV); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to load tenant sales GMV"})
	}
	overview["billing_income_idr"] = revenue.PaidIncomeIDR
	overview["paid_income_idr"] = revenue.PaidIncomeIDR
	overview["pending_income_idr"] = revenue.PendingIncomeIDR
	overview["expired_income_idr"] = revenue.ExpiredIncomeIDR
	overview["overdue_income_idr"] = revenue.OverdueIncomeIDR
	overview["previous_paid_income_idr"] = revenue.PreviousPaidIncomeIDR
	if revenue.PeriodDeltaPercent != nil {
		overview["period_delta_percent"] = revenue.PeriodDeltaPercent
	}
	overview["invoice_count_by_status"] = revenue.InvoiceCountByStatus
	overview["billing_cycle_mix"] = revenue.BillingCycleMix
	overview["revenue_timeseries"] = revenue.Timeseries
	overview["top_paying_tenants"] = revenue.TopPayingTenants
	overview["tenant_health"] = health
	overview["grace_period_tenants"] = health.GracePeriodTenants
	overview["expired_tenants"] = health.ExpiredTenants
	overview["trial_tenants"] = health.TrialTenants
	overview["will_expire_7"] = health.WillExpire7
	overview["will_expire_14"] = health.WillExpire14
	overview["will_expire_30"] = health.WillExpire30
	overview["urgent_items"] = urgent
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

func (s *PlatformService) AnalyticsRevenue(c echo.Context) error {
	start, end, err := parseDateRange(c)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	revenue, err := s.loadRevenueSummary(c.Request().Context(), start, end)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to load revenue metrics"})
	}
	return c.JSON(http.StatusOK, revenue)
}

func (s *PlatformService) TenantHealth(c echo.Context) error {
	health, err := s.loadTenantHealth(c.Request().Context())
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to load tenant health"})
	}
	return c.JSON(http.StatusOK, health)
}

func (s *PlatformService) loadTenantHealth(ctx context.Context) (*TenantHealthSummary, error) {
	var health TenantHealthSummary
	err := s.db.QueryRowContext(ctx, `
		SELECT
		  COUNT(*) FILTER (WHERE status != 'deleted') AS registered_tenants,
		  COUNT(*) FILTER (WHERE status = 'active') AS active_tenants,
		  COUNT(*) FILTER (WHERE subscription_status = 'trial' AND status != 'deleted') AS trial_tenants,
		  COUNT(*) FILTER (WHERE subscription_status = 'grace_period' AND status != 'deleted') AS grace_period_tenants,
		  COUNT(*) FILTER (
		    WHERE status != 'deleted'
		      AND subscription_status IN ('trial', 'active', 'grace_period')
		      AND COALESCE(
		        CASE WHEN subscription_plan = 'trial' THEN trial_ends_at ELSE subscription_ends_at END,
		        trial_ends_at,
		        subscription_ends_at
		      ) BETWEEN NOW() AND NOW() + INTERVAL '7 days'
		  ) AS will_expire_7,
		  COUNT(*) FILTER (
		    WHERE status != 'deleted'
		      AND subscription_status IN ('trial', 'active', 'grace_period')
		      AND COALESCE(
		        CASE WHEN subscription_plan = 'trial' THEN trial_ends_at ELSE subscription_ends_at END,
		        trial_ends_at,
		        subscription_ends_at
		      ) BETWEEN NOW() AND NOW() + INTERVAL '14 days'
		  ) AS will_expire_14,
		  COUNT(*) FILTER (
		    WHERE status != 'deleted'
		      AND subscription_status IN ('trial', 'active', 'grace_period')
		      AND COALESCE(
		        CASE WHEN subscription_plan = 'trial' THEN trial_ends_at ELSE subscription_ends_at END,
		        trial_ends_at,
		        subscription_ends_at
		      ) BETWEEN NOW() AND NOW() + INTERVAL '30 days'
		  ) AS will_expire_30,
		  COUNT(*) FILTER (WHERE subscription_status = 'expired' AND status != 'deleted') AS expired_tenants,
		  COUNT(*) FILTER (WHERE subscription_status = 'cancelled' AND status != 'deleted') AS cancelled_tenants,
		  COUNT(*) FILTER (WHERE status = 'suspended') AS suspended_tenants,
		  COUNT(*) FILTER (WHERE status = 'inactive') AS inactive_tenants,
		  COUNT(*) FILTER (WHERE scheduled_delete_at IS NOT NULL AND status != 'deleted') AS scheduled_deletes
		FROM tenants`,
	).Scan(
		&health.RegisteredTenants,
		&health.ActiveTenants,
		&health.TrialTenants,
		&health.GracePeriodTenants,
		&health.WillExpire7,
		&health.WillExpire14,
		&health.WillExpire30,
		&health.ExpiredTenants,
		&health.CancelledTenants,
		&health.SuspendedTenants,
		&health.InactiveTenants,
		&health.ScheduledDeletes,
	)
	return &health, err
}

func (s *PlatformService) loadRevenueSummary(ctx context.Context, start, end time.Time) (*RevenueSummary, error) {
	summary := &RevenueSummary{
		StartDate:            start.Format("2006-01-02"),
		EndDate:              end.Format("2006-01-02"),
		InvoiceCountByStatus: map[string]int64{},
		BillingCycleMix:      map[string]int64{},
		Timeseries:           []RevenuePoint{},
		TopPayingTenants:     []TopPayingTenant{},
		OverdueInvoices:      []TenantInvoice{},
	}

	if err := s.db.QueryRowContext(ctx, `
		SELECT
		  COALESCE(SUM(amount_idr) FILTER (WHERE status = 'paid' AND paid_at BETWEEN $1 AND $2), 0),
		  COALESCE(SUM(amount_idr) FILTER (WHERE status = 'pending' AND created_at BETWEEN $1 AND $2), 0),
		  COALESCE(SUM(amount_idr) FILTER (WHERE status = 'expired' AND created_at BETWEEN $1 AND $2), 0),
		  COALESCE(SUM(amount_idr) FILTER (WHERE status = 'pending' AND due_at < NOW()), 0)
		FROM billing_invoices`,
		start, end,
	).Scan(&summary.PaidIncomeIDR, &summary.PendingIncomeIDR, &summary.ExpiredIncomeIDR, &summary.OverdueIncomeIDR); err != nil {
		return nil, err
	}

	period := end.Sub(start)
	previousStart := start.Add(-period)
	if err := s.db.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(amount_idr), 0)
		FROM billing_invoices
		WHERE status = 'paid' AND paid_at >= $1 AND paid_at < $2`,
		previousStart, start,
	).Scan(&summary.PreviousPaidIncomeIDR); err != nil {
		return nil, err
	}
	if summary.PreviousPaidIncomeIDR > 0 {
		delta := (float64(summary.PaidIncomeIDR-summary.PreviousPaidIncomeIDR) / float64(summary.PreviousPaidIncomeIDR)) * 100
		summary.PeriodDeltaPercent = &delta
	}

	statusRows, err := s.db.QueryContext(ctx, `
		SELECT status, COUNT(*)
		FROM billing_invoices
		WHERE created_at BETWEEN $1 AND $2 OR paid_at BETWEEN $1 AND $2
		GROUP BY status`, start, end)
	if err != nil {
		return nil, err
	}
	defer statusRows.Close()
	for statusRows.Next() {
		var status string
		var count int64
		if err := statusRows.Scan(&status, &count); err != nil {
			return nil, err
		}
		summary.InvoiceCountByStatus[status] = count
	}
	if err := statusRows.Err(); err != nil {
		return nil, err
	}

	cycleRows, err := s.db.QueryContext(ctx, `
		SELECT billing_interval, COUNT(*)
		FROM billing_invoices
		WHERE created_at BETWEEN $1 AND $2 OR paid_at BETWEEN $1 AND $2
		GROUP BY billing_interval`, start, end)
	if err != nil {
		return nil, err
	}
	defer cycleRows.Close()
	for cycleRows.Next() {
		var cycle string
		var count int64
		if err := cycleRows.Scan(&cycle, &count); err != nil {
			return nil, err
		}
		summary.BillingCycleMix[cycle] = count
	}
	if err := cycleRows.Err(); err != nil {
		return nil, err
	}

	timeRows, err := s.db.QueryContext(ctx, `
		SELECT DATE_TRUNC('day', paid_at) AS day, COALESCE(SUM(amount_idr), 0), COUNT(*)
		FROM billing_invoices
		WHERE status = 'paid' AND paid_at BETWEEN $1 AND $2
		GROUP BY day
		ORDER BY day ASC`, start, end)
	if err != nil {
		return nil, err
	}
	defer timeRows.Close()
	for timeRows.Next() {
		var day time.Time
		var value, count int64
		if err := timeRows.Scan(&day, &value, &count); err != nil {
			return nil, err
		}
		summary.Timeseries = append(summary.Timeseries, RevenuePoint{
			Date:          day.Format("2006-01-02"),
			PaidIncomeIDR: value,
			InvoiceCount:  count,
		})
	}
	if err := timeRows.Err(); err != nil {
		return nil, err
	}

	topRows, err := s.db.QueryContext(ctx, `
		SELECT t.id, t.business_name, COALESCE(SUM(bi.amount_idr), 0), COUNT(bi.id)
		FROM billing_invoices bi
		JOIN tenants t ON t.id = bi.tenant_id
		WHERE bi.status = 'paid' AND bi.paid_at BETWEEN $1 AND $2
		GROUP BY t.id, t.business_name
		ORDER BY COALESCE(SUM(bi.amount_idr), 0) DESC
		LIMIT 10`, start, end)
	if err != nil {
		return nil, err
	}
	defer topRows.Close()
	for topRows.Next() {
		var item TopPayingTenant
		if err := topRows.Scan(&item.TenantID, &item.BusinessName, &item.PaidIncomeIDR, &item.InvoiceCount); err != nil {
			return nil, err
		}
		summary.TopPayingTenants = append(summary.TopPayingTenants, item)
	}
	if err := topRows.Err(); err != nil {
		return nil, err
	}

	overdueRows, err := s.db.QueryContext(ctx, tenantInvoiceSelect(`
		WHERE bi.status = 'pending' AND bi.due_at < NOW()
		ORDER BY bi.due_at ASC
		LIMIT 20`))
	if err != nil {
		return nil, err
	}
	defer overdueRows.Close()
	for overdueRows.Next() {
		inv, err := scanTenantInvoice(overdueRows)
		if err != nil {
			return nil, err
		}
		summary.OverdueInvoices = append(summary.OverdueInvoices, inv)
	}
	return summary, overdueRows.Err()
}

func (s *PlatformService) loadUrgentItems(ctx context.Context) ([]UrgentItem, error) {
	items := []UrgentItem{}

	expiredRows, err := s.db.QueryContext(ctx, `
		SELECT id, business_name,
		       COALESCE(CASE WHEN subscription_plan = 'trial' THEN trial_ends_at ELSE subscription_ends_at END, trial_ends_at, subscription_ends_at)
		FROM tenants
		WHERE status != 'deleted' AND subscription_status = 'expired'
		ORDER BY updated_at DESC
		LIMIT 8`)
	if err != nil {
		return nil, err
	}
	defer expiredRows.Close()
	for expiredRows.Next() {
		var item UrgentItem
		var due sql.NullTime
		if err := expiredRows.Scan(&item.TenantID, &item.BusinessName, &due); err != nil {
			return nil, err
		}
		item.Type = "expired_subscription"
		item.Label = "Subscription expired"
		item.Severity = "danger"
		if due.Valid {
			item.DueAt = &due.Time
		}
		items = append(items, item)
	}
	if err := expiredRows.Err(); err != nil {
		return nil, err
	}

	graceRows, err := s.db.QueryContext(ctx, `
		SELECT id, business_name,
		       COALESCE(CASE WHEN subscription_plan = 'trial' THEN trial_ends_at ELSE subscription_ends_at END, trial_ends_at, subscription_ends_at)
		FROM tenants
		WHERE status != 'deleted'
		  AND subscription_status = 'grace_period'
		  AND COALESCE(CASE WHEN subscription_plan = 'trial' THEN trial_ends_at ELSE subscription_ends_at END, trial_ends_at, subscription_ends_at)
		      <= NOW() + INTERVAL '7 days'
		ORDER BY COALESCE(CASE WHEN subscription_plan = 'trial' THEN trial_ends_at ELSE subscription_ends_at END, trial_ends_at, subscription_ends_at) ASC
		LIMIT 8`)
	if err != nil {
		return nil, err
	}
	defer graceRows.Close()
	for graceRows.Next() {
		var item UrgentItem
		var due sql.NullTime
		if err := graceRows.Scan(&item.TenantID, &item.BusinessName, &due); err != nil {
			return nil, err
		}
		item.Type = "grace_ending"
		item.Label = "Grace period ending"
		item.Severity = "warning"
		if due.Valid {
			item.DueAt = &due.Time
		}
		items = append(items, item)
	}
	if err := graceRows.Err(); err != nil {
		return nil, err
	}

	invoiceRows, err := s.db.QueryContext(ctx, `
		SELECT bi.tenant_id, t.business_name, bi.invoice_number, bi.amount_idr, bi.due_at
		FROM billing_invoices bi
		JOIN tenants t ON t.id = bi.tenant_id
		WHERE bi.status = 'pending' AND bi.due_at < NOW()
		ORDER BY bi.due_at ASC
		LIMIT 8`)
	if err != nil {
		return nil, err
	}
	defer invoiceRows.Close()
	for invoiceRows.Next() {
		var item UrgentItem
		var invoiceNumber string
		var dueAt time.Time
		if err := invoiceRows.Scan(&item.TenantID, &item.BusinessName, &invoiceNumber, &item.AmountIDR, &dueAt); err != nil {
			return nil, err
		}
		item.Type = "overdue_invoice"
		item.Label = invoiceNumber
		item.Severity = "danger"
		item.DueAt = &dueAt
		items = append(items, item)
	}
	return items, invoiceRows.Err()
}

func (s *PlatformService) ListTenants(c echo.Context) error {
	limit := clampInt(c.QueryParam("limit"), 50, 1, 200)
	offset := clampInt(c.QueryParam("offset"), 0, 0, 1000000)
	status := c.QueryParam("status")
	subscriptionStatus := c.QueryParam("subscription_status")
	plan := c.QueryParam("plan")
	expiresWithinDays := clampInt(c.QueryParam("expires_within_days"), 0, 0, 365)
	sortBy := tenantSortClause(c.QueryParam("sort"))
	q := strings.TrimSpace(c.QueryParam("q"))

	rows, err := s.db.QueryContext(c.Request().Context(), `
		SELECT t.id, t.business_name, t.slug, t.status,
		       COALESCE((
		         SELECT u.email FROM users u
		         WHERE u.tenant_id = t.id AND u.role = 'owner' AND u.status != 'deleted'
		         ORDER BY u.created_at ASC
		         LIMIT 1
		       ), '') AS owner_email,
		       t.subscription_plan, t.billing_cycle, t.subscription_status,
		       t.trial_ends_at, t.subscription_ends_at,
		       COALESCE(CASE WHEN t.subscription_plan = 'trial' THEN t.trial_ends_at ELSE t.subscription_ends_at END, t.trial_ends_at, t.subscription_ends_at) AS expires_at,
		       t.storage_used_bytes, t.storage_quota_bytes,
		       COALESCE((SELECT COUNT(*) FROM users u WHERE u.tenant_id = t.id AND u.status != 'deleted'), 0) AS user_count,
		       COALESCE((SELECT COUNT(*) FROM users u WHERE u.tenant_id = t.id AND u.status = 'active'), 0) AS active_user_count,
		       COALESCE((SELECT SUM(amount_idr) FROM billing_invoices bi WHERE bi.tenant_id = t.id AND bi.status = 'paid'), 0) AS paid_invoice_total_idr,
		       COALESCE((SELECT COUNT(*) FROM support_tickets st WHERE st.tenant_id = t.id AND st.status IN ('open','in_progress','waiting_on_tenant')), 0) AS open_ticket_count,
		       (SELECT MAX(u.last_login_at) FROM users u WHERE u.tenant_id = t.id) AS last_active_at,
		       t.created_at, t.updated_at,
		       t.suspended_at, t.deactivated_at, t.scheduled_delete_at, t.status_reason
		FROM tenants t
		WHERE ($1 = '' OR t.status = $1)
		  AND ($2 = '' OR t.subscription_status = $2)
		  AND ($3 = '' OR t.subscription_plan = $3)
		  AND ($4 = 0 OR COALESCE(CASE WHEN t.subscription_plan = 'trial' THEN t.trial_ends_at ELSE t.subscription_ends_at END, t.trial_ends_at, t.subscription_ends_at) BETWEEN NOW() AND NOW() + ($4 * INTERVAL '1 day'))
		  AND ($5 = '' OR t.business_name ILIKE '%' || $5 || '%' OR t.slug ILIKE '%' || $5 || '%')
		ORDER BY `+sortBy+`
		LIMIT $6 OFFSET $7`, status, subscriptionStatus, plan, expiresWithinDays, q, limit, offset)
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
	var total int64
	if err := s.db.QueryRowContext(c.Request().Context(), `
		SELECT COUNT(*)
		FROM tenants t
		WHERE ($1 = '' OR t.status = $1)
		  AND ($2 = '' OR t.subscription_status = $2)
		  AND ($3 = '' OR t.subscription_plan = $3)
		  AND ($4 = 0 OR COALESCE(CASE WHEN t.subscription_plan = 'trial' THEN t.trial_ends_at ELSE t.subscription_ends_at END, t.trial_ends_at, t.subscription_ends_at) BETWEEN NOW() AND NOW() + ($4 * INTERVAL '1 day'))
		  AND ($5 = '' OR t.business_name ILIKE '%' || $5 || '%' OR t.slug ILIKE '%' || $5 || '%')`,
		status, subscriptionStatus, plan, expiresWithinDays, q).Scan(&total); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to count tenants"})
	}
	return c.JSON(http.StatusOK, map[string]interface{}{"tenants": tenants, "limit": limit, "offset": offset, "total": total})
}

func (s *PlatformService) GetTenant(c echo.Context) error {
	tenant, err := s.getTenantSummary(c.Request().Context(), c.Param("tenant_id"))
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "tenant not found"})
	}
	activity := []ActivityItem{}
	if loaded, err := s.loadTenantActivity(c.Request().Context(), tenant.ID, 20); err == nil && loaded != nil {
		activity = loaded
	}
	billing := TenantBillingDetail{
		Invoices:        []TenantInvoice{},
		PaymentAttempts: []PaymentAttempt{},
	}
	if loaded, err := s.loadTenantBilling(c.Request().Context(), tenant.ID); err == nil {
		billing = loaded
		if billing.Invoices == nil {
			billing.Invoices = []TenantInvoice{}
		}
		if billing.PaymentAttempts == nil {
			billing.PaymentAttempts = []PaymentAttempt{}
		}
	}
	tickets := []Ticket{}
	if loaded, err := s.loadTenantTickets(c.Request().Context(), tenant.ID, ""); err == nil && loaded != nil {
		tickets = loaded
	}
	notes := []TenantNote{}
	if loaded, err := s.loadTenantNotes(c.Request().Context(), tenant.ID, 20); err == nil && loaded != nil {
		notes = loaded
	}
	return c.JSON(http.StatusOK, PlatformTenantDetail{
		Tenant:   tenant,
		Billing:  billing,
		Activity: activity,
		Tickets:  tickets,
		Notes:    notes,
	})
}

func (s *PlatformService) GetTenantActivity(c echo.Context) error {
	activity, err := s.loadTenantActivity(c.Request().Context(), c.Param("tenant_id"), 50)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to load activity"})
	}
	return c.JSON(http.StatusOK, map[string]interface{}{"activity": activity})
}

func (s *PlatformService) GetTenantBilling(c echo.Context) error {
	billing, err := s.loadTenantBilling(c.Request().Context(), c.Param("tenant_id"))
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to load tenant billing"})
	}
	return c.JSON(http.StatusOK, billing)
}

func (s *PlatformService) TenantAction(c echo.Context) error {
	adminID := c.Request().Header.Get("X-Platform-Admin-ID")
	if adminID == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "missing platform admin"})
	}
	tenantID := c.Param("tenant_id")
	action := c.Param("action")
	var req struct {
		Reason           string  `json:"reason"`
		SubscriptionPlan string  `json:"subscription_plan"`
		BillingCycle     string  `json:"billing_cycle"`
		ExtendDays       int     `json:"extend_days"`
		Note             string  `json:"note"`
		TicketID         string  `json:"ticket_id"`
		AssigneeID       *string `json:"assigned_to_platform_admin_id"`
	}
	_ = c.Bind(&req)
	reason := strings.TrimSpace(req.Reason)
	if reason == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "reason is required"})
	}

	tx, err := s.db.BeginTx(c.Request().Context(), nil)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to start transaction"})
	}
	defer tx.Rollback()

	var beforeStatus, beforePlan, beforeBillingCycle, beforeSubscriptionStatus string
	var trialEndsAt, subscriptionEndsAt sql.NullTime
	if err := tx.QueryRowContext(c.Request().Context(), `
		SELECT status, subscription_plan, billing_cycle, subscription_status, trial_ends_at, subscription_ends_at
		FROM tenants
		WHERE id = $1 AND status != 'deleted'
		FOR UPDATE`, tenantID).Scan(&beforeStatus, &beforePlan, &beforeBillingCycle, &beforeSubscriptionStatus, &trialEndsAt, &subscriptionEndsAt); err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "tenant not found"})
	}

	now := time.Now().UTC()
	afterStatus := beforeStatus
	beforeEventValue := beforeStatus
	afterEventValue := afterStatus
	eventAction := action
	var deleteAfter *time.Time
	result := map[string]interface{}{"tenant_id": tenantID}
	switch action {
	case "suspend":
		afterStatus = "suspended"
		if _, err := tx.ExecContext(c.Request().Context(), `UPDATE tenants SET status = $2, suspended_at = $3, status_reason = $4, updated_at = NOW() WHERE id = $1`, tenantID, afterStatus, now, reason); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to update tenant"})
		}
	case "deactivate":
		afterStatus = "inactive"
		if _, err := tx.ExecContext(c.Request().Context(), `UPDATE tenants SET status = $2, deactivated_at = $3, status_reason = $4, updated_at = NOW() WHERE id = $1`, tenantID, afterStatus, now, reason); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to update tenant"})
		}
	case "delete":
		eventAction = "schedule_delete"
		afterStatus = "inactive"
		scheduled := now.AddDate(0, 0, getEnvInt("TENANT_DELETE_GRACE_DAYS", 30))
		deleteAfter = &scheduled
		if _, err := tx.ExecContext(c.Request().Context(), `UPDATE tenants SET status = $2, deletion_requested_at = $3, scheduled_delete_at = $4, status_reason = $5, updated_at = NOW() WHERE id = $1`, tenantID, afterStatus, now, scheduled, reason); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to update tenant"})
		}
		result["delete_after"] = deleteAfter
	case "cancel-delete":
		eventAction = "cancel_delete"
		afterStatus = "active"
		if _, err := tx.ExecContext(c.Request().Context(), `UPDATE tenants SET status = $2, deletion_requested_at = NULL, scheduled_delete_at = NULL, status_reason = NULL, updated_at = NOW() WHERE id = $1`, tenantID, afterStatus); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to update tenant"})
		}
	case "reactivate":
		afterStatus = "active"
		if _, err := tx.ExecContext(c.Request().Context(), `UPDATE tenants SET status = $2, suspended_at = NULL, deactivated_at = NULL, deletion_requested_at = NULL, scheduled_delete_at = NULL, status_reason = NULL, updated_at = NOW() WHERE id = $1`, tenantID, afterStatus); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to update tenant"})
		}
	case "change-plan":
		eventAction = "change_plan"
		plan := strings.TrimSpace(req.SubscriptionPlan)
		if !validSubscriptionPlan(plan) {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "unsupported subscription_plan"})
		}
		beforeEventValue = beforePlan
		afterEventValue = plan
		if _, err := tx.ExecContext(c.Request().Context(), `
			UPDATE tenants
			SET subscription_plan = $2,
			    subscription_status = CASE
			      WHEN $2 = 'trial' THEN 'trial'
			      WHEN subscription_ends_at > NOW() THEN 'active'
			      ELSE subscription_status
			    END,
			    updated_at = NOW()
			WHERE id = $1`, tenantID, plan); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to update tenant plan"})
		}
		result["subscription_plan"] = plan
	case "change-billing-cycle":
		eventAction = "change_billing_cycle"
		cycle := strings.TrimSpace(req.BillingCycle)
		if !validBillingCycle(cycle) {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "unsupported billing_cycle"})
		}
		beforeEventValue = beforeBillingCycle
		afterEventValue = cycle
		if _, err := tx.ExecContext(c.Request().Context(), `UPDATE tenants SET billing_cycle = $2, updated_at = NOW() WHERE id = $1`, tenantID, cycle); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to update billing cycle"})
		}
		result["billing_cycle"] = cycle
	case "extend-trial":
		eventAction = "extend_trial"
		if beforePlan != "trial" {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "extend-trial is only available for trial tenants"})
		}
		if req.ExtendDays < 1 || req.ExtendDays > 365 {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "extend_days must be between 1 and 365"})
		}
		beforeEventValue = nullableTimeString(trialEndsAt)
		var newEndsAt time.Time
		if err := tx.QueryRowContext(c.Request().Context(), `
			UPDATE tenants
			SET trial_ends_at = GREATEST(COALESCE(trial_ends_at, NOW()), NOW()) + ($2 * INTERVAL '1 day'),
			    subscription_status = 'trial',
			    updated_at = NOW()
			WHERE id = $1
			RETURNING trial_ends_at`, tenantID, req.ExtendDays).Scan(&newEndsAt); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to extend trial"})
		}
		afterEventValue = newEndsAt.Format(time.RFC3339)
		result["trial_ends_at"] = newEndsAt
	case "extend-grace":
		eventAction = "extend_grace"
		if req.ExtendDays < 1 || req.ExtendDays > 365 {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "extend_days must be between 1 and 365"})
		}
		if beforePlan == "trial" {
			beforeEventValue = nullableTimeString(trialEndsAt)
			var newEndsAt time.Time
			if err := tx.QueryRowContext(c.Request().Context(), `
				UPDATE tenants
				SET trial_ends_at = GREATEST(COALESCE(trial_ends_at, NOW()), NOW()) + ($2 * INTERVAL '1 day'),
				    subscription_status = 'grace_period',
				    updated_at = NOW()
				WHERE id = $1
				RETURNING trial_ends_at`, tenantID, req.ExtendDays).Scan(&newEndsAt); err != nil {
				return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to extend grace period"})
			}
			afterEventValue = newEndsAt.Format(time.RFC3339)
			result["trial_ends_at"] = newEndsAt
		} else {
			beforeEventValue = nullableTimeString(subscriptionEndsAt)
			var newEndsAt time.Time
			if err := tx.QueryRowContext(c.Request().Context(), `
				UPDATE tenants
				SET subscription_ends_at = GREATEST(COALESCE(subscription_ends_at, NOW()), NOW()) + ($2 * INTERVAL '1 day'),
				    subscription_status = 'grace_period',
				    updated_at = NOW()
				WHERE id = $1
				RETURNING subscription_ends_at`, tenantID, req.ExtendDays).Scan(&newEndsAt); err != nil {
				return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to extend grace period"})
			}
			afterEventValue = newEndsAt.Format(time.RFC3339)
			result["subscription_ends_at"] = newEndsAt
		}
	case "regenerate-payment-link":
		eventAction = "regenerate_payment_link"
		expiresAt := now.Add(24 * time.Hour)
		var invoiceNumber string
		err := tx.QueryRowContext(c.Request().Context(), `
			UPDATE billing_invoices
			SET payment_link_expires_at = $2, updated_at = NOW()
			WHERE id = (
			  SELECT id FROM billing_invoices
			  WHERE tenant_id = $1 AND status = 'pending'
			  ORDER BY created_at DESC
			  LIMIT 1
			)
			RETURNING invoice_number`, tenantID, expiresAt).Scan(&invoiceNumber)
		if err == sql.ErrNoRows {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "no pending invoice to refresh"})
		}
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to refresh payment link"})
		}
		beforeEventValue = "pending"
		afterEventValue = invoiceNumber
		result["payment_link_expires_at"] = expiresAt
		result["invoice_number"] = invoiceNumber
	case "add-note":
		eventAction = "add_note"
		note := strings.TrimSpace(req.Note)
		if note == "" {
			note = reason
		}
		if _, err := tx.ExecContext(c.Request().Context(), `
			INSERT INTO platform_tenant_notes (tenant_id, platform_admin_id, body)
			VALUES ($1, $2, $3)`, tenantID, adminID, note); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to add tenant note"})
		}
		beforeEventValue = "note"
		afterEventValue = "created"
	case "assign-support-owner":
		eventAction = "assign_support_owner"
		if strings.TrimSpace(req.TicketID) == "" {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "ticket_id is required"})
		}
		assigneeID := adminID
		if req.AssigneeID != nil && strings.TrimSpace(*req.AssigneeID) != "" {
			assigneeID = strings.TrimSpace(*req.AssigneeID)
		}
		beforeEventValue = req.TicketID
		afterEventValue = assigneeID
		if _, err := tx.ExecContext(c.Request().Context(), `
			UPDATE support_tickets
			SET assigned_to_platform_admin_id = $3,
			    last_activity_at = NOW(),
			    updated_at = NOW()
			WHERE id = $1 AND tenant_id = $2`, req.TicketID, tenantID, assigneeID); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to assign support owner"})
		}
	default:
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "unsupported tenant action"})
	}
	if action == "suspend" || action == "deactivate" || action == "delete" || action == "cancel-delete" || action == "reactivate" {
		afterEventValue = afterStatus
	}
	if afterStatus != "active" {
		_, _ = tx.ExecContext(c.Request().Context(), `
			UPDATE sessions SET terminated_at = COALESCE(terminated_at, NOW()) WHERE tenant_id = $1 AND terminated_at IS NULL`, tenantID)
	}
	if _, err := tx.ExecContext(c.Request().Context(), `
		INSERT INTO tenant_lifecycle_events
		  (tenant_id, platform_admin_id, action, reason, before_status, after_status, effective_at, delete_after)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		tenantID, adminID, eventAction, nullString(reason), beforeEventValue, afterEventValue, now, deleteAfter); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to write lifecycle event"})
	}
	if err := tx.Commit(); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to commit tenant action"})
	}
	result["status"] = afterStatus
	result["subscription_status"] = beforeSubscriptionStatus
	return c.JSON(http.StatusOK, result)
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

func (s *PlatformService) ListAuditEvents(c echo.Context) error {
	limit := clampInt(c.QueryParam("limit"), 100, 1, 200)
	offset := clampInt(c.QueryParam("offset"), 0, 0, 1000000)
	tenantID := c.QueryParam("tenant_id")
	adminID := c.QueryParam("admin_id")
	action := c.QueryParam("action")

	rows, err := s.db.QueryContext(c.Request().Context(), `
		WITH combined_events AS (
			SELECT 'lifecycle' AS type,
			       tle.action AS action,
			       'platform_admin' AS actor_type,
			       COALESCE(t.business_name, 'tenant') AS resource,
			       tle.reason AS reason,
			       tle.before_status AS before_status,
			       tle.after_status AS after_status,
			       tle.created_at AS occurred_at,
			       tle.delete_after AS delete_after
			FROM tenant_lifecycle_events tle
			LEFT JOIN tenants t ON t.id = tle.tenant_id
			WHERE ($1 = '' OR tle.tenant_id = NULLIF($1, '')::uuid)
			  AND ($2 = '' OR tle.platform_admin_id = NULLIF($2, '')::uuid)
			  AND ($3 = '' OR LOWER(tle.action) = LOWER($3))
			UNION ALL
			SELECT 'audit' AS type,
			       ae.action AS action,
			       ae.actor_type AS actor_type,
			       COALESCE(t.business_name || ' - ', '') || ae.resource_type || ':' || ae.resource_id AS resource,
			       COALESCE(ae.purpose, NULLIF(ae.metadata::text, 'null')) AS reason,
			       ae.before_value::text AS before_status,
			       ae.after_value::text AS after_status,
			       ae.timestamp AS occurred_at,
			       NULL::timestamptz AS delete_after
			FROM audit_events ae
			LEFT JOIN tenants t ON t.id = ae.tenant_id
			WHERE ($1 = '' OR ae.tenant_id = NULLIF($1, '')::uuid)
			  AND ($2 = '' OR ae.actor_id = NULLIF($2, '')::uuid)
			  AND ($3 = '' OR LOWER(ae.action) = LOWER($3))
		)
		SELECT type, action, actor_type, resource, reason, before_status, after_status, occurred_at, delete_after
		FROM combined_events
		ORDER BY occurred_at DESC
		LIMIT $4 OFFSET $5`, tenantID, adminID, action, limit, offset)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to load audit events"})
	}
	defer rows.Close()

	items := []ActivityItem{}
	for rows.Next() {
		var item ActivityItem
		var reason, before, after sql.NullString
		var deleteAfter sql.NullTime
		if err := rows.Scan(&item.Type, &item.Action, &item.ActorType, &item.Resource, &reason, &before, &after, &item.OccurredAt, &deleteAfter); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to scan audit events"})
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
	if err := rows.Err(); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to load audit events"})
	}
	return c.JSON(http.StatusOK, map[string]interface{}{"events": items, "limit": limit, "offset": offset})
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
		       COALESCE((
		         SELECT u.email FROM users u
		         WHERE u.tenant_id = t.id AND u.role = 'owner' AND u.status != 'deleted'
		         ORDER BY u.created_at ASC
		         LIMIT 1
		       ), ''),
		       t.subscription_plan, t.billing_cycle, t.subscription_status,
		       t.trial_ends_at, t.subscription_ends_at,
		       COALESCE(CASE WHEN t.subscription_plan = 'trial' THEN t.trial_ends_at ELSE t.subscription_ends_at END, t.trial_ends_at, t.subscription_ends_at),
		       t.storage_used_bytes, t.storage_quota_bytes,
		       COALESCE((SELECT COUNT(*) FROM users u WHERE u.tenant_id = t.id AND u.status != 'deleted'), 0),
		       COALESCE((SELECT COUNT(*) FROM users u WHERE u.tenant_id = t.id AND u.status = 'active'), 0),
		       COALESCE((SELECT SUM(amount_idr) FROM billing_invoices bi WHERE bi.tenant_id = t.id AND bi.status = 'paid'), 0),
		       COALESCE((SELECT COUNT(*) FROM support_tickets st WHERE st.tenant_id = t.id AND st.status IN ('open','in_progress','waiting_on_tenant')), 0),
		       (SELECT MAX(u.last_login_at) FROM users u WHERE u.tenant_id = t.id),
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

func (s *PlatformService) loadTenantBilling(ctx context.Context, tenantID string) (TenantBillingDetail, error) {
	detail := TenantBillingDetail{
		Invoices:        []TenantInvoice{},
		PaymentAttempts: []PaymentAttempt{},
	}
	rows, err := s.db.QueryContext(ctx, tenantInvoiceSelect(`
		WHERE bi.tenant_id = $1
		ORDER BY bi.created_at DESC
		LIMIT 100`), tenantID)
	if err != nil {
		return detail, err
	}
	defer rows.Close()
	for rows.Next() {
		inv, err := scanTenantInvoice(rows)
		if err != nil {
			return detail, err
		}
		detail.Invoices = append(detail.Invoices, inv)
	}
	if err := rows.Err(); err != nil {
		return detail, err
	}

	attemptRows, err := s.db.QueryContext(ctx, `
		SELECT bpa.id, bpa.invoice_id, bpa.tenant_id, bpa.amount_idr,
		       bpa.midtrans_order_id, bpa.payment_method, bpa.status, bpa.error_msg, bpa.created_at
		FROM billing_payment_attempts bpa
		WHERE bpa.tenant_id = $1
		ORDER BY bpa.created_at DESC
		LIMIT 100`, tenantID)
	if err != nil {
		return detail, err
	}
	defer attemptRows.Close()
	for attemptRows.Next() {
		attempt, err := scanPaymentAttempt(attemptRows)
		if err != nil {
			return detail, err
		}
		detail.PaymentAttempts = append(detail.PaymentAttempts, attempt)
	}
	return detail, attemptRows.Err()
}

func (s *PlatformService) loadTenantTickets(ctx context.Context, tenantID, status string) ([]Ticket, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT st.id, st.tenant_id, t.business_name, st.subject, st.description,
		       st.category, st.priority, st.status, st.assigned_to_platform_admin_id,
		       st.created_at, st.updated_at, st.last_activity_at, st.resolved_at
		FROM support_tickets st
		JOIN tenants t ON t.id = st.tenant_id
		WHERE st.tenant_id = $1
		  AND ($2 = '' OR st.status = $2)
		ORDER BY st.last_activity_at DESC
		LIMIT 100`, tenantID, status)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	tickets := []Ticket{}
	for rows.Next() {
		t, err := scanTicket(rows)
		if err != nil {
			return nil, err
		}
		tickets = append(tickets, t)
	}
	return tickets, rows.Err()
}

func (s *PlatformService) loadTenantNotes(ctx context.Context, tenantID string, limit int) ([]TenantNote, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, tenant_id, platform_admin_id, body, created_at
		FROM platform_tenant_notes
		WHERE tenant_id = $1
		ORDER BY created_at DESC
		LIMIT $2`, tenantID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	notes := []TenantNote{}
	for rows.Next() {
		var note TenantNote
		var adminID sql.NullString
		if err := rows.Scan(&note.ID, &note.TenantID, &adminID, &note.Body, &note.CreatedAt); err != nil {
			return nil, err
		}
		if adminID.Valid {
			note.AdminID = &adminID.String
		}
		notes = append(notes, note)
	}
	return notes, rows.Err()
}

func scanTenantSummary(row interface {
	Scan(dest ...interface{}) error
}) (TenantSummary, error) {
	var tenant TenantSummary
	var trialEndsAt, subscriptionEndsAt, expiresAt, lastActiveAt, suspendedAt, deactivatedAt, scheduledDeleteAt sql.NullTime
	var ownerEmail, statusReason sql.NullString
	err := row.Scan(
		&tenant.ID, &tenant.BusinessName, &tenant.Slug, &tenant.Status,
		&ownerEmail,
		&tenant.SubscriptionPlan, &tenant.BillingCycle, &tenant.SubscriptionStatus,
		&trialEndsAt, &subscriptionEndsAt, &expiresAt,
		&tenant.StorageUsedBytes, &tenant.StorageQuotaBytes,
		&tenant.UserCount, &tenant.ActiveUserCount, &tenant.PaidInvoiceTotalIDR, &tenant.OpenTicketCount, &lastActiveAt,
		&tenant.CreatedAt, &tenant.UpdatedAt,
		&suspendedAt, &deactivatedAt, &scheduledDeleteAt, &statusReason,
	)
	if ownerEmail.Valid {
		tenant.OwnerEmail = maskEmail(ownerEmail.String)
	}
	if trialEndsAt.Valid {
		tenant.TrialEndsAt = &trialEndsAt.Time
	}
	if subscriptionEndsAt.Valid {
		tenant.SubscriptionEndsAt = &subscriptionEndsAt.Time
	}
	if expiresAt.Valid {
		tenant.ExpiresAt = &expiresAt.Time
	}
	if lastActiveAt.Valid {
		tenant.LastActiveAt = &lastActiveAt.Time
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

func tenantInvoiceSelect(suffix string) string {
	return `
		SELECT bi.id, bi.tenant_id, t.business_name, bi.invoice_number, bi.amount_idr,
		       bi.billing_interval, bi.period_start, bi.period_end, bi.due_at, bi.status,
		       bi.paid_at, bi.midtrans_order_id, bi.midtrans_payment_url,
		       bi.payment_link_expires_at, bi.created_at, bi.updated_at
		FROM billing_invoices bi
		JOIN tenants t ON t.id = bi.tenant_id
	` + suffix
}

func scanTenantInvoice(row interface {
	Scan(dest ...interface{}) error
}) (TenantInvoice, error) {
	var inv TenantInvoice
	var paidAt, paymentLinkExpiresAt sql.NullTime
	var midtransOrderID, paymentURL sql.NullString
	err := row.Scan(
		&inv.ID, &inv.TenantID, &inv.TenantName, &inv.InvoiceNumber, &inv.AmountIDR,
		&inv.BillingInterval, &inv.PeriodStart, &inv.PeriodEnd, &inv.DueAt, &inv.Status,
		&paidAt, &midtransOrderID, &paymentURL, &paymentLinkExpiresAt, &inv.CreatedAt, &inv.UpdatedAt,
	)
	if paidAt.Valid {
		inv.PaidAt = &paidAt.Time
	}
	if midtransOrderID.Valid {
		inv.MidtransOrderID = &midtransOrderID.String
	}
	if paymentURL.Valid {
		inv.PaymentURL = &paymentURL.String
	}
	if paymentLinkExpiresAt.Valid {
		inv.PaymentLinkExpiresAt = &paymentLinkExpiresAt.Time
	}
	return inv, err
}

func scanPaymentAttempt(row interface {
	Scan(dest ...interface{}) error
}) (PaymentAttempt, error) {
	var attempt PaymentAttempt
	var midtransOrderID, paymentMethod, errorMsg sql.NullString
	err := row.Scan(
		&attempt.ID, &attempt.InvoiceID, &attempt.TenantID, &attempt.AmountIDR,
		&midtransOrderID, &paymentMethod, &attempt.Status, &errorMsg, &attempt.CreatedAt,
	)
	if midtransOrderID.Valid {
		attempt.MidtransOrderID = &midtransOrderID.String
	}
	if paymentMethod.Valid {
		attempt.PaymentMethod = &paymentMethod.String
	}
	if errorMsg.Valid {
		attempt.ErrorMsg = &errorMsg.String
	}
	return attempt, err
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
	rangeName := c.QueryParam("range")
	if rangeName != "" && rangeName != "custom" && startStr == "" && endStr == "" {
		switch rangeName {
		case "month":
			return time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC), now, nil
		case "quarter":
			quarterStartMonth := time.Month(((int(now.Month())-1)/3)*3 + 1)
			return time.Date(now.Year(), quarterStartMonth, 1, 0, 0, 0, 0, time.UTC), now, nil
		case "year":
			return time.Date(now.Year(), time.January, 1, 0, 0, 0, 0, time.UTC), now, nil
		default:
			return time.Time{}, time.Time{}, errors.New("invalid range, expected month, quarter, year, or custom")
		}
	}
	if rangeName == "custom" && (startStr == "" || endStr == "") {
		return time.Time{}, time.Time{}, errors.New("custom range requires start_date and end_date")
	}
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

func tenantSortClause(sort string) string {
	switch sort {
	case "business_name":
		return "t.business_name ASC"
	case "-business_name":
		return "t.business_name DESC"
	case "expires_at":
		return "expires_at ASC NULLS LAST"
	case "-expires_at":
		return "expires_at DESC NULLS LAST"
	case "paid_total":
		return "paid_invoice_total_idr ASC"
	case "-paid_total":
		return "paid_invoice_total_idr DESC"
	case "last_active":
		return "last_active_at ASC NULLS LAST"
	case "-last_active":
		return "last_active_at DESC NULLS LAST"
	case "created_at":
		return "t.created_at ASC"
	default:
		return "t.created_at DESC"
	}
}

func validSubscriptionPlan(plan string) bool {
	switch plan {
	case "trial", "starter", "professional", "enterprise":
		return true
	default:
		return false
	}
}

func validBillingCycle(cycle string) bool {
	return cycle == "monthly" || cycle == "annual"
}

func nullableTimeString(value sql.NullTime) string {
	if !value.Valid {
		return ""
	}
	return value.Time.Format(time.RFC3339)
}

func maskEmail(email string) string {
	email = strings.TrimSpace(email)
	if email == "" {
		return ""
	}
	parts := strings.Split(email, "@")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "stored securely"
	}
	name := parts[0]
	if len(name) <= 2 {
		return name[:1] + "***@" + parts[1]
	}
	return name[:2] + "***@" + parts[1]
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
