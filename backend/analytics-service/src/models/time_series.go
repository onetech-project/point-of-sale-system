package models

// TimeSeriesData represents a single data point in a time series chart
type TimeSeriesData struct {
	Date   string  `json:"date"`             // ISO 8601 date (YYYY-MM-DD)
	Label  string  `json:"label"`            // Human-readable label (e.g., "Jan 15", "Week 3")
	Value  float64 `json:"value"`            // Metric value (revenue, orders, etc.)
	Value2 float64 `json:"value2,omitempty"` // Optional second metric for dual-axis charts
}

// SalesTrendResponse represents the response for the sales trend endpoint
type SalesTrendResponse struct {
	Period      string           `json:"period"`       // e.g., "daily", "weekly", "monthly"
	StartDate   string           `json:"start_date"`   // ISO 8601 date
	EndDate     string           `json:"end_date"`     // ISO 8601 date
	RevenueData []TimeSeriesData `json:"revenue_data"` // Revenue time series
	OrdersData  []TimeSeriesData `json:"orders_data"`  // Order count time series
}

// TimeSeriesRequest represents query parameters for time series data
type TimeSeriesRequest struct {
	Granularity string `query:"granularity"` // "daily", "weekly", "monthly", "quarterly", "yearly"
	StartDate   string `query:"start_date"`  // ISO 8601 date (YYYY-MM-DD)
	EndDate     string `query:"end_date"`    // ISO 8601 date (YYYY-MM-DD)
}

// Validate checks if the time series request is valid
func (r *TimeSeriesRequest) Validate() error {
	validGranularities := map[string]bool{
		"daily":     true,
		"weekly":    true,
		"monthly":   true,
		"quarterly": true,
		"yearly":    true,
	}

	if !validGranularities[r.Granularity] {
		return ErrInvalidGranularity
	}

	// Additional validation for date formats and ranges can be added here

	return nil
}

// Common validation errors
var (
	ErrInvalidGranularity   = &ValidationError{Field: "granularity", Message: "must be one of: daily, weekly, monthly, quarterly, yearly"}
	ErrInvalidDateRange     = &ValidationError{Field: "date_range", Message: "start_date must be before end_date"}
	ErrFutureDateNotAllowed = &ValidationError{Field: "date", Message: "dates in the future are not allowed"}
)

// ValidationError represents a validation error
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func (e *ValidationError) Error() string {
	return e.Field + ": " + e.Message
}
