package model

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ==================== Order Status ====================

type OrderStatus string

const (
	OrderStatusPending   OrderStatus = "pending"
	OrderStatusConfirmed OrderStatus = "confirmed"
	OrderStatusPacked    OrderStatus = "packed"
	OrderStatusShipped   OrderStatus = "shipped"
	OrderStatusDelivered OrderStatus = "delivered"
	OrderStatusCanceled  OrderStatus = "canceled"
	OrderStatusReturned  OrderStatus = "returned"
	OrderStatusRefunded  OrderStatus = "refunded"
)

func (s OrderStatus) IsValid() bool {
	switch s {
	case OrderStatusPending, OrderStatusConfirmed, OrderStatusPacked,
		OrderStatusShipped, OrderStatusDelivered, OrderStatusCanceled,
		OrderStatusReturned, OrderStatusRefunded:
		return true
	}
	return false
}

func (s OrderStatus) DisplayName() string {
	names := map[OrderStatus]string{
		OrderStatusPending:   "Chờ xác nhận",
		OrderStatusConfirmed: "Đã xác nhận",
		OrderStatusPacked:    "Đã đóng gói",
		OrderStatusShipped:   "Đang giao hàng",
		OrderStatusDelivered: "Đã giao hàng",
		OrderStatusCanceled:  "Đã hủy",
		OrderStatusReturned:  "Đã trả hàng",
		OrderStatusRefunded:  "Đã hoàn tiền",
	}
	return names[s]
}

func (s OrderStatus) CanCancel() bool {
	return s == OrderStatusPending || s == OrderStatusConfirmed
}

func (s OrderStatus) CanReturn() bool {
	return s == OrderStatusDelivered
}

// ==================== Payment Status ====================

type PaymentStatus string

const (
	PaymentStatusPending  PaymentStatus = "pending"
	PaymentStatusPaid     PaymentStatus = "paid"
	PaymentStatusFailed   PaymentStatus = "failed"
	PaymentStatusRefunded PaymentStatus = "refunded"
)

func (s PaymentStatus) IsValid() bool {
	switch s {
	case PaymentStatusPending, PaymentStatusPaid, PaymentStatusFailed, PaymentStatusRefunded:
		return true
	}
	return false
}

func (s PaymentStatus) DisplayName() string {
	names := map[PaymentStatus]string{
		PaymentStatusPending:  "Chờ thanh toán",
		PaymentStatusPaid:     "Đã thanh toán",
		PaymentStatusFailed:   "Thanh toán thất bại",
		PaymentStatusRefunded: "Đã hoàn tiền",
	}
	return names[s]
}

// ==================== Payment Method ====================

type PaymentMethod string

const (
	PaymentMethodCOD          PaymentMethod = "cod"
	PaymentMethodBankTransfer PaymentMethod = "bank_transfer"
	PaymentMethodEWallet      PaymentMethod = "e_wallet"
	PaymentMethodCreditCard   PaymentMethod = "credit_card"
)

func (m PaymentMethod) IsValid() bool {
	switch m {
	case PaymentMethodCOD, PaymentMethodBankTransfer, PaymentMethodEWallet, PaymentMethodCreditCard:
		return true
	}
	return false
}

func (m PaymentMethod) DisplayName() string {
	names := map[PaymentMethod]string{
		PaymentMethodCOD:          "Thanh toán khi nhận hàng",
		PaymentMethodBankTransfer: "Chuyển khoản ngân hàng",
		PaymentMethodEWallet:      "Ví điện tử",
		PaymentMethodCreditCard:   "Thẻ tín dụng",
	}
	return names[m]
}

func (m PaymentMethod) RequiresPrePayment() bool {
	return m != PaymentMethodCOD
}

// ==================== Order Item ====================

type OrderItem struct {
	ProductID   primitive.ObjectID `bson:"product_id" json:"product_id"`
	VariantID   string             `bson:"variant_id,omitempty" json:"variant_id,omitempty"`
	ProductName string             `bson:"product_name" json:"product_name"`
	SKU         string             `bson:"sku" json:"sku"`
	Image       Image              `bson:"image" json:"image"` // ✅ Changed from string to Image
	Price       float64            `bson:"price" json:"price"`
	Quantity    int                `bson:"quantity" json:"quantity"`
	Subtotal    float64            `bson:"subtotal" json:"subtotal"`
}

func (i *OrderItem) CalculateSubtotal() {
	i.Subtotal = i.Price * float64(i.Quantity)
}

// ==================== Shipping Address ====================

type ShippingAddress struct {
	RecipientName string             `bson:"recipient_name" json:"recipient_name"`
	PhoneNumber   string             `bson:"phone_number" json:"phone_number"`
	ProvinceID    primitive.ObjectID `bson:"province_id" json:"province_id"`
	ProvinceName  string             `bson:"province_name" json:"province_name"`
	WardID        string             `bson:"ward_id" json:"ward_id"`
	WardName      string             `bson:"ward_name" json:"ward_name"`
	Detail        string             `bson:"detail" json:"detail"`
	FullAddress   string             `bson:"full_address" json:"full_address"`
}

func (a *ShippingAddress) BuildFullAddress() {
	a.FullAddress = a.Detail + ", " + a.WardName + ", " + a.ProvinceName
}

// ==================== Status History ====================

type StatusHistory struct {
	Status    OrderStatus `bson:"status" json:"status"`
	Note      string      `bson:"note,omitempty" json:"note,omitempty"`
	UpdatedBy string      `bson:"updated_by" json:"updated_by"`
	UpdatedAt time.Time   `bson:"updated_at" json:"updated_at"`
}

