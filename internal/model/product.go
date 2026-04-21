// model/product.go
package model

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Product struct {
	ID         primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	CategoryID primitive.ObjectID `bson:"category_id" json:"category_id"`
	SellerID   primitive.ObjectID `bson:"seller_id" json:"seller_id"`

	// Basic Info
	Name        string `bson:"name" json:"name"`
	Slug        string `bson:"slug" json:"slug"` // URL-friendly: "ao-thun-nam-cotton"
	Description string `bson:"description" json:"description"`
	SKU         string `bson:"sku,omitempty" json:"sku,omitempty"` // Mã sản phẩm

	// Media
	Thumbnail Image   `bson:"thumbnail" json:"thumbnail"`
	Images    []Image `bson:"images" json:"images"`
	Videos    []Video `bson:"videos,omitempty" json:"videos,omitempty"`

	// Pricing (cho sản phẩm KHÔNG có variants)
	Price     float64  `bson:"price" json:"price"`
	SalePrice *float64 `bson:"sale_price,omitempty" json:"sale_price,omitempty"` // Giá khuyến mãi

	// Stock (cho sản phẩm KHÔNG có variants)
	StockQuantity int `bson:"stock_quantity" json:"stock_quantity"`

	// Stats
	SoldCount   int     `bson:"sold_count" json:"sold_count"`
	Rating      float64 `bson:"rating" json:"rating"`
	RatingCount int     `bson:"rating_count" json:"rating_count"`
	ViewCount   int     `bson:"view_count" json:"view_count"`

	// Variants (biến thể: màu sắc, size, ...)
	HasVariants bool             `bson:"has_variants" json:"has_variants"`
	Attributes  []AttributeSpec  `bson:"attributes,omitempty" json:"attributes,omitempty"` // Định nghĩa attributes
	Variants    []ProductVariant `bson:"variants,omitempty" json:"variants,omitempty"`     // Các biến thể

	// Status
	Status ProductStatus `bson:"status" json:"status"`

	// Timestamps
	CreatedAt time.Time  `bson:"created_at" json:"created_at"`
	UpdatedAt time.Time  `bson:"updated_at" json:"updated_at"`
	DeletedAt *time.Time `bson:"deleted_at,omitempty" json:"deleted_at,omitempty"`
}

// ProductStatus - Trạng thái sản phẩm
type ProductStatus string

const (
	ProductStatusDraft      ProductStatus = "draft"        // Nháp, chưa publish
	ProductStatusPending    ProductStatus = "pending"      // Chờ duyệt
	ProductStatusActive     ProductStatus = "active"       // Đang bán
	ProductStatusInactive   ProductStatus = "inactive"     // Tạm ngưng bán
	ProductStatusOutOfStock ProductStatus = "out_of_stock" // Hết hàng
	ProductStatusRejected   ProductStatus = "rejected"     // Bị từ chối
)

// AttributeSpec - Định nghĩa attribute (VD: "Màu sắc" có các giá trị "Đỏ", "Xanh", "Vàng")
type AttributeSpec struct {
	Name   string   `bson:"name" json:"name"`     // "Màu sắc", "Size"
	Values []string `bson:"values" json:"values"` // ["Đỏ", "Xanh", "Vàng"]
}

// ProductVariant - Biến thể sản phẩm (VD: Áo đỏ size M)
type ProductVariant struct {
	ID            primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	SKU           string             `bson:"sku,omitempty" json:"sku,omitempty"`
	Attributes    []Attribute        `bson:"attributes" json:"attributes"` // [{"name": "Màu", "value": "Đỏ"}, {"name": "Size", "value": "M"}]
	Price         float64            `bson:"price" json:"price"`
	SalePrice     *float64           `bson:"sale_price,omitempty" json:"sale_price,omitempty"`
	StockQuantity int                `bson:"stock_quantity" json:"stock_quantity"`
	SoldCount     int                `bson:"sold_count" json:"sold_count"`
	Image         *Image             `bson:"image,omitempty" json:"image,omitempty"`
	Status        ProductStatus      `bson:"status" json:"status"`
}

// Attribute - Một cặp attribute name-value
type Attribute struct {
	Name  string `bson:"name" json:"name"`   // "Màu sắc"
	Value string `bson:"value" json:"value"` // "Đỏ"
}

// model/product.go

// GetEffectivePrice - Lấy giá thực tế (ưu tiên sale_price)
func (p *Product) GetEffectivePrice() float64 {
	if p.SalePrice != nil && *p.SalePrice > 0 {
		return *p.SalePrice
	}
	return p.Price
}

// GetTotalStock - Tổng stock (bao gồm variants)
func (p *Product) GetTotalStock() int {
	if !p.HasVariants {
		return p.StockQuantity
	}

	total := 0
	for _, v := range p.Variants {
		total += v.StockQuantity
	}
	return total
}

// GetTotalSold - Tổng đã bán (bao gồm variants)
func (p *Product) GetTotalSold() int {
	if !p.HasVariants {
		return p.SoldCount
	}

	total := 0
	for _, v := range p.Variants {
		total += v.SoldCount
	}
	return total
}

// IsAvailable - Kiểm tra còn hàng không
func (p *Product) IsAvailable() bool {
	if p.Status != ProductStatusActive {
		return false
	}
	return p.GetTotalStock() > 0
}

// GetPriceRange - Lấy khoảng giá (cho sản phẩm có variants)
func (p *Product) GetPriceRange() (min, max float64) {
	if !p.HasVariants || len(p.Variants) == 0 {
		return p.GetEffectivePrice(), p.GetEffectivePrice()
	}

	min = p.Variants[0].Price
	max = p.Variants[0].Price

	for _, v := range p.Variants {
		price := v.Price
		if v.SalePrice != nil && *v.SalePrice > 0 {
			price = *v.SalePrice
		}
		if price < min {
			min = price
		}
		if price > max {
			max = price
		}
	}

	return min, max
}

// GetVariantByAttributes - Tìm variant theo attributes
func (p *Product) GetVariantByAttributes(attrs []Attribute) *ProductVariant {
	for i, v := range p.Variants {
		if v.MatchAttributes(attrs) {
			return &p.Variants[i]
		}
	}
	return nil
}

// MatchAttributes - Kiểm tra variant có match với attributes không
func (v *ProductVariant) MatchAttributes(attrs []Attribute) bool {
	if len(v.Attributes) != len(attrs) {
		return false
	}

	for _, attr := range attrs {
		found := false
		for _, va := range v.Attributes {
			if va.Name == attr.Name && va.Value == attr.Value {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	return true
}

// GetEffectivePrice - Lấy giá thực tế của variant
func (v *ProductVariant) GetEffectivePrice() float64 {
	if v.SalePrice != nil && *v.SalePrice > 0 {
		return *v.SalePrice
	}
	return v.Price
}
