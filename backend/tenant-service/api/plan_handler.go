package api

import (
	"net/http"
	"os"
	"strconv"

	"github.com/labstack/echo/v4"
)

const (
	defaultMonthlyPriceIDR  = 299000
	defaultAnnualDiscountPct = 20
	defaultTrialDays         = 7
)

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}

// GetPublicPlan returns the current subscription plan pricing.
// This is a public endpoint — no authentication required.
// Values are read from environment variables so they can be updated
// without a code deploy. The billing-service will own this endpoint
// once it is deployed; this handler acts as the interim implementation.
func GetPublicPlan(c echo.Context) error {
	monthlyPriceIDR := getEnvInt("PLAN_MONTHLY_PRICE_IDR", defaultMonthlyPriceIDR)
	annualDiscountPct := getEnvInt("PLAN_ANNUAL_DISCOUNT_PCT", defaultAnnualDiscountPct)
	trialDays := getEnvInt("PLAN_TRIAL_DAYS", defaultTrialDays)

	annualTotal := int(float64(monthlyPriceIDR) * 12 * (1 - float64(annualDiscountPct)/100))

	return c.JSON(http.StatusOK, map[string]interface{}{
		"monthly_price_idr":  monthlyPriceIDR,
		"annual_discount_pct": annualDiscountPct,
		"annual_price_idr":   annualTotal,
		"trial_days":         trialDays,
	})
}
