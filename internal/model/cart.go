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
	ProductID primitive.ObjectID  `bson:"product_id" json:"product_id"`
	VariantID *primitive.ObjectID `bson:"variant_id,omitempty" json:"variant_id,omitempty"` // nil nếu product không có variant
	SellerID  primitive.ObjectID  `bson:"seller_id" json:"seller_id"`
	Quantity  int                 `bson:"quantity" json:"quantity"`
	AddedAt   time.Time           `bson:"added_at" json:"added_at"`

	// Snapshot data (lưu tại thời điểm add to cart để hiển thị)
	// Sẽ được cập nhật khi load cart
	Snapshot *CartItemSnapshot `bson:"snapshot,omitempty" json:"snapshot,omitempty"`
}

// CartItemSnapshot - Dữ liệu snapshot để hiển thị (có thể outdated)
type CartItemSnapshot struct {
	ProductName string      `bson:"product_name" json:"product_name"`
	SellerName  string      `bson:"seller_name" json:"seller_name"`
	SKU         string      `bson:"sku,omitempty" json:"sku,omitempty"`
	Price       float64     `bson:"price" json:"price"`
	SalePrice   *float64    `bson:"sale_price,omitempty" json:"sale_price,omitempty"`
	Image       Image       `bson:"image" json:"image"`
	Attributes  []Attribute `bson:"attributes,omitempty" json:"attributes,omitempty"` // VD: [{Màu: Đỏ}, {Size: M}]
	Stock       int         `bson:"stock" json:"stock"`
	IsAvailable bool        `bson:"is_available" json:"is_available"`
}

// ============ HELPER METHODS ============

// GetEffectivePrice - Lấy giá thực tế
func (s *CartItemSnapshot) GetEffectivePrice() float64 {
	if s.SalePrice != nil && *s.SalePrice > 0 {
		return *s.SalePrice
	}
	return s.Price
}

// GetItemTotal - Tính tổng tiền của item
func (item *CartItem) GetItemTotal() float64 {
	if item.Snapshot == nil {
		return 0
	}
	return item.Snapshot.GetEffectivePrice() * float64(item.Quantity)
}

// GetCartTotal - Tính tổng tiền giỏ hàng
func (c *Cart) GetCartTotal() float64 {
	var total float64
	for _, item := range c.Items {
		total += item.GetItemTotal()
	}
	return total
}

// GetItemCount - Đếm số lượng sản phẩm
func (c *Cart) GetItemCount() int {
	count := 0
	for _, item := range c.Items {
		count += item.Quantity
	}
	return count
}

// FindItem - Tìm item trong cart
func (c *Cart) FindItem(productID, variantID primitive.ObjectID) *CartItem {
	for i := range c.Items {
		if c.Items[i].ProductID == productID {
			// Nếu không có variant
			if c.Items[i].VariantID == nil && variantID.IsZero() {
				return &c.Items[i]
			}
			// Nếu có variant
			if c.Items[i].VariantID != nil && *c.Items[i].VariantID == variantID {
				return &c.Items[i]
			}
		}
	}
	return nil
}
