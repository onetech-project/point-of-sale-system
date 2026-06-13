package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/pos/api-gateway/middleware"
)

func TestIngredientInventoryRoutesSplitFromProductInventoryRoutes(t *testing.T) {
	productHits := 0
	productService := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		productHits++
		if r.Method != http.MethodGet || r.URL.Path != "/api/v1/inventory/adjustments" {
			t.Fatalf("unexpected product-service request: %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"service":"product"}`))
	}))
	defer productService.Close()

	inventoryHits := 0
	inventoryService := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		inventoryHits++
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/inventory/adjustments":
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/uoms":
		default:
			t.Fatalf("unexpected inventory-service request: %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"service":"inventory"}`))
	}))
	defer inventoryService.Close()

	e := echo.New()
	protected := e.Group("")
	protected.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Set("role", string(middleware.RoleManager))
			c.Set("tenant_id", "tenant-1")
			return next(c)
		}
	})
	registerIngredientInventoryRoutes(protected, inventoryService.URL)
	registerProductRoutes(protected, productService.URL)

	postAdjustmentReq := httptest.NewRequest(http.MethodPost, "/api/v1/inventory/adjustments", strings.NewReader(`{}`))
	postAdjustmentRec := httptest.NewRecorder()
	e.ServeHTTP(postAdjustmentRec, postAdjustmentReq)
	if postAdjustmentRec.Code != http.StatusOK {
		t.Fatalf("POST adjustment status = %d, want 200; body=%s", postAdjustmentRec.Code, postAdjustmentRec.Body.String())
	}

	getAdjustmentReq := httptest.NewRequest(http.MethodGet, "/api/v1/inventory/adjustments", nil)
	getAdjustmentRec := httptest.NewRecorder()
	e.ServeHTTP(getAdjustmentRec, getAdjustmentReq)
	if getAdjustmentRec.Code != http.StatusOK {
		t.Fatalf("GET adjustment status = %d, want 200; body=%s", getAdjustmentRec.Code, getAdjustmentRec.Body.String())
	}

	uomsReq := httptest.NewRequest(http.MethodGet, "/api/v1/uoms", nil)
	uomsRec := httptest.NewRecorder()
	e.ServeHTTP(uomsRec, uomsReq)
	if uomsRec.Code != http.StatusOK {
		t.Fatalf("GET uoms status = %d, want 200; body=%s", uomsRec.Code, uomsRec.Body.String())
	}

	if productHits != 1 {
		t.Fatalf("product hits = %d, want 1", productHits)
	}
	if inventoryHits != 2 {
		t.Fatalf("inventory hits = %d, want 2", inventoryHits)
	}
}

func TestProductRecipeRoutesGoToInventoryService(t *testing.T) {
	productService := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/recipe") || strings.Contains(r.URL.Path, "/recipes") {
			t.Fatalf("recipe route reached product-service: %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"service":"product"}`))
	}))
	defer productService.Close()

	inventoryHits := 0
	inventoryService := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		inventoryHits++
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/products/product-1/recipe":
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/products/product-1/recipes":
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/products/product-1/recipes":
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/products/product-1/recipes/2":
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/products/product-1/recipe/cost":
		default:
			t.Fatalf("unexpected inventory-service request: %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"service":"inventory"}`))
	}))
	defer inventoryService.Close()

	e := echo.New()
	protected := e.Group("")
	protected.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Set("role", string(middleware.RoleManager))
			c.Set("tenant_id", "tenant-1")
			return next(c)
		}
	})
	registerIngredientInventoryRoutes(protected, inventoryService.URL)
	registerProductRoutes(protected, productService.URL)

	requests := []*http.Request{
		httptest.NewRequest(http.MethodGet, "/api/v1/products/product-1/recipe", nil),
		httptest.NewRequest(http.MethodPost, "/api/v1/products/product-1/recipes", strings.NewReader(`{}`)),
		httptest.NewRequest(http.MethodGet, "/api/v1/products/product-1/recipes", nil),
		httptest.NewRequest(http.MethodGet, "/api/v1/products/product-1/recipes/2", nil),
		httptest.NewRequest(http.MethodGet, "/api/v1/products/product-1/recipe/cost", nil),
	}
	for _, req := range requests {
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("%s %s status = %d, want 200; body=%s", req.Method, req.URL.Path, rec.Code, rec.Body.String())
		}
	}
	if inventoryHits != len(requests) {
		t.Fatalf("inventory hits = %d, want %d", inventoryHits, len(requests))
	}
}

func TestBundleInventoryRoutesGoToInventoryService(t *testing.T) {
	productService := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/inventory/bundles") {
			t.Fatalf("bundle route reached product-service: %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"service":"product"}`))
	}))
	defer productService.Close()

	inventoryHits := 0
	inventoryService := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		inventoryHits++
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/inventory/bundles":
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/inventory/bundles":
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/inventory/bundles/bundle-1":
		case r.Method == http.MethodPut && r.URL.Path == "/api/v1/inventory/bundles/bundle-1":
		case r.Method == http.MethodDelete && r.URL.Path == "/api/v1/inventory/bundles/bundle-1":
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/inventory/bundles/bundle-1/cost":
		default:
			t.Fatalf("unexpected inventory-service request: %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"service":"inventory"}`))
	}))
	defer inventoryService.Close()

	e := echo.New()
	protected := e.Group("")
	protected.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Set("role", string(middleware.RoleManager))
			c.Set("tenant_id", "tenant-1")
			return next(c)
		}
	})
	registerIngredientInventoryRoutes(protected, inventoryService.URL)
	registerProductRoutes(protected, productService.URL)

	requests := []*http.Request{
		httptest.NewRequest(http.MethodGet, "/api/v1/inventory/bundles", nil),
		httptest.NewRequest(http.MethodPost, "/api/v1/inventory/bundles", strings.NewReader(`{}`)),
		httptest.NewRequest(http.MethodGet, "/api/v1/inventory/bundles/bundle-1", nil),
		httptest.NewRequest(http.MethodPut, "/api/v1/inventory/bundles/bundle-1", strings.NewReader(`{}`)),
		httptest.NewRequest(http.MethodDelete, "/api/v1/inventory/bundles/bundle-1", nil),
		httptest.NewRequest(http.MethodGet, "/api/v1/inventory/bundles/bundle-1/cost", nil),
	}
	for _, req := range requests {
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("%s %s status = %d, want 200; body=%s", req.Method, req.URL.Path, rec.Code, rec.Body.String())
		}
	}
	if inventoryHits != len(requests) {
		t.Fatalf("inventory hits = %d, want %d", inventoryHits, len(requests))
	}
}