// ==================== Order ====================

type Order struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	OrderNumber string             `bson:"order_number" json:"order_number"`
	CustomerID  primitive.ObjectID `bson:"customer_id" json:"customer_id"`
	SellerID    primitive.ObjectID `bson:"seller_id" json:"seller_id"`

	// Items
	Items []OrderItem `bson:"items" json:"items"`

	// Pricing
	Subtotal    float64 `bson:"subtotal" json:"subtotal"`
	ShippingFee float64 `bson:"shipping_fee" json:"shipping_fee"`
	Discount    float64 `bson:"discount" json:"discount"`
	Tax         float64 `bson:"tax" json:"tax"`
	Total       float64 `bson:"total" json:"total"`

	// Shipping
	ShippingAddress ShippingAddress `bson:"shipping_address" json:"shipping_address"`

	// Payment
	PaymentMethod PaymentMethod `bson:"payment_method" json:"payment_method"`
	PaymentStatus PaymentStatus `bson:"payment_status" json:"payment_status"`
	PaidAt        *time.Time    `bson:"paid_at,omitempty" json:"paid_at,omitempty"`

	// Status
	Status        OrderStatus     `bson:"status" json:"status"`
	StatusHistory []StatusHistory `bson:"status_history" json:"status_history"`

	// Tracking
	TrackingNumber    string     `bson:"tracking_number,omitempty" json:"tracking_number,omitempty"`
	ShippingCarrier   string     `bson:"shipping_carrier,omitempty" json:"shipping_carrier,omitempty"`
	EstimatedDelivery *time.Time `bson:"estimated_delivery,omitempty" json:"estimated_delivery,omitempty"`

	// Notes
	CustomerNote string `bson:"customer_note,omitempty" json:"customer_note,omitempty"`
	SellerNote   string `bson:"seller_note,omitempty" json:"seller_note,omitempty"`
	CancelReason string `bson:"cancel_reason,omitempty" json:"cancel_reason,omitempty"`

	// Timestamps
	CreatedAt   time.Time  `bson:"created_at" json:"created_at"`
	UpdatedAt   time.Time  `bson:"updated_at" json:"updated_at"`
	ConfirmedAt *time.Time `bson:"confirmed_at,omitempty" json:"confirmed_at,omitempty"`
	ShippedAt   *time.Time `bson:"shipped_at,omitempty" json:"shipped_at,omitempty"`
	DeliveredAt *time.Time `bson:"delivered_at,omitempty" json:"delivered_at,omitempty"`
	CanceledAt  *time.Time `bson:"canceled_at,omitempty" json:"canceled_at,omitempty"`
}

// ==================== Order Methods ====================

func (o *Order) CalculateTotals() {
	o.Subtotal = 0
	for i := range o.Items {
		o.Items[i].CalculateSubtotal()
		o.Subtotal += o.Items[i].Subtotal
	}
	o.Total = o.Subtotal + o.ShippingFee + o.Tax - o.Discount
}

func (o *Order) GetItemCount() int {
	count := 0
	for _, item := range o.Items {
		count += item.Quantity
	}
	return count
}

func (o *Order) AddStatusHistory(status OrderStatus, note, updatedBy string) {
	history := StatusHistory{
		Status:    status,
		Note:      note,
		UpdatedBy: updatedBy,
		UpdatedAt: time.Now(),
	}
	o.StatusHistory = append(o.StatusHistory, history)
	o.Status = status
	o.UpdatedAt = time.Now()

	now := time.Now()
	switch status {
	case OrderStatusConfirmed:
		o.ConfirmedAt = &now
	case OrderStatusShipped:
		o.ShippedAt = &now
	case OrderStatusDelivered:
		o.DeliveredAt = &now
	case OrderStatusCanceled:
		o.CanceledAt = &now
	}
}

func (o *Order) CanTransitionTo(newStatus OrderStatus) bool {
	transitions := map[OrderStatus][]OrderStatus{
		OrderStatusPending:   {OrderStatusConfirmed, OrderStatusCanceled},
		OrderStatusConfirmed: {OrderStatusPacked, OrderStatusCanceled},
		OrderStatusPacked:    {OrderStatusShipped, OrderStatusCanceled},
		OrderStatusShipped:   {OrderStatusDelivered},
		OrderStatusDelivered: {OrderStatusReturned},
		OrderStatusReturned:  {OrderStatusRefunded},
	}

	allowed, exists := transitions[o.Status]
	if !exists {
		return false
	}

	for _, s := range allowed {
		if s == newStatus {
			return true
		}
	}
	return false
}

func (o *Order) CanCancel() bool {
	return o.Status.CanCancel()
}

func (o *Order) CanReturn() bool {
	return o.Status.CanReturn()
}

func (o *Order) IsCompleted() bool {
	return o.Status == OrderStatusDelivered || o.Status == OrderStatusRefunded
}

func (o *Order) IsCanceled() bool {
	return o.Status == OrderStatusCanceled
}

func (o *Order) IsPaid() bool {
	return o.PaymentStatus == PaymentStatusPaid
}

func (o *Order) MarkAsPaid() {
	o.PaymentStatus = PaymentStatusPaid
	now := time.Now()
	o.PaidAt = &now
	o.UpdatedAt = now
}

func (o *Order) SetTracking(trackingNumber, carrier string, estimatedDelivery *time.Time) {
	o.TrackingNumber = trackingNumber
	o.ShippingCarrier = carrier
	o.EstimatedDelivery = estimatedDelivery
	o.UpdatedAt = time.Now()
}
