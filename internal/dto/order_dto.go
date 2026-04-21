package dto

import (
	"time"

	"github.com/DucLove1/SE357-ShoppingManagement-BE/internal/model"
	"github.com/DucLove1/SE357-ShoppingManagement-BE/internal/repo"
)

// ==================== Request DTOs ====================

type PlaceOrderRequest struct {
	Items           []OrderItemRequest     `json:"items" binding:"required,min=1"`
	ShippingAddress ShippingAddressRequest `json:"shipping_address" binding:"required"`
	PaymentMethod   model.PaymentMethod    `json:"payment_method" binding:"required"`
	CustomerNote    string                 `json:"customer_note"`
}

type OrderItemRequest struct {
	ProductID string `json:"product_id" binding:"required"`
	VariantID string `json:"variant_id"`
	Quantity  int    `json:"quantity" binding:"required,min=1"`
}

type ShippingAddressRequest struct {
	RecipientName string `json:"recipient_name" binding:"required"`
	PhoneNumber   string `json:"phone_number" binding:"required"`
	ProvinceID    string `json:"province_id" binding:"required"`
	WardID        string `json:"ward_id" binding:"required"`
	Detail        string `json:"detail" binding:"required"`
}

type CancelOrderRequest struct {
	Reason string `json:"reason" binding:"required"`
}

type ReturnOrderRequest struct {
	Reason string `json:"reason" binding:"required"`
}

type RejectOrderRequest struct {
	Reason string `json:"reason" binding:"required"`
}

type UpdateTrackingRequest struct {
	TrackingNumber    string     `json:"tracking_number" binding:"required"`
	ShippingCarrier   string     `json:"shipping_carrier" binding:"required"`
	EstimatedDelivery *time.Time `json:"estimated_delivery"`
}

type UpdateOrderStatusRequest struct {
	Status model.OrderStatus `json:"status" binding:"required"`
	Note   string            `json:"note"`
}

// ==================== Query DTOs ====================

type OrderFilterQuery struct {
	Status        string `form:"status"`
	PaymentStatus string `form:"payment_status"`
	PaymentMethod string `form:"payment_method"`
	Search        string `form:"search"`
	StartDate     string `form:"start_date"`
	EndDate       string `form:"end_date"`
	CustomerID    string `form:"customer_id"`
	SellerID      string `form:"seller_id"`
	Page          int    `form:"page"`
	PageSize      int    `form:"page_size"`
	SortBy        string `form:"sort_by"`
	SortOrder     string `form:"sort_order"`
}

// ==================== Response DTOs ====================

type OrderResponse struct {
	ID                string                  `json:"id"`
	OrderNumber       string                  `json:"order_number"`
	CustomerID        string                  `json:"customer_id"`
	SellerID          string                  `json:"seller_id"`
	Items             []OrderItemResponse     `json:"items"`
	Subtotal          float64                 `json:"subtotal"`
	ShippingFee       float64                 `json:"shipping_fee"`
	Discount          float64                 `json:"discount"`
	Tax               float64                 `json:"tax"`
	Total             float64                 `json:"total"`
	ShippingAddress   ShippingAddressResponse `json:"shipping_address"`
	PaymentMethod     string                  `json:"payment_method"`
	PaymentStatus     string                  `json:"payment_status"`
	PaidAt            *time.Time              `json:"paid_at,omitempty"`
	Status            string                  `json:"status"`
	StatusHistory     []StatusHistoryResponse `json:"status_history"`
	TrackingNumber    string                  `json:"tracking_number,omitempty"`
	ShippingCarrier   string                  `json:"shipping_carrier,omitempty"`
	EstimatedDelivery *time.Time              `json:"estimated_delivery,omitempty"`
	CustomerNote      string                  `json:"customer_note,omitempty"`
	CancelReason      string                  `json:"cancel_reason,omitempty"`
	CreatedAt         time.Time               `json:"created_at"`
	UpdatedAt         time.Time               `json:"updated_at"`
	ConfirmedAt       *time.Time              `json:"confirmed_at,omitempty"`
	ShippedAt         *time.Time              `json:"shipped_at,omitempty"`
	DeliveredAt       *time.Time              `json:"delivered_at,omitempty"`
}

type OrderItemResponse struct {
	ProductID   string        `json:"product_id"`
	VariantID   string        `json:"variant_id,omitempty"`
	ProductName string        `json:"product_name"`
	VariantName string        `json:"variant_name,omitempty"`
	SKU         string        `json:"sku"`
	Image       ImageResponse `json:"image"`
	Price       float64       `json:"price"`
	Quantity    int           `json:"quantity"`
	Subtotal    float64       `json:"subtotal"`
}

type ShippingAddressResponse struct {
	RecipientName string `json:"recipient_name"`
	PhoneNumber   string `json:"phone_number"`
	ProvinceID    string `json:"province_id"`
	ProvinceName  string `json:"province_name"`
	WardID        string `json:"ward_id"`
	WardName      string `json:"ward_name"`
	Detail        string `json:"detail"`
	FullAddress   string `json:"full_address"`
}

type StatusHistoryResponse struct {
	Status    string    `json:"status"`
	Note      string    `json:"note,omitempty"`
	UpdatedBy string    `json:"updated_by"`
	UpdatedAt time.Time `json:"updated_at"`
}

