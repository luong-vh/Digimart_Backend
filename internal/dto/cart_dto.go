// dto/cart_dto.go
package dto

import (
	"github.com/DucLove1/SE357-ShoppingManagement-BE/internal/model"
)

// Request DTOs
type AddCartItemRequest struct {
	ProductID string  `json:"product_id" validate:"required"`
	VariantID *string `json:"variant_id,omitempty"`
	Quantity  int     `json:"quantity" validate:"required"`
}

type UpdateCartItemRequest struct {
	ProductID string  `json:"product_id" validate:"required"`
	VariantID *string `json:"variant_id,omitempty"`
	Quantity  int     `json:"quantity" validate:"required,min=0"`
}

type AddCartItemsRequest struct {
	Items []AddCartItemRequest `json:"items" validate:"required,dive"`
}

type RemoveCartItemsRequest struct {
	Items []RemoveCartItemIdentifier `json:"items" validate:"required"`
}

type RemoveCartItemIdentifier struct {
	ProductID string  `json:"product_id" validate:"required"`
	VariantID *string `json:"variant_id,omitempty"`
}

// Response DTOs
type CartResponse struct {
	ID            string             `json:"id"`
	UserID        string             `json:"user_id"`
	Items         []CartItemResponse `json:"items"`
	TotalAmount   float64            `json:"total_amount"`
	TotalQuantity int                `json:"total_quantity"`
	CreatedAt     string             `json:"created_at"`
	UpdatedAt     string             `json:"updated_at"`
}

type CartItemResponse struct {
	ProductID string                  `json:"product_id"`
	VariantID *string                 `json:"variant_id,omitempty"`
	SellerID  string                  `json:"seller_id"`
	Quantity  int                     `json:"quantity"`
	AddedAt   string                  `json:"added_at"`
	Snapshot  *model.CartItemSnapshot `json:"snapshot,omitempty"`
	ItemTotal float64                 `json:"item_total"`
}

type CartValidationResponse struct {
	IsValid       bool              `json:"is_valid"`
	InvalidItems  []InvalidCartItem `json:"invalid_items"`
	ValidItems    []ValidCartItem   `json:"valid_items"`
	TotalAmount   float64           `json:"total_amount"`
	TotalQuantity int               `json:"total_quantity"`
}

type InvalidCartItem struct {
	ProductID      string  `json:"product_id"`
	VariantID      *string `json:"variant_id,omitempty"`
	Reason         string  `json:"reason"`
	AvailableStock int     `json:"available_stock,omitempty"`
	RequestedQty   int     `json:"requested_qty,omitempty"`
}

type ValidCartItem struct {
	ProductID   string  `json:"product_id"`
	VariantID   *string `json:"variant_id,omitempty"`
	Quantity    int     `json:"quantity"`
	Price       float64 `json:"price"`
	TotalAmount float64 `json:"total_amount"`
}

type CartStatsResponse struct {
	TotalCarts       int64   `json:"total_carts"`
	NonEmptyCarts    int64   `json:"non_empty_carts"`
	AverageItemCount float64 `json:"average_item_count"`
}

// Converter functions
func FromCart(cart *model.Cart) *CartResponse {
	if cart == nil {
		return nil
	}

	items := make([]CartItemResponse, len(cart.Items))
	var totalAmount float64
	var totalQuantity int

	for i, item := range cart.Items {
		itemTotal := item.GetItemTotal()
		items[i] = CartItemResponse{
			ProductID: item.ProductID.Hex(),
			VariantID: func() *string {
				if item.VariantID != nil {
					v := item.VariantID.Hex()
					return &v
				}
				return nil
			}(),
			SellerID:  item.SellerID.Hex(),
			Quantity:  item.Quantity,
			AddedAt:   item.AddedAt.Format("2006-01-02T15:04:05Z07:00"),
			Snapshot:  item.Snapshot,
			ItemTotal: itemTotal,
		}
		totalAmount += itemTotal
		totalQuantity += item.Quantity
	}

	return &CartResponse{
		ID:            cart.ID.Hex(),
		UserID:        cart.UserID.Hex(),
		Items:         items,
		TotalAmount:   totalAmount,
		TotalQuantity: totalQuantity,
		CreatedAt:     cart.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:     cart.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}
