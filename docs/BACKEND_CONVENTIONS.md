# Backend Development Conventions (Go)

**Last Updated:** December 1, 2024  
**Project:** Point of Sale System  
**Language:** Go 1.21+  
**Framework:** Echo v4

---

## 📁 Service Structure

### Standard Service Layout
```
<service-name>/
├── main.go                 # Entry point
├── go.mod                  # Dependencies
├── go.sum
├── .env.example            # Environment template
├── README.md               # Service documentation
├── api/                    # HTTP handlers
│   ├── handler1.go
│   └── handler2.go
├── src/
│   ├── config/            # Configuration
│   │   ├── database.go
│   │   └── redis.go
│   ├── middleware/        # Middleware
│   │   └── tenant.go
│   ├── models/            # Data models
│   │   └── model.go
│   ├── repository/        # Database layer
│   │   └── repository.go
│   ├── services/          # Business logic
│   │   └── service.go
│   └── utils/             # Utilities
│       ├── logger.go
│       └── errors.go
└── tests/                 # Tests
    └── integration/
```

**Pattern:** `<feature>-service` (e.g., `product-service`, `auth-service`)

---

## 🗄️ Database & RLS

### ⚠️ CRITICAL: RLS Context Setup

**Problem:** PostgreSQL's `SET LOCAL` does NOT support parameterized queries.

#### ❌ WRONG - Will cause syntax error
```go
// This will fail with "syntax error at or near $1"
_, err := db.Exec("SET LOCAL app.current_tenant_id = $1", tenantID)
```

#### ✅ CORRECT - Use fmt.Sprintf
```go
import "fmt"

// Set RLS context using string formatting
// Safe because tenant_id is always a UUID (validated format)
setContextSQL := fmt.Sprintf("SET LOCAL app.current_tenant_id = '%s'", tenantID)
_, err := db.Exec(setContextSQL)
if err != nil {
    return fmt.Errorf("failed to set tenant context: %w", err)
}
```

### Why String Formatting Is Safe Here

1. **Tenant ID is always a UUID** (validated format: `^[0-9a-f-]{36}$`)
2. **UUIDs cannot contain SQL injection characters** (only alphanumeric + hyphens)
3. **SET LOCAL is session-scoped** (only affects current transaction)
4. **Consistent with other PostgreSQL session variables**

### Tenant Middleware Pattern

```go
package middleware

import (
    "fmt"
    "github.com/labstack/echo/v4"
    "github.com/pos/backend/<service>/src/config"
)

func TenantMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
    return func(c echo.Context) error {
        // 1. Get tenant ID from header (set by API Gateway)
        tenantID := c.Request().Header.Get("X-Tenant-ID")
        
        // 2. Fallback to context if not in header
        if tenantID == "" {
            tenantIDCtx := c.Get("tenant_id")
            if tenantIDCtx != nil {
                tenantID = tenantIDCtx.(string)
            }
        }
        
        // 3. Validate tenant ID exists
        if tenantID == "" {
            c.Logger().Error("Tenant ID not found")
            return echo.NewHTTPError(401, "Tenant ID not found")
        }

        // 4. Set in context for handlers
        c.Set("tenant_id", tenantID)

        // 5. Set RLS context in database
        // Note: SET LOCAL doesn't support parameterized queries
        setContextSQL := fmt.Sprintf("SET LOCAL app.current_tenant_id = '%s'", tenantID)
        _, err := config.DB.Exec(setContextSQL)
        if err != nil {
            c.Logger().Errorf("Failed to set RLS context: %v", err)
            return echo.NewHTTPError(500, "Failed to set tenant context")
        }

        return next(c)
    }
}
```

### Database Transaction Pattern

