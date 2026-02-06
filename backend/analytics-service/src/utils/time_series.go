package utils

import (
	"time"
)

// TimeSeriesPoint represents a single point in a time series
type TimeSeriesPoint struct {
	Date  time.Time
	Label string
	Value float64
}

// GenerateTimeSeriesLabels generates labels for a time period
func GenerateTimeSeriesLabels(start, end time.Time, granularity string) []string {
	labels := []string{}
	current := start

	switch granularity {
	case "hour":
		for current.Before(end) || current.Equal(end) {
			labels = append(labels, current.Format("15:04"))
			current = current.Add(time.Hour)
		}
	case "day":
		for current.Before(end) || current.Equal(end) {
			labels = append(labels, current.Format("Jan 02"))
			current = current.AddDate(0, 0, 1)
		}
	case "week":
		for current.Before(end) || current.Equal(end) {
			labels = append(labels, current.Format("Week of Jan 02"))
			current = current.AddDate(0, 0, 7)
		}
	case "month":
		for current.Before(end) || current.Equal(end) {
			labels = append(labels, current.Format("Jan 2006"))
			current = current.AddDate(0, 1, 0)
		}
	default: // day by default
		for current.Before(end) || current.Equal(end) {
			labels = append(labels, current.Format("Jan 02"))
			current = current.AddDate(0, 0, 1)
		}
	}

	return labels
}

// GroupOrdersByPeriod groups orders by time period
func GroupOrdersByPeriod(start, end time.Time, granularity string) []time.Time {
	periods := []time.Time{}
	current := start

	switch granularity {
	case "hour":
		for current.Before(end) || current.Equal(end) {
			periods = append(periods, current)
			current = current.Add(time.Hour)
		}
	case "day":
		for current.Before(end) || current.Equal(end) {
			periods = append(periods, current)
			current = current.AddDate(0, 0, 1)
		}
	case "week":
		for current.Before(end) || current.Equal(end) {
			periods = append(periods, current)
			current = current.AddDate(0, 0, 7)
		}
	case "month":
		for current.Before(end) || current.Equal(end) {
			periods = append(periods, current)
			current = current.AddDate(0, 1, 0)
		}
	default: // day by default
		for current.Before(end) || current.Equal(end) {
			periods = append(periods, current)
			current = current.AddDate(0, 0, 1)
		}
	}

	return periods
}

// DetermineGranularity automatically determines the best granularity based on date range
func DetermineGranularity(start, end time.Time) string {
	duration := end.Sub(start)

	hours := duration.Hours()

	if hours <= 24 {
		return "hour"
	} else if hours <= 24*7 {
		return "day"
	} else if hours <= 24*60 {
		return "day"
	} else {
		return "month"
	}
}

// NormalizeToStartOfPeriod normalizes a timestamp to the start of its period
func NormalizeToStartOfPeriod(t time.Time, granularity string) time.Time {
	loc := t.Location()

	switch granularity {
	case "hour":
		return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), 0, 0, 0, loc)
	case "day":
		return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, loc)
	case "week":
		// Normalize to Monday
		weekday := int(t.Weekday())
		if weekday == 0 {
			weekday = 7 // Sunday = 7
		}
		daysToMonday := weekday - 1
		monday := t.AddDate(0, 0, -daysToMonday)
		return time.Date(monday.Year(), monday.Month(), monday.Day(), 0, 0, 0, 0, loc)
	case "month":
		return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, loc)
	default:
		return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, loc)
	}
}
