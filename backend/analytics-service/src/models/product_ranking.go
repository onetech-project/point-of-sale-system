package models

import "github.com/google/uuid"

// ProductRanking represents a product's ranking by sales or quantity
type ProductRanking struct {
	ProductID    uuid.UUID `json:"product_id"`
	Name         string    `json:"name"`
	SKU          string    `json:"sku"`
	QuantitySold int64     `json:"quantity_sold"`
	Revenue      float64   `json:"revenue"`
	ImageURL     string    `json:"image_url,omitempty"`
	CategoryName string    `json:"category_name,omitempty"`
}

// TopProductsResponse contains top and bottom products by different metrics
type TopProductsResponse struct {
	TopByRevenue     []ProductRanking `json:"top_by_revenue"`
	TopByQuantity    []ProductRanking `json:"top_by_quantity"`
	BottomByRevenue  []ProductRanking `json:"bottom_by_revenue"`
	BottomByQuantity []ProductRanking `json:"bottom_by_quantity"`
}
