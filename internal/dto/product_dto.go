// dto/product_dto.go
package dto

import (
	"github.com/luong-vh/Digimart_Backend/internal/model"
)

// Request DTOs
type CreateProductRequest struct {
	CategoryID      string                 `json:"category_id" validate:"required"`
	Name            string                 `json:"name" validate:"required,min=1,max=200"`
	Description     string                 `json:"description" validate:"required"`
	Brand           string                 `json:"brand,omitempty"`
	Specs           []model.ProductSpec    `json:"specs,omitempty"`
	Attributes      []model.ProductSpec    `json:"attributes,omitempty"`
	Specifications  []model.ProductSpec    `json:"specifications,omitempty"`
	Thumbnail       model.Image            `json:"thumbnail" validate:"required"`
	Images          []model.Image          `json:"images"`
	Videos          []model.Video          `json:"videos,omitempty"`
	BasePrice       float64                `json:"base_price" validate:"required,min=0"`
	DiscountPercent float64                `json:"discount_percent" validate:"min=0,max=100"`
	IsHasVariant    bool                   `json:"is_has_variant"`
	StockQuantity   *int                   `json:"stock_quantity,omitempty"`
	Variants        []model.ProductVariant `json:"variants,omitempty"`
}

type UpdateProductRequest struct {
	CategoryID      *string              `json:"category_id,omitempty"`
	Name            *string              `json:"name,omitempty" validate:"omitempty,min=1,max=200"`
	Description     *string              `json:"description,omitempty"`
	Brand           *string              `json:"brand,omitempty"`
	Specs           []model.ProductSpec  `json:"specs,omitempty"`
	Attributes      []model.ProductSpec  `json:"attributes,omitempty"`
	Specifications  []model.ProductSpec  `json:"specifications,omitempty"`
	Thumbnail       *model.Image         `json:"thumbnail,omitempty"`
	Images          []model.Image        `json:"images,omitempty"`
	Videos          []model.Video        `json:"videos,omitempty"`
	BasePrice       *float64             `json:"base_price,omitempty" validate:"omitempty,min=0"`
	DiscountPercent *float64             `json:"discount_percent,omitempty" validate:"omitempty,min=0,max=100"`
	StockQuantity   *int                 `json:"stock_quantity,omitempty"`
	Status          *model.ProductStatus `json:"status,omitempty" validate:"omitempty,oneof=draft active "`
}

// Image operations
type AddImagesRequest struct {
	Images []model.Image `json:"images" validate:"required,dive"`
}

type DeleteImageRequest struct {
	ImageID string `json:"image_id" validate:"required"`
}

// Video operations
type AddVideosRequest struct {
	Videos []model.Video `json:"videos" validate:"required,dive"`
}

type DeleteVideoRequest struct {
	VideoID string `json:"video_id" validate:"required"`
}

// Variant CRUD
type CreateVariantRequest struct {
	Title           string  `json:"title" validate:"required"`
	Description     string  `json:"description,omitempty"`
	PriceAdjustment float64 `json:"price_adjustment"`
	StockQuantity   int     `json:"stock_quantity" validate:"required,min=0"`
}

type UpdateVariantRequest struct {
	Title           *string  `json:"title,omitempty"`
	Description     *string  `json:"description,omitempty"`
	PriceAdjustment *float64 `json:"price_adjustment,omitempty"`
	StockQuantity   *int     `json:"stock_quantity,omitempty" validate:"omitempty,min=0"`
}

// Response DTOs
type ProductResponse struct {
	ID              string                 `json:"id"`
	CategoryID      string                 `json:"category_id"`
	Name            string                 `json:"name"`
	Description     string                 `json:"description"`
	Brand           string                 `json:"brand,omitempty"`
	Specs           []model.ProductSpec    `json:"specs,omitempty"`
	Attributes      []model.ProductSpec    `json:"attributes,omitempty"`
	Specifications  []model.ProductSpec    `json:"specifications,omitempty"`
	Thumbnail       model.Image            `json:"thumbnail"`
	Images          []model.Image          `json:"images"`
	Videos          []model.Video          `json:"videos,omitempty"`
	BasePrice       float64                `json:"base_price"`
	DiscountPercent float64                `json:"discount_percent"`
	IsHasVariant    bool                   `json:"is_has_variant"`
	StockQuantity   *int                   `json:"stock_quantity,omitempty"`
	SoldCount       int                    `json:"sold_count"`
	Rating          float64                `json:"rating"`
	RatingCount     int                    `json:"rating_count"`
	Variants        []model.ProductVariant `json:"variants,omitempty"`
	Status          model.ProductStatus    `json:"status"`
	CreatedAt       string                 `json:"created_at"`
	UpdatedAt       string                 `json:"updated_at"`
}

type ProductStatsResponse struct {
	Total       int64 `json:"total"`
	Active      int64 `json:"active"`
	OutOfStock  int64 `json:"out_of_stock"`
	NewThisWeek int64 `json:"new_this_week"`
}

func (r CreateProductRequest) NormalizedSpecs() []model.ProductSpec {
	if len(r.Specs) > 0 {
		return r.Specs
	}
	if len(r.Attributes) > 0 {
		return r.Attributes
	}
	return r.Specifications
}

func (r UpdateProductRequest) NormalizedSpecs() ([]model.ProductSpec, bool) {
	if r.Specs != nil {
		return r.Specs, true
	}
	if r.Attributes != nil {
		return r.Attributes, true
	}
	if r.Specifications != nil {
		return r.Specifications, true
	}
	return nil, false
}

type VariantResponse struct {
	ID              string  `json:"id"`
	Title           string  `json:"title"`
	Description     string  `json:"description,omitempty"`
	PriceAdjustment float64 `json:"price_adjustment"`
	FinalPrice      float64 `json:"final_price"`
	StockQuantity   int     `json:"stock_quantity"`
}

// Converter functions
func FromProduct(product *model.Product) *ProductResponse {
	if product == nil {
		return nil
	}

	return &ProductResponse{
		ID:              product.ID.Hex(),
		CategoryID:      product.CategoryID.Hex(),
		Name:            product.Name,
		Description:     product.Description,
		Brand:           product.Brand,
		Specs:           product.Specs,
		Attributes:      product.Specs,
		Specifications:  product.Specs,
		Thumbnail:       product.Thumbnail,
		Images:          product.Images,
		Videos:          product.Videos,
		BasePrice:       product.BasePrice,
		DiscountPercent: product.DiscountPercent,
		IsHasVariant:    product.IsHasVariant,
		StockQuantity:   product.StockQuantity,
		SoldCount:       product.SoldCount,
		Rating:          product.Rating,
		RatingCount:     product.RatingCount,
		Variants:        product.Variants,
		Status:          product.Status,
		CreatedAt:       product.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:       product.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

func FromProducts(products []*model.Product) []*ProductResponse {
	result := make([]*ProductResponse, len(products))
	for i, p := range products {
		result[i] = FromProduct(p)
	}
	return result
}

func FromVariant(variant *model.ProductVariant) *VariantResponse {
	if variant == nil {
		return nil
	}
	return &VariantResponse{
		ID:              variant.ID.Hex(),
		Title:           variant.Title,
		Description:     variant.Description,
		PriceAdjustment: variant.PriceAdjustment,
		FinalPrice:      variant.FinalPrice,
		StockQuantity:   variant.StockQuantity,
	}
}
