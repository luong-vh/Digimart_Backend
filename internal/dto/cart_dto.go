// dto/cart_dto.go
package dto

import (
	"github.com/luong-vh/Digimart_Backend/internal/model"
)

// Request DTOs
type AddCartItemRequest struct {
	ProductID string `json:"product_id" validate:"required"`
	VariantID string `json:"variant_id,omitempty"`
	Quantity  int    `json:"quantity" validate:"required,min=1"`
}

type UpdateCartItemRequest struct {
	ItemID   string `json:"item_id" validate:"required"`
	Quantity int    `json:"quantity" validate:"required,min=0"`
}

type AddCartItemsRequest struct {
	Items []AddCartItemRequest `json:"items" validate:"required,dive"`
}

type RemoveCartItemsRequest struct {
	Items []RemoveCartItemIdentifier `json:"items" validate:"required"`
}

type RemoveCartItemIdentifier struct {
	ID string `json:"id" validate:"required"`
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
	ID          string                  `json:"id"`
	ProductID   string                  `json:"product_id"`
	VariantID   *string                 `json:"variant_id,omitempty"`
	Quantity    int                     `json:"quantity"`
	IsAvailable bool                    `json:"is_available"`
	AddedAt     string                  `json:"added_at"`
	Snapshot    *model.CartItemSnapshot `json:"snapshot,omitempty"`
}

// CartValidationResult for checkout validation
type CartValidationResult struct {
	IsValid      bool                    `json:"is_valid"`
	InvalidItems []CartItemInvalidReason `json:"invalid_items"` // luôn trả về array, không null
	TotalPrice   float64                 `json:"total_price"`
	TotalItems   int                     `json:"total_items"`
	ValidItems   int                     `json:"valid_items"`
}

type CartItemInvalidReason struct {
	ItemID string `json:"item_id"`
	Reason string `json:"reason"`
}
type ValidatedCartItem struct {
	ItemID      string  `json:"item_id"`
	ProductID   string  `json:"product_id"`
	VariantID   *string `json:"variant_id,omitempty"`
	Quantity    int     `json:"quantity"`
	Price       float64 `json:"price"`
	TotalAmount float64 `json:"total_amount"`
	IsAvailable bool    `json:"is_available"`
	Reason      string  `json:"reason,omitempty"` // If not available
}
