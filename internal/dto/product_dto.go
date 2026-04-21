// dto/product_dto.go
package dto

import (
	"github.com/DucLove1/SE357-ShoppingManagement-BE/internal/model"
)

// Request DTOs
type CreateProductRequest struct {
	CategoryID    string                 `json:"category_id" validate:"required"`
	Name          string                 `json:"name" validate:"required,min=1,max=200"`
	Description   string                 `json:"description" validate:"required"`
	SKU           string                 `json:"sku,omitempty"`
	Thumbnail     model.Image            `json:"thumbnail" validate:"required"`
	Images        []model.Image          `json:"images"`
	Videos        []model.Video          `json:"videos,omitempty"`
	Price         float64                `json:"price" validate:"required,min=0"`
	SalePrice     *float64               `json:"sale_price,omitempty"`
	StockQuantity int                    `json:"stock_quantity" validate:"required,min=0"`
	HasVariants   bool                   `json:"has_variants"`
	Attributes    []model.AttributeSpec  `json:"attributes,omitempty"`
	Variants      []model.ProductVariant `json:"variants,omitempty"`
}

type UpdateProductRequest struct {
	CategoryID    *string               `json:"category_id,omitempty"`
	Name          *string               `json:"name,omitempty" validate:"omitempty,min=1,max=200"`
	Description   *string               `json:"description,omitempty"`
	SKU           *string               `json:"sku,omitempty"`
	Thumbnail     *model.Image          `json:"thumbnail,omitempty"`
	Images        []model.Image         `json:"images,omitempty"`
	Videos        []model.Video         `json:"videos,omitempty"`
	Price         *float64              `json:"price,omitempty" validate:"omitempty,min=0"`
	SalePrice     *float64              `json:"sale_price,omitempty"`
	StockQuantity *int                  `json:"stock_quantity,omitempty" validate:"omitempty,min=0"`
	Attributes    []model.AttributeSpec `json:"attributes,omitempty"`
}

type AddVariantRequest struct {
	SKU           string            `json:"sku,omitempty"`
	Attributes    []model.Attribute `json:"attributes" validate:"required"`
	Price         float64           `json:"price" validate:"required,min=0"`
	SalePrice     *float64          `json:"sale_price,omitempty"`
	StockQuantity int               `json:"stock_quantity" validate:"required,min=0"`
	Image         *model.Image      `json:"image,omitempty"`
}

type UpdateVariantRequest struct {
	SKU           *string              `json:"sku,omitempty"`
	Attributes    []model.Attribute    `json:"attributes,omitempty"`
	Price         *float64             `json:"price,omitempty" validate:"omitempty,min=0"`
	SalePrice     *float64             `json:"sale_price,omitempty"`
	StockQuantity *int                 `json:"stock_quantity,omitempty" validate:"omitempty,min=0"`
	Image         *model.Image         `json:"image,omitempty"`
	Status        *model.ProductStatus `json:"status,omitempty"`
}

// Response DTOs
type ProductResponse struct {
	ID            string                 `json:"id"`
	CategoryID    string                 `json:"category_id"`
	SellerID      string                 `json:"seller_id"`
	Name          string                 `json:"name"`
	Slug          string                 `json:"slug"`
	Description   string                 `json:"description"`
	SKU           string                 `json:"sku,omitempty"`
	Thumbnail     model.Image            `json:"thumbnail"`
	Images        []model.Image          `json:"images"`
	Videos        []model.Video          `json:"videos,omitempty"`
	Price         float64                `json:"price"`
	SalePrice     *float64               `json:"sale_price,omitempty"`
	StockQuantity int                    `json:"stock_quantity"`
	SoldCount     int                    `json:"sold_count"`
	Rating        float64                `json:"rating"`
	RatingCount   int                    `json:"rating_count"`
	ViewCount     int                    `json:"view_count"`
	HasVariants   bool                   `json:"has_variants"`
	Attributes    []model.AttributeSpec  `json:"attributes,omitempty"`
	Variants      []model.ProductVariant `json:"variants,omitempty"`
	Status        model.ProductStatus    `json:"status"`
	CreatedAt     string                 `json:"created_at"`
	UpdatedAt     string                 `json:"updated_at"`
}

type ProductStatsResponse struct {
	Total       int64 `json:"total"`
	Active      int64 `json:"active"`
	OutOfStock  int64 `json:"out_of_stock"`
	NewThisWeek int64 `json:"new_this_week"`
}

// Converter functions
func FromProduct(product *model.Product) *ProductResponse {
	if product == nil {
		return nil
	}

	return &ProductResponse{
		ID:            product.ID.Hex(),
		CategoryID:    product.CategoryID.Hex(),
		SellerID:      product.SellerID.Hex(),
		Name:          product.Name,
		Slug:          product.Slug,
		Description:   product.Description,
		SKU:           product.SKU,
		Thumbnail:     product.Thumbnail,
		Images:        product.Images,
		Videos:        product.Videos,
		Price:         product.Price,
		SalePrice:     product.SalePrice,
		StockQuantity: product.StockQuantity,
		SoldCount:     product.SoldCount,
		Rating:        product.Rating,
		RatingCount:   product.RatingCount,
		ViewCount:     product.ViewCount,
		HasVariants:   product.HasVariants,
		Attributes:    product.Attributes,
		Variants:      product.Variants,
		Status:        product.Status,
		CreatedAt:     product.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:     product.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

func FromProducts(products []*model.Product) []*ProductResponse {
	result := make([]*ProductResponse, len(products))
	for i, p := range products {
		result[i] = FromProduct(p)
	}
	return result
}

// dto/product_dto.go (thêm vào)

type UpdateProductStatusRequest struct {
	Status model.ProductStatus `json:"status" validate:"required"`
}

type UpdateStockRequest struct {
	Quantity int `json:"quantity" validate:"required,min=0"`
}