```go
// Always use transactions for multi-step operations
func (s *Service) CreateWithTransaction(ctx context.Context, data *Data) error {
    tx, err := s.db.BeginTx(ctx, nil)
    if err != nil {
        return fmt.Errorf("failed to begin transaction: %w", err)
    }
    defer tx.Rollback() // Safe to call even after commit

    // Set RLS context in transaction
    setContextSQL := fmt.Sprintf("SET LOCAL app.current_tenant_id = '%s'", tenantID)
    _, err = tx.ExecContext(ctx, setContextSQL)
    if err != nil {
        return fmt.Errorf("failed to set tenant context: %w", err)
    }

    // Perform operations
    if err := s.doOperation1(ctx, tx, data); err != nil {
        return err // tx.Rollback() called by defer
    }

    if err := s.doOperation2(ctx, tx, data); err != nil {
        return err
    }

    // Commit transaction
    if err := tx.Commit(); err != nil {
        return fmt.Errorf("failed to commit transaction: %w", err)
    }

    return nil
}
```

### ⚠️ CRITICAL: JOINs with RLS Tables

**Problem:** When joining tables with RLS enabled, the joined table may not be accessible without explicit tenant filtering.

#### ❌ WRONG - May return NULL for joined fields
```go
query := `
    SELECT p.id, p.name, p.category_id, c.name as category_name
    FROM products p
    LEFT JOIN categories c ON p.category_id = c.id
    WHERE p.id = $1
`
// Result: category_name may be NULL even when category_id exists
```

#### ✅ CORRECT - Add tenant filter to JOIN condition
```go
query := `
    SELECT p.id, p.name, p.category_id, c.name as category_name
    FROM products p
    LEFT JOIN categories c ON p.category_id = c.id AND c.tenant_id = p.tenant_id
    WHERE p.id = $1
`
// Result: category_name correctly populated
```

### Why This Is Required

1. **RLS policies apply to JOINed tables** - The categories table has RLS enabled
2. **SET LOCAL only sets the session variable** - JOINs need explicit tenant filtering
3. **LEFT JOIN may silently return NULL** - Without tenant filter, RLS blocks the category row
4. **Always add tenant filter to JOINs** - Even with RLS middleware active

---

## 📦 Models & Validation

### Model Structure

```go
package models

import (
    "time"
    "github.com/google/uuid"
)

// Database model
type Product struct {
    ID           uuid.UUID  `json:"id" db:"id"`
    TenantID     uuid.UUID  `json:"tenant_id" db:"tenant_id"`
    Name         string     `json:"name" db:"name" validate:"required,min=1,max=255"`
    Description  *string    `json:"description,omitempty" db:"description"`
    Price        float64    `json:"price" db:"price" validate:"required,gte=0"`
    StockQty     int        `json:"stock_quantity" db:"stock_quantity"`
    ArchivedAt   *time.Time `json:"archived_at,omitempty" db:"archived_at"`
    CreatedAt    time.Time  `json:"created_at" db:"created_at"`
    UpdatedAt    time.Time  `json:"updated_at" db:"updated_at"`
}

// Request DTOs
type CreateProductRequest struct {
    Name        string   `json:"name" validate:"required,min=1,max=255"`
    Description *string  `json:"description,omitempty"`
    Price       float64  `json:"price" validate:"required,gte=0"`
    CategoryID  *string  `json:"category_id,omitempty" validate:"omitempty,uuid"`
}

type UpdateProductRequest struct {
    Name        *string  `json:"name,omitempty" validate:"omitempty,min=1,max=255"`
    Description *string  `json:"description,omitempty"`
    Price       *float64 `json:"price,omitempty" validate:"omitempty,gte=0"`
}

// Response DTOs
type ProductResponse struct {
    ID          string  `json:"id"`
    Name        string  `json:"name"`
    Price       float64 `json:"price"`
    CategoryName *string `json:"category_name,omitempty"`
    CreatedAt   string  `json:"created_at"`
}
```

### Field Conventions

| Type | Usage | Example |
|------|-------|---------|
| `uuid.UUID` | IDs | `ID uuid.UUID` |
| `string` | Required text | `Name string` |
| `*string` | Optional text | `Description *string` |
| `float64` | Money/decimals | `Price float64` |
| `int` | Counts/quantities | `StockQty int` |
| `bool` | Flags | `IsActive bool` |
| `time.Time` | Timestamps | `CreatedAt time.Time` |
| `*time.Time` | Optional timestamps | `ArchivedAt *time.Time` |

