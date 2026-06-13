package models

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/shopspring/decimal"
)

type Number struct {
	decimal.Decimal
}

func NewNumber(value decimal.Decimal) Number {
	return Number{Decimal: value}
}

func ZeroNumber() Number {
	return NewNumber(decimal.Zero)
}

func (n *Number) UnmarshalJSON(data []byte) error {
	data = bytes.TrimSpace(data)
	if len(data) == 0 || bytes.Equal(data, []byte("null")) {
		n.Decimal = decimal.Zero
		return nil
	}
	if data[0] == '"' {
		var s string
		if err := json.Unmarshal(data, &s); err != nil {
			return err
		}
		value, err := decimal.NewFromString(s)
		if err != nil {
			return fmt.Errorf("invalid decimal: %w", err)
		}
		n.Decimal = value
		return nil
	}
	value, err := decimal.NewFromString(string(data))
	if err != nil {
		return fmt.Errorf("invalid decimal: %w", err)
	}
	n.Decimal = value
	return nil
}

func (n Number) MarshalJSON() ([]byte, error) {
	return []byte(n.Decimal.String()), nil
}
