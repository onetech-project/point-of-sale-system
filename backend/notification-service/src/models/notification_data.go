package models

// StaffNotificationData contains the data for staff order notification emails
type StaffNotificationData struct {
	OrderID         string                  `json:"order_id"`
	OrderReference  string                  `json:"order_reference"`
	TransactionID   string                  `json:"transaction_id"`
	CustomerName    string                  `json:"customer_name"`
	CustomerEmail   string                  `json:"customer_email,omitempty"`
	CustomerPhone   string                  `json:"customer_phone"`
	DeliveryType    string                  `json:"delivery_type"`
	DeliveryAddress string                  `json:"delivery_address,omitempty"`
	TableNumber     string                  `json:"table_number,omitempty"`
	Items           []StaffNotificationItem `json:"items"`
	SubtotalAmount  string                  `json:"subtotal_amount"`
	DeliveryFee     string                  `json:"delivery_fee,omitempty"`
	TotalAmount     string                  `json:"total_amount"`
	PaymentMethod   string                  `json:"payment_method"`
	PaidAt          string                  `json:"paid_at"`
}

// StaffNotificationItem represents an order item in staff notification
type StaffNotificationItem struct {
	ProductName string `json:"product_name"`
	Quantity    int    `json:"quantity"`
	UnitPrice   string `json:"unit_price"`
	TotalPrice  string `json:"total_price"`
}

// CustomerReceiptData contains the data for customer email receipt
type CustomerReceiptData struct {
	OrderReference    string                `json:"order_reference"`
	CustomerName      string                `json:"customer_name"`
	CustomerEmail     string                `json:"customer_email"`
	DeliveryType      string                `json:"delivery_type"`
	DeliveryAddress   string                `json:"delivery_address,omitempty"`
	TableNumber       string                `json:"table_number,omitempty"`
	Items             []CustomerReceiptItem `json:"items"`
	SubtotalAmount    string                `json:"subtotal_amount"`
	DeliveryFee       string                `json:"delivery_fee,omitempty"`
	TotalAmount       string                `json:"total_amount"`
	PaymentMethod     string                `json:"payment_method"`
	PaidAt            string                `json:"paid_at"`
	ShowPaidWatermark bool                  `json:"show_paid_watermark"`
}

// CustomerReceiptItem represents an order item in customer receipt
type CustomerReceiptItem struct {
	ProductName string `json:"product_name"`
	Quantity    int    `json:"quantity"`
	UnitPrice   string `json:"unit_price"`
	TotalPrice  string `json:"total_price"`
}