### Validation Tags

```go
validate:"required"              // Field must be present
validate:"omitempty"             // Skip validation if empty
validate:"min=1,max=255"         // String length
validate:"gte=0,lte=100"         // Number range
validate:"email"                 // Email format
validate:"uuid"                  // UUID format
validate:"oneof=active inactive" // Enum values
```

---

## 🔧 Repository Layer

### Repository Pattern

```go
package repository

import (
    "context"
    "database/sql"
    "github.com/google/uuid"
    "github.com/pos/backend/<service>/src/models"
)

type ProductRepository struct {
    db *sql.DB
}

func NewProductRepository(db *sql.DB) *ProductRepository {
    return &ProductRepository{db: db}
}

// Create
func (r *ProductRepository) Create(ctx context.Context, product *models.Product) error {
    query := `
        INSERT INTO products (id, tenant_id, name, price, created_at, updated_at)
        VALUES ($1, $2, $3, $4, $5, $6)
    `
    _, err := r.db.ExecContext(ctx, query,
        product.ID,
        product.TenantID,
        product.Name,
        product.Price,
        product.CreatedAt,
        product.UpdatedAt,
    )
    return err
}

// Get by ID
func (r *ProductRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Product, error) {
    query := `
        SELECT id, tenant_id, name, price, created_at, updated_at
        FROM products
        WHERE id = $1 AND archived_at IS NULL
    `
    
    product := &models.Product{}
    err := r.db.QueryRowContext(ctx, query, id).Scan(
        &product.ID,
        &product.TenantID,
        &product.Name,
        &product.Price,
        &product.CreatedAt,
        &product.UpdatedAt,
    )
    
    if err == sql.ErrNoRows {
        return nil, nil // Not found
    }
    if err != nil {
        return nil, err
    }
    
    return product, nil
}

// List with pagination
func (r *ProductRepository) List(ctx context.Context, limit, offset int) ([]*models.Product, error) {
    query := `
        SELECT id, tenant_id, name, price, created_at, updated_at
        FROM products
        WHERE archived_at IS NULL
        ORDER BY created_at DESC
        LIMIT $1 OFFSET $2
    `
    
    rows, err := r.db.QueryContext(ctx, query, limit, offset)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var products []*models.Product
    for rows.Next() {
        product := &models.Product{}
        if err := rows.Scan(
            &product.ID,
            &product.TenantID,
            &product.Name,
            &product.Price,
            &product.CreatedAt,
            &product.UpdatedAt,
        ); err != nil {
            return nil, err
        }
        products = append(products, product)
    }

    return products, rows.Err()
}

// Update
func (r *ProductRepository) Update(ctx context.Context, product *models.Product) error {
    query := `
        UPDATE products
        SET name = $1, price = $2, updated_at = $3
        WHERE id = $4
    `
    _, err := r.db.ExecContext(ctx, query,
        product.Name,
        product.Price,
        product.UpdatedAt,
        product.ID,
    )
    return err
}

// Delete (soft delete)
func (r *ProductRepository) Archive(ctx context.Context, id uuid.UUID) error {
    query := `
        UPDATE products
        SET archived_at = NOW()
        WHERE id = $1 AND archived_at IS NULL
    `
    _, err := r.db.ExecContext(ctx, query, id)
    return err
}
```

---

## 🎯 Service Layer

### Service Pattern

