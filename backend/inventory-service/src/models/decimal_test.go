package models

import (
	"encoding/json"
	"testing"

	"github.com/shopspring/decimal"
)

func TestNumberJSONUsesNumericDecimals(t *testing.T) {
	var n Number
	if err := json.Unmarshal([]byte(`12.345600`), &n); err != nil {
		t.Fatalf("unmarshal number: %v", err)
	}
	if !n.Equal(decimal.RequireFromString("12.3456")) {
		t.Fatalf("number = %s, want 12.3456", n.String())
	}

	data, err := json.Marshal(NewNumber(decimal.RequireFromString("12.3400")))
	if err != nil {
		t.Fatalf("marshal number: %v", err)
	}
	if string(data) != "12.34" {
		t.Fatalf("json = %s, want 12.34", data)
	}
}
