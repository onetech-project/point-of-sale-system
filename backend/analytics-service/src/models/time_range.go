package models

import (
	"fmt"
	"time"
)

// TimeRange represents the time period for analytics queries
type TimeRange string

const (
	TimeRangeToday      TimeRange = "today"
	TimeRangeYesterday  TimeRange = "yesterday"
	TimeRangeThisWeek   TimeRange = "this_week"
	TimeRangeLastWeek   TimeRange = "last_week"
	TimeRangeThisMonth  TimeRange = "this_month"
	TimeRangeLastMonth  TimeRange = "last_month"
	TimeRangeThisYear   TimeRange = "this_year"
	TimeRangeLast30Days TimeRange = "last_30_days"
	TimeRangeLast90Days TimeRange = "last_90_days"
	TimeRangeCustom     TimeRange = "custom"
)

// GetDateRange returns the start and end dates for a time range
func (tr TimeRange) GetDateRange() (start, end time.Time, err error) {
	now := time.Now()
	loc := now.Location()

	// Normalize to start of day
	startOfToday := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
	endOfToday := time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 999999999, loc)

	switch tr {
	case TimeRangeToday:
		return startOfToday, endOfToday, nil

	case TimeRangeYesterday:
		yesterday := startOfToday.AddDate(0, 0, -1)
		return yesterday, yesterday.Add(24*time.Hour - time.Nanosecond), nil

	case TimeRangeThisWeek:
		// Start from Monday
		weekday := int(now.Weekday())
		if weekday == 0 {
			weekday = 7 // Sunday = 7
		}
		monday := startOfToday.AddDate(0, 0, -(weekday - 1))
		return monday, endOfToday, nil

	case TimeRangeLastWeek:
		weekday := int(now.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		lastMonday := startOfToday.AddDate(0, 0, -(weekday + 6))
		lastSunday := lastMonday.AddDate(0, 0, 6).Add(24*time.Hour - time.Nanosecond)
		return lastMonday, lastSunday, nil

	case TimeRangeThisMonth:
		firstDay := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, loc)
		return firstDay, endOfToday, nil

	case TimeRangeLastMonth:
		firstDayThisMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, loc)
		firstDayLastMonth := firstDayThisMonth.AddDate(0, -1, 0)
		lastDayLastMonth := firstDayThisMonth.Add(-time.Nanosecond)
		return firstDayLastMonth, lastDayLastMonth, nil

	case TimeRangeThisYear:
		firstDay := time.Date(now.Year(), 1, 1, 0, 0, 0, 0, loc)
		return firstDay, endOfToday, nil

	case TimeRangeLast30Days:
		start := startOfToday.AddDate(0, 0, -29) // Last 30 days including today
		return start, endOfToday, nil

	case TimeRangeLast90Days:
		start := startOfToday.AddDate(0, 0, -89) // Last 90 days including today
		return start, endOfToday, nil

	default:
		return time.Time{}, time.Time{}, fmt.Errorf("unsupported time range: %s", tr)
	}
}

// IsValid checks if the time range value is valid
func (tr TimeRange) IsValid() bool {
	switch tr {
	case TimeRangeToday, TimeRangeYesterday, TimeRangeThisWeek, TimeRangeLastWeek,
		TimeRangeThisMonth, TimeRangeLastMonth, TimeRangeThisYear,
		TimeRangeLast30Days, TimeRangeLast90Days, TimeRangeCustom:
		return true
	default:
		return false
	}
}

// GetCacheTTL returns the appropriate cache TTL for this time range
func (tr TimeRange) GetCacheTTL(currentMonthTTL, historicalTTL time.Duration) time.Duration {
	now := time.Now()
	start, _, err := tr.GetDateRange()
	if err != nil {
		return currentMonthTTL
	}

	// If the range includes the current month, use shorter TTL
	if start.Year() == now.Year() && start.Month() == now.Month() {
		return currentMonthTTL
	}

	// Historical data changes less frequently
	return historicalTTL
}