```go
package services

import (
    "context"
    "fmt"
    "time"
    "github.com/google/uuid"
    "github.com/pos/backend/<service>/src/models"
    "github.com/pos/backend/<service>/src/repository"
)

type ProductService struct {
    repo *repository.ProductRepository
}

func NewProductService(repo *repository.ProductRepository) *ProductService {
    return &ProductService{repo: repo}
}

// Create product with business logic
func (s *ProductService) CreateProduct(ctx context.Context, tenantID uuid.UUID, req *models.CreateProductRequest) (*models.Product, error) {
    // Validate business rules
    if req.Price < 0 {
        return nil, fmt.Errorf("price cannot be negative")
    }

    // Create model
    product := &models.Product{
        ID:        uuid.New(),
        TenantID:  tenantID,
        Name:      req.Name,
        Price:     req.Price,
        CreatedAt: time.Now(),
        UpdatedAt: time.Now(),
    }

    // Save to database
    if err := s.repo.Create(ctx, product); err != nil {
        return nil, fmt.Errorf("failed to create product: %w", err)
    }

    return product, nil
}

// Get product
func (s *ProductService) GetProduct(ctx context.Context, id uuid.UUID) (*models.Product, error) {
    product, err := s.repo.GetByID(ctx, id)
    if err != nil {
        return nil, fmt.Errorf("failed to get product: %w", err)
    }
    if product == nil {
        return nil, fmt.Errorf("product not found")
    }
    return product, nil
}

// List products with pagination
func (s *ProductService) ListProducts(ctx context.Context, page, limit int) ([]*models.Product, int, error) {
    if page < 1 {
        page = 1
    }
    if limit < 1 || limit > 100 {
        limit = 20
    }

    offset := (page - 1) * limit

    products, err := s.repo.List(ctx, limit, offset)
    if err != nil {
        return nil, 0, fmt.Errorf("failed to list products: %w", err)
    }

    // Get total count for pagination
    total, err := s.repo.Count(ctx)
    if err != nil {
        return nil, 0, fmt.Errorf("failed to count products: %w", err)
    }

    return products, total, nil
}
```

---

## 🌐 HTTP Handlers

### Handler Pattern

```go
package api

import (
    "net/http"
    "github.com/google/uuid"
    "github.com/labstack/echo/v4"
    "github.com/pos/backend/<service>/src/models"
    "github.com/pos/backend/<service>/src/services"
)

type ProductHandler struct {
    service *services.ProductService
}

func NewProductHandler(service *services.ProductService) *ProductHandler {
    return &ProductHandler{service: service}
}

// Create product
func (h *ProductHandler) Create(c echo.Context) error {
    // Get tenant ID from context (set by middleware)
    tenantID, err := getTenantID(c)
    if err != nil {
        return echo.NewHTTPError(http.StatusUnauthorized, "Tenant ID not found")
    }

    // Parse request body
    var req models.CreateProductRequest
    if err := c.Bind(&req); err != nil {
        return echo.NewHTTPError(http.StatusBadRequest, "Invalid request body")
    }

    // Validate request
    if err := c.Validate(&req); err != nil {
        return echo.NewHTTPError(http.StatusBadRequest, err.Error())
    }

    // Call service
    product, err := h.service.CreateProduct(c.Request().Context(), tenantID, &req)
    if err != nil {
        c.Logger().Errorf("Failed to create product: %v", err)
        return echo.NewHTTPError(http.StatusInternalServerError, "Failed to create product")
    }

    return c.JSON(http.StatusCreated, product)
}

// Get product by ID
func (h *ProductHandler) GetByID(c echo.Context) error {
    // Parse ID from URL
    id, err := uuid.Parse(c.Param("id"))
    if err != nil {
        return echo.NewHTTPError(http.StatusBadRequest, "Invalid product ID")
    }

    // Call service
    product, err := h.service.GetProduct(c.Request().Context(), id)
    if err != nil {
        if err.Error() == "product not found" {
            return echo.NewHTTPError(http.StatusNotFound, "Product not found")
        }
        c.Logger().Errorf("Failed to get product: %v", err)
        return echo.NewHTTPError(http.StatusInternalServerError, "Failed to get product")
    }

    return c.JSON(http.StatusOK, product)
}

// List products
func (h *ProductHandler) List(c echo.Context) error {
    // Parse query parameters
    page := getIntQueryParam(c, "page", 1)
    limit := getIntQueryParam(c, "limit", 20)

    // Call service
    products, total, err := h.service.ListProducts(c.Request().Context(), page, limit)
    if err != nil {
        c.Logger().Errorf("Failed to list products: %v", err)
        return echo.NewHTTPError(http.StatusInternalServerError, "Failed to list products")
    }

    // Return paginated response
    return c.JSON(http.StatusOK, map[string]interface{}{
        "data":  products,
        "total": total,
        "page":  page,
        "limit": limit,
    })
}

// Helper functions
func getTenantID(c echo.Context) (uuid.UUID, error) {
    tenantIDStr, ok := c.Get("tenant_id").(string)
    if !ok {
        return uuid.Nil, fmt.Errorf("tenant ID not found in context")
    }
    return uuid.Parse(tenantIDStr)
}

func getIntQueryParam(c echo.Context, key string, defaultValue int) int {
    value := c.QueryParam(key)
    if value == "" {
        return defaultValue
    }
    intValue, err := strconv.Atoi(value)
    if err != nil {
        return defaultValue
    }
    return intValue
}
```

