package models

import "time"

type OnboardingProgress struct {
	Completed   bool       `json:"completed"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
}

type CompleteOnboardingRequest struct {
	TourKey     string `json:"tour_key"`
	TourVersion int    `json:"tour_version"`
}
