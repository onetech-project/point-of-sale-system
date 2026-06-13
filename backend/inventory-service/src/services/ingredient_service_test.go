package services

import (
	"testing"

	"github.com/shopspring/decimal"
)

func TestWeightedAverageCost(t *testing.T) {
	currentStock := decimal.RequireFromString("10")
	currentAvg := decimal.RequireFromString("5.00")
	addedQty := decimal.RequireFromString("5")
	addedCost := decimal.RequireFromString("8.00")

	got := weightedAverageCost(currentStock, currentAvg, addedQty, addedCost)
	want := decimal.RequireFromString("6")

	if !got.Equal(want) {
		t.Fatalf("weightedAverageCost = %s, want %s", got, want)
	}
}