---

## 🚀 Main Application Setup

### main.go Pattern

```go
package main

import (
    "log"
    "os"
    
    "github.com/labstack/echo/v4"
    "github.com/labstack/echo/v4/middleware"
    "github.com/pos/backend/<service>/api"
    "github.com/pos/backend/<service>/src/config"
    customMiddleware "github.com/pos/backend/<service>/src/middleware"
    "github.com/pos/backend/<service>/src/repository"
    "github.com/pos/backend/<service>/src/services"
)

func main() {
    // Initialize database
    config.InitDB()
    defer config.CloseDB()

    // Initialize Echo
    e := echo.New()

    // Global middleware
    e.Use(middleware.Logger())
    e.Use(middleware.Recover())
    e.Use(middleware.CORS())

    // Health check
    e.GET("/health", func(c echo.Context) error {
        return c.JSON(200, map[string]string{"status": "ok"})
    })

    // Initialize repositories
    productRepo := repository.NewProductRepository(config.DB)

    // Initialize services
    productService := services.NewProductService(productRepo)

    // Initialize handlers
    productHandler := api.NewProductHandler(productService)

    // Routes with tenant middleware
    v1 := e.Group("/api/v1")
    v1.Use(customMiddleware.TenantMiddleware)

    // Product routes
    v1.POST("/products", productHandler.Create)
    v1.GET("/products", productHandler.List)
    v1.GET("/products/:id", productHandler.GetByID)
    v1.PUT("/products/:id", productHandler.Update)
    v1.DELETE("/products/:id", productHandler.Delete)

    // Start server
    port := os.Getenv("PORT")
    if port == "" {
        port = "8080"
    }

    log.Printf("Server starting on port %s", port)
    if err := e.Start(":" + port); err != nil {
        log.Fatal(err)
    }
}
```

---

## ⚠️ Common Mistakes to Avoid

### 1. ❌ Using $1 with SET LOCAL
```go
// WRONG - Will cause SQL syntax error
_, err := db.Exec("SET LOCAL app.current_tenant_id = $1", tenantID)
```

✅ **Correct:**
```go
setContextSQL := fmt.Sprintf("SET LOCAL app.current_tenant_id = '%s'", tenantID)
_, err := db.Exec(setContextSQL)
```

---

### 2. ❌ Not Setting RLS Context
```go
// WRONG - Queries will fail or return all tenants' data
func (h *Handler) GetAll(c echo.Context) error {
    products, _ := h.repo.GetAll() // Missing RLS context!
    return c.JSON(200, products)
}
```

✅ **Correct:**
```go
// Use TenantMiddleware to set RLS context
v1.Use(middleware.TenantMiddleware)
v1.GET("/products", handler.GetAll) // RLS context set by middleware
```

---

### 3. ❌ Not Using Transactions for Multi-Step Operations
```go
// WRONG - If second operation fails, first remains committed
func (s *Service) CreateOrder(ctx context.Context, order *Order) error {
    s.repo.CreateOrder(order)        // Commits
    s.repo.UpdateInventory(order)    // Fails - inconsistent state!
}
```

