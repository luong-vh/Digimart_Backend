// model/cart.go
package model

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Cart struct {
	ID     primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID primitive.ObjectID `bson:"user_id" json:"user_id"`
	Items  []CartItem         `bson:"items" json:"items"`

	CreatedAt time.Time `bson:"created_at" json:"created_at"`
	UpdatedAt time.Time `bson:"updated_at" json:"updated_at"`
}

type CartItem struct {
	ID        primitive.ObjectID  `bson:"_id,omitempty" json:"id"`
	ProductID primitive.ObjectID  `bson:"product_id" json:"product_id"`
	VariantID *primitive.ObjectID `bson:"variant_id,omitempty" json:"variant_id,omitempty"`
	Quantity  int                 `bson:"quantity" json:"quantity"`

	AddedAt   time.Time `bson:"added_at" json:"added_at"`
	UpdatedAt time.Time `bson:"updated_at,omitempty" json:"updated_at,omitempty"`
}

type CartItemSnapshot struct {
	ProductID   primitive.ObjectID `bson:"product_id" json:"product_id"`
	ProductName string             `bson:"product_name" json:"product_name"`
	Brand       string             `bson:"brand,omitempty" json:"brand,omitempty"`
	Thumbnail   Image              `bson:"thumbnail" json:"thumbnail"`

	VariantID    *primitive.ObjectID `bson:"variant_id,omitempty" json:"variant_id,omitempty"`
	VariantTitle *string             `bson:"variant_title,omitempty" json:"variant_title,omitempty"`

	BasePrice       float64 `bson:"base_price" json:"base_price"`
	DiscountPercent float64 `bson:"discount_percent" json:"discount_percent"`
	PriceAdjustment float64 `bson:"price_adjustment,omitempty" json:"price_adjustment,omitempty"`
	FinalPrice      float64 `bson:"final_price" json:"final_price"`

	StockQuantity int  `bson:"stock_quantity" json:"stock_quantity"`
	IsAvailable   bool `bson:"is_available" json:"is_available"`
}
