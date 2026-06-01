package services

import "github.com/pos/billing-service/src/utils"

const (
	defaultMonthlyPriceIDR   = 299000
	defaultAnnualDiscountPct = 20
	defaultTrialDays         = 7
	defaultGracePeriodDays   = 7
	defaultRetentionDays     = 30
)

// PublicPlan is the public subscription pricing payload.
type PublicPlan struct {
	MonthlyPriceIDR   int `json:"monthly_price_idr"`
	AnnualDiscountPct int `json:"annual_discount_pct"`
	AnnualPriceIDR    int `json:"annual_price_idr"`
	TrialDays         int `json:"trial_days"`
}

// GetPublicPlanFromEnv returns subscription pricing from billing-service config.
func GetPublicPlanFromEnv() PublicPlan {
	monthlyPriceIDR := utils.GetEnvInt("PLAN_MONTHLY_PRICE_IDR", defaultMonthlyPriceIDR)
	annualDiscountPct := utils.GetEnvInt("PLAN_ANNUAL_DISCOUNT_PCT", defaultAnnualDiscountPct)

	return PublicPlan{
		MonthlyPriceIDR:   monthlyPriceIDR,
		AnnualDiscountPct: annualDiscountPct,
		AnnualPriceIDR:    computeAnnualAmount(monthlyPriceIDR, annualDiscountPct),
		TrialDays:         utils.GetEnvInt("PLAN_TRIAL_DAYS", defaultTrialDays),
	}
}

// GetGracePeriodDaysFromEnv returns the subscription grace-period length.
func GetGracePeriodDaysFromEnv() int {
	return utils.GetEnvInt("PLAN_GRACE_PERIOD_DAYS", defaultGracePeriodDays)
}

// GetRetentionDaysFromEnv returns the operational data retention window.
func GetRetentionDaysFromEnv() int {
	return utils.GetEnvInt("PLAN_RETENTION_DAYS", defaultRetentionDays)
}

func computeAnnualAmount(monthlyPriceIDR, annualDiscountPct int) int {
	annual := monthlyPriceIDR * 12
	discount := annual * annualDiscountPct / 100
	return annual - discount
}