✅ **Correct:**
```go
func (s *Service) CreateOrder(ctx context.Context, order *Order) error {
    tx, _ := s.db.BeginTx(ctx, nil)
    defer tx.Rollback()
    
    s.repo.CreateOrderTx(tx, order)
    s.repo.UpdateInventoryTx(tx, order)
    
    return tx.Commit()
}
```

---

### 4. ❌ Returning Database Errors to Client
```go
// WRONG - Exposes internal details
func (h *Handler) Create(c echo.Context) error {
    _, err := h.service.Create(data)
    if err != nil {
        return c.JSON(500, err.Error()) // "pq: duplicate key..."
    }
}
```

✅ **Correct:**
```go
func (h *Handler) Create(c echo.Context) error {
    _, err := h.service.Create(data)
    if err != nil {
        c.Logger().Errorf("Failed to create: %v", err) // Log details
        return echo.NewHTTPError(500, "Failed to create resource") // Generic message
    }
}
```

---

### 5. ❌ Not Validating UUIDs from URL Parameters
```go
// WRONG - Can cause runtime panic
func (h *Handler) Get(c echo.Context) error {
    id := uuid.Parse(c.Param("id")) // Panic if invalid!
}
```

✅ **Correct:**
```go
func (h *Handler) Get(c echo.Context) error {
    id, err := uuid.Parse(c.Param("id"))
    if err != nil {
        return echo.NewHTTPError(400, "Invalid ID format")
    }
}
```

---

### 6. ❌ Not Handling sql.ErrNoRows
```go
// WRONG - Generic error for "not found"
func (r *Repo) GetByID(id uuid.UUID) (*Model, error) {
    var m Model
    err := r.db.QueryRow("SELECT ... WHERE id = $1", id).Scan(&m)
    return &m, err // Returns sql.ErrNoRows to caller
}
```

✅ **Correct:**
```go
func (r *Repo) GetByID(id uuid.UUID) (*Model, error) {
    var m Model
    err := r.db.QueryRow("SELECT ... WHERE id = $1", id).Scan(&m)
    if err == sql.ErrNoRows {
        return nil, nil // Not found - return nil, nil
    }
    if err != nil {
        return nil, err // Other database error
    }
    return &m, nil
}
```

---

### 7. ❌ Forgetting to Close Rows
```go
// WRONG - Resource leak
func (r *Repo) GetAll() ([]*Model, error) {
    rows, _ := r.db.Query("SELECT ...")
    // No defer rows.Close()!
    
    for rows.Next() {
        // ...
    }
}
```

✅ **Correct:**
```go
func (r *Repo) GetAll() ([]*Model, error) {
    rows, err := r.db.Query("SELECT ...")
    if err != nil {
        return nil, err
    }
    defer rows.Close() // Always close!
    
    for rows.Next() {
        // ...
    }
    return items, rows.Err()
}
```

---

## 🔒 Security Best Practices

### 1. Always Use Context for Database Operations
```go
// Use context for cancellation and timeouts
func (r *Repo) Create(ctx context.Context, data *Model) error {
    _, err := r.db.ExecContext(ctx, query, args...)
    return err
}
```

### 2. Validate All Input
```go
// Use validation tags and check in handlers
type CreateRequest struct {
    Name  string `json:"name" validate:"required,min=1,max=255"`
    Email string `json:"email" validate:"required,email"`
}

if err := c.Validate(&req); err != nil {
    return echo.NewHTTPError(400, err.Error())
}
```

### 3. Hash Passwords with bcrypt
```go
import "golang.org/x/crypto/bcrypt"

// Hash password
hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)

// Verify password
err := bcrypt.CompareHashAndPassword(hashedPassword, []byte(password))
```

### 4. Use Prepared Statements (Parameterized Queries)
```go
// Always use $1, $2, etc. for user input
query := "SELECT * FROM users WHERE email = $1"
row := db.QueryRowContext(ctx, query, email)

// NEVER concatenate user input into SQL
// WRONG: query := fmt.Sprintf("SELECT * FROM users WHERE email = '%s'", email)
```

