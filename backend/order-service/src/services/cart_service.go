package services

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/point-of-sale-system/order-service/src/models"
	"github.com/point-of-sale-system/order-service/src/repository"
)

type CartService struct {
	cartRepo        *repository.CartRepository
	reservationRepo *repository.ReservationRepository
	db              *sql.DB
}

func NewCartService(cartRepo *repository.CartRepository, reservationRepo *repository.ReservationRepository, db *sql.DB) *CartService {
	return &CartService{
		cartRepo:        cartRepo,
		reservationRepo: reservationRepo,
		db:              db,
	}
}

func (s *CartService) GetCart(ctx context.Context, tenantID, sessionID string) (*models.Cart, error) {
	cart, err := s.cartRepo.Get(ctx, tenantID, sessionID)
	if err != nil {
		return nil, err
	}

	// Validate and adjust cart items based on current stock availability
	if err := s.ValidateAndAdjustCart(ctx, cart); err != nil {
		return nil, err
	}

	return cart, nil
}

// ValidateAndAdjustCart validates all cart items against current stock availability
// and automatically adjusts quantities or removes items as needed
func (s *CartService) ValidateAndAdjustCart(ctx context.Context, cart *models.Cart) error {
	if cart == nil || len(cart.Items) == 0 {
		return nil
	}

	adjusted := false
	itemsToKeep := []models.CartItem{}

	for _, item := range cart.Items {
		// Get product stock from database
		var stockQty int
		query := `SELECT stock_quantity FROM products WHERE id = $1 AND tenant_id = $2 AND archived_at IS NULL`
		err := s.db.QueryRowContext(ctx, query, item.ProductID, cart.TenantID).Scan(&stockQty)
		if err == sql.ErrNoRows {
			// Product no longer exists or archived - remove from cart
			adjusted = true
			continue
		}
		if err != nil {
			return fmt.Errorf("failed to check product stock: %w", err)
		}

		// Get total reserved quantity for this product
		reservedQty, err := s.reservationRepo.GetTotalReservedQuantity(ctx, item.ProductID)
		if err != nil {
			return fmt.Errorf("failed to check reservations: %w", err)
		}

		// Calculate available stock
		availableStock := stockQty - reservedQty

		if availableStock <= 0 {
			// No stock available - remove item from cart
			adjusted = true
			continue
		}

		if item.Quantity > availableStock {
			// Adjust quantity to available stock
			item.Quantity = availableStock
			item.TotalPrice = item.Quantity * item.UnitPrice
			adjusted = true
		}

		itemsToKeep = append(itemsToKeep, item)
	}

	// Update cart if adjustments were made
	if adjusted {
		cart.Items = itemsToKeep
		if err := s.cartRepo.Save(ctx, cart); err != nil {
			return fmt.Errorf("failed to save adjusted cart: %w", err)
		}
	}

	return nil
}

func (s *CartService) AddItem(ctx context.Context, tenantID, sessionID, productID, productName string, quantity, unitPrice int) (*models.Cart, error) {
	cart, err := s.cartRepo.Get(ctx, tenantID, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get cart: %w", err)
	}

	// Calculate new quantity for this product
	newQuantity := quantity
	for _, item := range cart.Items {
		if item.ProductID == productID {
			newQuantity += item.Quantity
			break
		}
	}

	// Validate stock availability
	if err := s.validateStock(ctx, tenantID, productID, newQuantity); err != nil {
		return nil, err
	}

	found := false
	for i, item := range cart.Items {
		if item.ProductID == productID {
			cart.Items[i].Quantity += quantity
			cart.Items[i].TotalPrice = cart.Items[i].Quantity * cart.Items[i].UnitPrice
			found = true
			break
		}
	}

	if !found {
		cart.Items = append(cart.Items, models.CartItem{
			ProductID:   productID,
			ProductName: productName,
			Quantity:    quantity,
			UnitPrice:   unitPrice,
			TotalPrice:  quantity * unitPrice,
		})
	}

	if err := s.cartRepo.Save(ctx, cart); err != nil {
		return nil, fmt.Errorf("failed to save cart: %w", err)
	}

	return cart, nil
}

func (s *CartService) UpdateItem(ctx context.Context, tenantID, sessionID, productID string, quantity int) (*models.Cart, error) {
	cart, err := s.cartRepo.Get(ctx, tenantID, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get cart: %w", err)
	}

	// Validate stock availability if increasing quantity
	if quantity > 0 {
		if err := s.validateStock(ctx, tenantID, productID, quantity); err != nil {
			return nil, err
		}
	}

	found := false
	for i, item := range cart.Items {
		if item.ProductID == productID {
			if quantity <= 0 {
				cart.Items = append(cart.Items[:i], cart.Items[i+1:]...)
			} else {
				cart.Items[i].Quantity = quantity
				cart.Items[i].TotalPrice = cart.Items[i].Quantity * cart.Items[i].UnitPrice
			}
			found = true
			break
		}
	}

	if !found {
		return nil, fmt.Errorf("product not found in cart")
	}

	if err := s.cartRepo.Save(ctx, cart); err != nil {
		return nil, fmt.Errorf("failed to save cart: %w", err)
	}

	return cart, nil
}

func (s *CartService) RemoveItem(ctx context.Context, tenantID, sessionID, productID string) (*models.Cart, error) {
	cart, err := s.cartRepo.Get(ctx, tenantID, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get cart: %w", err)
	}

	for i, item := range cart.Items {
		if item.ProductID == productID {
			cart.Items = append(cart.Items[:i], cart.Items[i+1:]...)
			break
		}
	}

	if err := s.cartRepo.Save(ctx, cart); err != nil {
		return nil, fmt.Errorf("failed to save cart: %w", err)
	}

	return cart, nil
}

func (s *CartService) ClearCart(ctx context.Context, tenantID, sessionID string) error {
	return s.cartRepo.Delete(ctx, tenantID, sessionID)
}

// validateStock checks if the requested quantity is available (stock - active reservations)
func (s *CartService) validateStock(ctx context.Context, tenantID, productID string, requestedQty int) error {
	// Get product stock from database
	var stockQty int
	query := `SELECT stock_quantity FROM products WHERE id = $1 AND tenant_id = $2 AND archived_at IS NULL`
	err := s.db.QueryRowContext(ctx, query, productID, tenantID).Scan(&stockQty)
	if err == sql.ErrNoRows {
		return fmt.Errorf("product not found or unavailable")
	}
	if err != nil {
		return fmt.Errorf("failed to check product stock: %w", err)
	}

	// Get total reserved quantity for this product
	reservedQty, err := s.reservationRepo.GetTotalReservedQuantity(ctx, productID)
	if err != nil {
		return fmt.Errorf("failed to check reservations: %w", err)
	}

	// Calculate available stock
	availableStock := stockQty - reservedQty

	if requestedQty > availableStock {
		return fmt.Errorf("insufficient stock: only %d available (requested: %d)", availableStock, requestedQty)
	}

	return nil
}
