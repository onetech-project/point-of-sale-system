package services

import (
	"context"
	"database/sql"
	"time"

	"github.com/pos/analytics-service/src/models"
	"github.com/pos/analytics-service/src/repository"
	"github.com/pos/analytics-service/src/utils"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
)

// AnalyticsService orchestrates analytics operations with caching
type AnalyticsService struct {
	salesRepo     *repository.SalesRepository
	productRepo   *repository.ProductRepository
	customerRepo  *repository.CustomerRepository
	cache         *CacheService
	currentTTL    time.Duration
	historicalTTL time.Duration
}

// NewAnalyticsService creates a new analytics service
func NewAnalyticsService(db *sql.DB, redisClient *redis.Client, encryptor utils.Encryptor, currentTTL, historicalTTL time.Duration) *AnalyticsService {
	return &AnalyticsService{
		salesRepo:     repository.NewSalesRepository(db),
		productRepo:   repository.NewProductRepository(db),
		customerRepo:  repository.NewCustomerRepository(db, encryptor),
		cache:         NewCacheService(redisClient),
		currentTTL:    currentTTL,
		historicalTTL: historicalTTL,
	}
}

// GetSalesOverview returns sales metrics, daily sales, and category breakdown with caching
func (s *AnalyticsService) GetSalesOverview(ctx context.Context, tenantID string, timeRange models.TimeRange, startDate, endDate *time.Time) (*models.SalesOverviewResponse, error) {
	// Determine date range
	var start, end time.Time
	var err error

	if timeRange == models.TimeRangeCustom && startDate != nil && endDate != nil {
		start = *startDate
		end = *endDate
	} else {
		start, end, err = timeRange.GetDateRange()
		if err != nil {
			return nil, err
		}
	}

	// Try to get from cache
	cacheKey := GenerateKeyWithTimeRange(tenantID, string(timeRange), "sales_overview")
	var response models.SalesOverviewResponse
	if err := s.cache.Get(ctx, cacheKey, &response); err == nil {
		log.Debug().Str("cache_key", cacheKey).Msg("Cache hit for sales overview")
		return &response, nil
	}

	// Cache miss - query database
	log.Debug().Str("cache_key", cacheKey).Msg("Cache miss for sales overview")

	// Get sales metrics
	metrics, err := s.salesRepo.GetSalesMetrics(ctx, tenantID, start, end)
	if err != nil {
		return nil, err
	}

	// Get daily sales data
	dailySales, err := s.salesRepo.GetDailySales(ctx, tenantID, start, end)
	if err != nil {
		return nil, err
	}

	// Get category breakdown
	categoryBreakdown, err := s.salesRepo.GetCategoryBreakdown(ctx, tenantID, start, end)
	if err != nil {
		return nil, err
	}

	response = models.SalesOverviewResponse{
		Metrics:           *metrics,
		SalesChart:        dailySales,
		CategoryBreakdown: categoryBreakdown,
	}

	// Cache the response with appropriate TTL
	ttl := timeRange.GetCacheTTL(s.currentTTL, s.historicalTTL)
	if err := s.cache.Set(ctx, cacheKey, response, ttl); err != nil {
		log.Warn().Err(err).Msg("Failed to cache sales overview")
	}

	return &response, nil
}

// GetTopProducts returns top and bottom products by revenue and quantity with caching
func (s *AnalyticsService) GetTopProducts(ctx context.Context, tenantID string, timeRange models.TimeRange, startDate, endDate *time.Time, limit int) (*models.TopProductsResponse, error) {
	// Determine date range
	var start, end time.Time
	var err error

	if timeRange == models.TimeRangeCustom && startDate != nil && endDate != nil {
		start = *startDate
		end = *endDate
	} else {
		start, end, err = timeRange.GetDateRange()
		if err != nil {
			return nil, err
		}
	}

	// Try to get from cache
	cacheKey := GenerateKeyWithTimeRange(tenantID, string(timeRange), "top_products")
	var response models.TopProductsResponse
	if err := s.cache.Get(ctx, cacheKey, &response); err == nil {
		log.Debug().Str("cache_key", cacheKey).Msg("Cache hit for top products")
		return &response, nil
	}

	// Cache miss - query database
	log.Debug().Str("cache_key", cacheKey).Msg("Cache miss for top products")

	// Query all rankings in parallel
	topByRevenueChan := make(chan []models.ProductRanking, 1)
	topByQuantityChan := make(chan []models.ProductRanking, 1)
	bottomByRevenueChan := make(chan []models.ProductRanking, 1)
	bottomByQuantityChan := make(chan []models.ProductRanking, 1)
	errChan := make(chan error, 4)

	go func() {
		products, err := s.productRepo.GetTopProductsByRevenue(ctx, tenantID, start, end, limit)
		if err != nil {
			errChan <- err
			return
		}
		topByRevenueChan <- products
	}()

	go func() {
		products, err := s.productRepo.GetTopProductsByQuantity(ctx, tenantID, start, end, limit)
		if err != nil {
			errChan <- err
			return
		}
		topByQuantityChan <- products
	}()

	go func() {
		products, err := s.productRepo.GetBottomProductsByRevenue(ctx, tenantID, start, end, limit)
		if err != nil {
			errChan <- err
			return
		}
		bottomByRevenueChan <- products
	}()

	go func() {
		products, err := s.productRepo.GetBottomProductsByQuantity(ctx, tenantID, start, end, limit)
		if err != nil {
			errChan <- err
			return
		}
		bottomByQuantityChan <- products
	}()

	// Wait for all goroutines to complete
	for i := 0; i < 4; i++ {
		select {
		case err := <-errChan:
			return nil, err
		case response.TopByRevenue = <-topByRevenueChan:
		case response.TopByQuantity = <-topByQuantityChan:
		case response.BottomByRevenue = <-bottomByRevenueChan:
		case response.BottomByQuantity = <-bottomByQuantityChan:
		}
	}

	// Cache the response
	ttl := timeRange.GetCacheTTL(s.currentTTL, s.historicalTTL)
	if err := s.cache.Set(ctx, cacheKey, response, ttl); err != nil {
		log.Warn().Err(err).Msg("Failed to cache top products")
	}

	return &response, nil
}