---

## 📊 Error Handling

### Error Wrapping Pattern
```go
import "fmt"

// Wrap errors to preserve context
func (s *Service) ProcessOrder(ctx context.Context, order *Order) error {
    if err := s.validateOrder(order); err != nil {
        return fmt.Errorf("validation failed: %w", err)
    }
    
    if err := s.repo.Save(ctx, order); err != nil {
        return fmt.Errorf("failed to save order: %w", err)
    }
    
    return nil
}
```

### Custom Errors
```go
package utils

import "errors"

var (
    ErrNotFound      = errors.New("resource not found")
    ErrUnauthorized  = errors.New("unauthorized")
    ErrAlreadyExists = errors.New("resource already exists")
)

// Usage
func (s *Service) Get(id string) (*Model, error) {
    model, err := s.repo.GetByID(id)
    if err != nil {
        return nil, fmt.Errorf("failed to get model: %w", err)
    }
    if model == nil {
        return nil, utils.ErrNotFound
    }
    return model, nil
}
```

---

## 🧪 Testing Patterns

### Unit Test Example
```go
package services_test

import (
    "context"
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/mock"
)

// Mock repository
type MockProductRepo struct {
    mock.Mock
}

func (m *MockProductRepo) Create(ctx context.Context, product *Product) error {
    args := m.Called(ctx, product)
    return args.Error(0)
}

// Test
func TestCreateProduct(t *testing.T) {
    // Setup
    mockRepo := new(MockProductRepo)
    service := NewProductService(mockRepo)
    
    product := &Product{Name: "Test Product"}
    mockRepo.On("Create", mock.Anything, product).Return(nil)
    
    // Execute
    err := service.CreateProduct(context.Background(), product)
    
    // Assert
    assert.NoError(t, err)
    mockRepo.AssertExpectations(t)
}
```

---

## ✅ Feature Implementation Checklist

When adding a new feature:

- [ ] Create models in `src/models/<feature>.go`
- [ ] Create repository in `src/repository/<feature>_repository.go`
- [ ] Create service in `src/services/<feature>_service.go`
- [ ] Create handler in `api/<feature>_handler.go`
- [ ] Add routes in `main.go`
- [ ] Apply `TenantMiddleware` to routes
- [ ] Use `fmt.Sprintf` for `SET LOCAL` queries
- [ ] Handle `sql.ErrNoRows` properly
- [ ] Validate all input
- [ ] Use transactions for multi-step operations
- [ ] Add logging for errors
- [ ] Write unit tests
- [ ] Test with multiple tenants
- [ ] Document API endpoints

---

---

## 📤 API Response Patterns

### State-Changing Operations Should Return Updated Data

**Rule:** Operations that modify resources (archive, restore, update) should return the updated resource, not just 204 No Content.

#### ❌ WRONG - Returns empty response
```go
func (h *Handler) ArchiveProduct(c echo.Context) error {
    // ... archive logic ...
    return c.NoContent(http.StatusNoContent)  // Frontend doesn't know new state!
}
```

#### ✅ CORRECT - Returns updated resource
```go
func (h *Handler) ArchiveProduct(c echo.Context) error {
    // ... archive logic ...
    
    // Fetch and return the updated resource
    product, err := h.service.GetProduct(ctx, id)
    if err != nil {
        return utils.RespondInternalError(c, "Archived but failed to retrieve")
    }
    
    return c.JSON(http.StatusOK, product)  // Frontend gets updated state!
}
```

### Why Return Updated Data?

1. **Frontend State Management** - No need for additional API call to refresh data
2. **Consistency** - Frontend and backend stay in sync
3. **User Experience** - Instant UI updates without loading states
4. **Debugging** - Easier to verify operation success

### Exceptions

204 No Content is acceptable for:
- **DELETE** operations (resource no longer exists)
- **Logout** operations (session destroyed)
- **True "fire-and-forget"** operations with no state to return

