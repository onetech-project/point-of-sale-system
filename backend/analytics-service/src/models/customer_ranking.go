package models

// CustomerRanking represents a customer's ranking by spending or order count
type CustomerRanking struct {
	Name         string  `json:"name"`  // Masked for display
	Phone        string  `json:"phone"` // Masked for display
	Email        string  `json:"email"` // Masked for display
	OrderCount   int64   `json:"order_count"`
	TotalSpent   float64 `json:"total_spent"`
	AverageOrder float64 `json:"average_order"`
}

// TopCustomersResponse contains top customers by spending
type TopCustomersResponse struct {
	TopBySpending []CustomerRanking `json:"top_by_spending"`
	TopByOrders   []CustomerRanking `json:"top_by_orders"`
}
