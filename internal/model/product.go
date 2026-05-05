// model/product.go
package model

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Product struct {
	ID         primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	CategoryID primitive.ObjectID `bson:"category_id" json:"category_id"`

	// Basic Info
	Name        string `bson:"name" json:"name"`
	Description string `bson:"description" json:"description"`
	Brand       string `bson:"brand,omitempty" json:"brand,omitempty"` // Thương hiệu

	IsHasVariant    bool    `bson:"is_has_variant" json:"is_has_variant"`
	BasePrice       float64 `bson:"base_price" json:"base_price"`
	DiscountPercent float64 `bson:"discount_percent" json:"discount_percent"`

	// Media
	Thumbnail Image   `bson:"thumbnail" json:"thumbnail"`
	Images    []Image `bson:"images" json:"images"`
	Videos    []Video `bson:"videos,omitempty" json:"videos,omitempty"`

	StockQuantity *int `bson:"stock_quantity,omitempty" json:"stock_quantity"`

	// Stats
	SoldCount   int     `bson:"sold_count" json:"sold_count"`
	Rating      float64 `bson:"rating" json:"rating"`
	RatingCount int     `bson:"rating_count" json:"rating_count"`

	// Variants
	Variants []ProductVariant `bson:"variants,omitempty" json:"variants,omitempty"`

	// Status
	Status ProductStatus `bson:"status" json:"status"`

	// Timestamps
	CreatedAt time.Time  `bson:"created_at" json:"created_at"`
	UpdatedAt time.Time  `bson:"updated_at" json:"updated_at"`
	DeletedAt *time.Time `bson:"deleted_at,omitempty" json:"deleted_at,omitempty"`
}

type ProductStatus string

const (
	ProductStatusDraft  ProductStatus = "draft"  // Nháp, chưa publish
	ProductStatusActive ProductStatus = "active" // Đang bán
)

type ProductVariant struct {
	ID              primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Title           string             `bson:"title" json:"title"`
	Description     string             `bson:"description" json:"description"`
	PriceAdjustment float64            `bson:"price_adjustment" json:"price_adjustment"`
	FinalPrice      float64            `bson:"final_price" json:"final_price"`
	StockQuantity   int                `bson:"stock_quantity" json:"stock_quantity"`
}