---

## 🖼️ File Upload & Serving

### Pattern: Separate Upload and Serve Endpoints

```go
// Upload endpoint - multipart/form-data
e.POST("/products/:id/photo", h.UploadPhoto)

// Serve endpoint - returns file directly
e.GET("/products/:id/photo", h.GetPhoto)

// Delete endpoint
e.DELETE("/products/:id/photo", h.DeletePhoto)
```

### Photo Upload Handler
```go
func (h *Handler) UploadPhoto(c echo.Context) error {
    file, header, err := c.Request().FormFile("photo")
    if err != nil {
        return utils.RespondBadRequest(c, "Photo file is required")
    }
    defer file.Close()
    
    // Process and save file...
    
    // Return updated product with photo_path
    product, _ := h.service.GetProduct(ctx, id)
    return c.JSON(http.StatusOK, product)
}
```

### Photo Serve Handler
```go
func (h *Handler) GetPhoto(c echo.Context) error {
    photoPath, err := h.service.GetPhotoPath(ctx, id, tenantID)
    if err != nil {
        return utils.RespondNotFound(c, "Photo not found")
    }
    
    return c.File(photoPath)  // Echo serves file directly
}
```

### Frontend Photo Display
```typescript
// Service method
getPhotoUrl(id: string): string {
    return `${API_BASE_URL}/api/v1/products/${id}/photo`;
}

// Component usage
<img src={productService.getPhotoUrl(product.id)} alt={product.name} />
```

**Key Points:**
- Store relative path in DB (`uploads/tenant/product/photo.png`)
- Serve using absolute path (`/path/to/uploads/...`)
- Verify file exists before serving
- Check tenant ownership in GET endpoint

---

## 🔄 Update Operations & Data Preservation

### Preserve Non-Updatable Fields

When updating resources, some fields should be preserved from the existing record.

#### ❌ WRONG - Overwrites stock quantity
```go
func (h *Handler) UpdateProduct(c echo.Context) error {
    var req UpdateRequest
    c.Bind(&req)
    
    product := &models.Product{
        ID:            id,
        Name:          req.Name,
        Price:         req.Price,
        StockQuantity: req.StockQuantity,  // ❌ Should use dedicated endpoint!
    }
    // ... update ...
}
```

#### ✅ CORRECT - Preserves stock quantity
```go
func (h *Handler) UpdateProduct(c echo.Context) error {
    var req UpdateRequest
    c.Bind(&req)
    
    // Fetch existing product
    existing, err := h.service.GetProduct(ctx, id)
    if err != nil {
        return utils.RespondNotFound(c, "Product not found")
    }
    
    product := &models.Product{
        ID:            id,
        Name:          req.Name,
        Price:         req.Price,
        StockQuantity: existing.StockQuantity,  // ✅ Preserve!
    }
    // ... update ...
}
```

### Fields That Should Be Preserved

- **Stock quantities** - Use dedicated stock adjustment endpoints
- **Audit timestamps** - Let DB handle `updated_at`
- **Photo paths** - Use dedicated photo upload endpoints
- **Foreign keys** - Unless explicitly changing relationships
- **Calculated fields** - Should be computed, not set

---

## 📚 Quick Reference

| Pattern | Example |
|---------|---------|
| Service port | `8080` (or from `PORT` env) |
| Binary naming | `<service>.bin` |
| RLS context | `fmt.Sprintf("SET LOCAL app.current_tenant_id = '%s'", tenantID)` |
| UUID type | `github.com/google/uuid` |
| HTTP framework | `github.com/labstack/echo/v4` |
| DB driver | `database/sql` with `github.com/lib/pq` |
| Validation | `github.com/go-playground/validator/v10` |
| State-change response | Return updated resource (200 + JSON) |
| Delete response | 204 No Content (no body) |
| File serve | `c.File(absolutePath)` |

---

**Follow these conventions to avoid common pitfalls and maintain code consistency!** 🚀