// GetTopCustomers returns top customers by spending and order count with caching
func (s *AnalyticsService) GetTopCustomers(ctx context.Context, tenantID string, timeRange models.TimeRange, startDate, endDate *time.Time, limit int) (*models.TopCustomersResponse, error) {
	// Determine date range
	var start, end time.Time
	var err error

	if timeRange == models.TimeRangeCustom && startDate != nil && endDate != nil {
		start = *startDate
		end = *endDate
	} else {
		start, end, err = timeRange.GetDateRange()
		if err != nil {
			return nil, err
		}
	}

	// Try to get from cache
	cacheKey := GenerateKeyWithTimeRange(tenantID, string(timeRange), "top_customers")
	var response models.TopCustomersResponse
	if err := s.cache.Get(ctx, cacheKey, &response); err == nil {
		log.Debug().Str("cache_key", cacheKey).Msg("Cache hit for top customers")
		return &response, nil
	}

	// Cache miss - query database
	log.Debug().Str("cache_key", cacheKey).Msg("Cache miss for top customers")

	// Query both rankings in parallel
	topBySpendingChan := make(chan []models.CustomerRanking, 1)
	topByOrdersChan := make(chan []models.CustomerRanking, 1)
	errChan := make(chan error, 2)

	go func() {
		customers, err := s.customerRepo.GetTopCustomersBySpending(ctx, tenantID, start, end, limit)
		if err != nil {
			errChan <- err
			return
		}
		topBySpendingChan <- customers
	}()

	go func() {
		customers, err := s.customerRepo.GetTopCustomersByOrders(ctx, tenantID, start, end, limit)
		if err != nil {
			errChan <- err
			return
		}
		topByOrdersChan <- customers
	}()

	// Wait for both goroutines to complete
	for i := 0; i < 2; i++ {
		select {
		case err := <-errChan:
			return nil, err
		case response.TopBySpending = <-topBySpendingChan:
		case response.TopByOrders = <-topByOrdersChan:
		}
	}

	// Cache the response
	ttl := timeRange.GetCacheTTL(s.currentTTL, s.historicalTTL)
	if err := s.cache.Set(ctx, cacheKey, response, ttl); err != nil {
		log.Warn().Err(err).Msg("Failed to cache top customers")
	}

	return &response, nil
}

// GetSalesTrend returns time series data for sales with caching
func (s *AnalyticsService) GetSalesTrend(ctx context.Context, tenantID string, startDate, endDate time.Time, granularity string) (*models.SalesTrendResponse, error) {
	// Generate cache key
	cacheKey := GenerateKeyWithTimeRange(tenantID, granularity, "sales_trend_"+startDate.Format("20060102")+"_"+endDate.Format("20060102"))

	// Try cache first
	var cached models.SalesTrendResponse
	if err := s.cache.Get(ctx, cacheKey, &cached); err == nil {
		log.Debug().Str("cache_key", cacheKey).Msg("Sales trend cache hit")
		return &cached, nil
	}

	log.Debug().Str("cache_key", cacheKey).Msg("Sales trend cache miss")

	// Fetch from repository
	revenueData, ordersData, err := s.salesRepo.GetSalesTrend(ctx, tenantID, startDate, endDate, granularity)
	if err != nil {
		return nil, err
	}

	response := models.SalesTrendResponse{
		Period:      granularity,
		StartDate:   startDate.Format("2006-01-02"),
		EndDate:     endDate.Format("2006-01-02"),
		RevenueData: revenueData,
		OrdersData:  ordersData,
	}

	// Determine TTL: use historical TTL for past data, current TTL for recent data
	ttl := s.historicalTTL
	now := time.Now()
	if endDate.After(now.AddDate(0, 0, -7)) {
		ttl = s.currentTTL
	}

	// Cache the result
	if err := s.cache.Set(ctx, cacheKey, response, ttl); err != nil {
		log.Warn().Err(err).Msg("Failed to cache sales trend")
	}

	return &response, nil
}
