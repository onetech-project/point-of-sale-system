package models

// CartItem represents an item in the guest's shopping cart
type CartItem struct {
	ProductID   string `json:"product_id"`
	Quantity    int    `json:"quantity"`
	ProductName string `json:"product_name"`
	UnitPrice   int    `json:"unit_price"`
	TotalPrice  int    `json:"total_price"`
}

// Cart represents the shopping cart stored in Redis
type Cart struct {
	TenantID  string     `json:"tenant_id"`
	SessionID string     `json:"session_id"`
	Items     []CartItem `json:"items"`
	UpdatedAt string     `json:"updated_at"`
}

// GetTotal calculates the total cart amount
func (c *Cart) GetTotal() int {
	total := 0
	for _, item := range c.Items {
		total += item.TotalPrice
	}
	return total
}

// GetItemCount returns total number of items in cart
func (c *Cart) GetItemCount() int {
	count := 0
	for _, item := range c.Items {
		count += item.Quantity
	}
	return count
}