type OrderListResponse struct {
	ID            string    `json:"id"`
	OrderNumber   string    `json:"order_number"`
	CustomerID    string    `json:"customer_id"`
	SellerID      string    `json:"seller_id"`
	ItemCount     int       `json:"item_count"`
	Total         float64   `json:"total"`
	Status        string    `json:"status"`
	PaymentStatus string    `json:"payment_status"`
	PaymentMethod string    `json:"payment_method"`
	CreatedAt     time.Time `json:"created_at"`
}

type PaginatedOrdersResponse struct {
	Orders     []OrderListResponse `json:"orders"`
	Pagination Pagination          `json:"pagination"`
}

type OrderStatsResponse struct {
	TotalOrders     int64   `json:"total_orders"`
	PendingOrders   int64   `json:"pending_orders"`
	ConfirmedOrders int64   `json:"confirmed_orders"`
	PackedOrders    int64   `json:"packed_orders"`
	ShippedOrders   int64   `json:"shipped_orders"`
	DeliveredOrders int64   `json:"delivered_orders"`
	CanceledOrders  int64   `json:"canceled_orders"`
	ReturnedOrders  int64   `json:"returned_orders"`
	RefundedOrders  int64   `json:"refunded_orders"`
	TotalRevenue    float64 `json:"total_revenue"`
}

// ==================== Converters ====================

func FromOrder(o *model.Order) *OrderResponse {
	if o == nil {
		return nil
	}

	items := make([]OrderItemResponse, len(o.Items))
	for i, item := range o.Items {
		items[i] = OrderItemResponse{
			ProductID:   item.ProductID.Hex(),
			VariantID:   item.VariantID,
			ProductName: item.ProductName,
			SKU:         item.SKU,
			Image:       FromImage(item.Image),
			Price:       item.Price,
			Quantity:    item.Quantity,
			Subtotal:    item.Subtotal,
		}
	}

	history := make([]StatusHistoryResponse, len(o.StatusHistory))
	for i, h := range o.StatusHistory {
		history[i] = StatusHistoryResponse{
			Status:    string(h.Status),
			Note:      h.Note,
			UpdatedBy: h.UpdatedBy,
			UpdatedAt: h.UpdatedAt,
		}
	}

	return &OrderResponse{
		ID:          o.ID.Hex(),
		OrderNumber: o.OrderNumber,
		CustomerID:  o.CustomerID.Hex(),
		SellerID:    o.SellerID.Hex(),
		Items:       items,
		Subtotal:    o.Subtotal,
		ShippingFee: o.ShippingFee,
		Discount:    o.Discount,
		Tax:         o.Tax,
		Total:       o.Total,
		ShippingAddress: ShippingAddressResponse{
			RecipientName: o.ShippingAddress.RecipientName,
			PhoneNumber:   o.ShippingAddress.PhoneNumber,
			ProvinceID:    o.ShippingAddress.ProvinceID.Hex(),
			ProvinceName:  o.ShippingAddress.ProvinceName,
			WardID:        o.ShippingAddress.WardID,
			WardName:      o.ShippingAddress.WardName,
			Detail:        o.ShippingAddress.Detail,
			FullAddress:   o.ShippingAddress.FullAddress,
		},
		PaymentMethod:     string(o.PaymentMethod),
		PaymentStatus:     string(o.PaymentStatus),
		PaidAt:            o.PaidAt,
		Status:            string(o.Status),
		StatusHistory:     history,
		TrackingNumber:    o.TrackingNumber,
		ShippingCarrier:   o.ShippingCarrier,
		EstimatedDelivery: o.EstimatedDelivery,
		CustomerNote:      o.CustomerNote,
		CancelReason:      o.CancelReason,
		CreatedAt:         o.CreatedAt,
		UpdatedAt:         o.UpdatedAt,
		ConfirmedAt:       o.ConfirmedAt,
		ShippedAt:         o.ShippedAt,
		DeliveredAt:       o.DeliveredAt,
	}
}

func FromOrderList(orders []*model.Order) []OrderListResponse {
	result := make([]OrderListResponse, len(orders))
	for i, o := range orders {
		result[i] = OrderListResponse{
			ID:            o.ID.Hex(),
			OrderNumber:   o.OrderNumber,
			CustomerID:    o.CustomerID.Hex(),
			SellerID:      o.SellerID.Hex(),
			ItemCount:     o.GetItemCount(),
			Total:         o.Total,
			Status:        string(o.Status),
			PaymentStatus: string(o.PaymentStatus),
			PaymentMethod: string(o.PaymentMethod),
			CreatedAt:     o.CreatedAt,
		}
	}
	return result
}

func FromOrderStats(stats *repo.OrderStats) *OrderStatsResponse {
	if stats == nil {
		return &OrderStatsResponse{}
	}
	return &OrderStatsResponse{
		TotalOrders:     stats.TotalOrders,
		PendingOrders:   stats.PendingOrders,
		ConfirmedOrders: stats.ConfirmedOrders,
		PackedOrders:    stats.PackedOrders,
		ShippedOrders:   stats.ShippedOrders,
		DeliveredOrders: stats.DeliveredOrders,
		CanceledOrders:  stats.CanceledOrders,
		ReturnedOrders:  stats.ReturnedOrders,
		RefundedOrders:  stats.RefundedOrders,
		TotalRevenue:    stats.TotalRevenue,
	}
}

func FromImage(img model.Image) ImageResponse {
	return ImageResponse{
		URL:        img.URL,
		PublicID:   img.PublicID,
		UploadedAt: img.UploadedAt,
	}
}

type ImageResponse struct {
	URL        string    `json:"url"`
	PublicID   string    `json:"public_id"`
	UploadedAt time.Time `json:"uploaded_at"`
}
