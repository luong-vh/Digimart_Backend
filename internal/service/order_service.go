package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/DucLove1/SE357-ShoppingManagement-BE/internal/apperror"
	"github.com/DucLove1/SE357-ShoppingManagement-BE/internal/dto"
	"github.com/DucLove1/SE357-ShoppingManagement-BE/internal/model"
	"github.com/DucLove1/SE357-ShoppingManagement-BE/internal/repo"
	"github.com/DucLove1/SE357-ShoppingManagement-BE/internal/util"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// Constants
const (
	FreeShippingThreshold = 500000.0
	DefaultShippingFee    = 30000.0
	MaxItemsPerOrder      = 50
	MaxQuantityPerItem    = 999
	DefaultPageSize       = 10
	MaxPageSize           = 100
)

type OrderService interface {
	// Customer
	PlaceOrder(customerID string, req *dto.PlaceOrderRequest) (*dto.OrderResponse, error)
	GetMyOrders(customerID string, query *dto.OrderFilterQuery) (*dto.PaginatedOrdersResponse, error)
	GetOrderByID(orderID, userID, role string) (*dto.OrderResponse, error)
	GetOrderByNumber(orderNumber, userID, role string) (*dto.OrderResponse, error)
	CancelOrder(orderID, customerID string, req *dto.CancelOrderRequest) (*dto.OrderResponse, error)
	RequestReturn(orderID, customerID string, req *dto.ReturnOrderRequest) (*dto.OrderResponse, error)

	// Seller
	GetSellerOrders(sellerID string, query *dto.OrderFilterQuery) (*dto.PaginatedOrdersResponse, error)
	ConfirmOrder(orderID, sellerID string) (*dto.OrderResponse, error)
	PackOrder(orderID, sellerID string) (*dto.OrderResponse, error)
	ShipOrder(orderID, sellerID string, req *dto.UpdateTrackingRequest) (*dto.OrderResponse, error)
	UpdateTracking(orderID, sellerID string, req *dto.UpdateTrackingRequest) (*dto.OrderResponse, error)
	RejectOrder(orderID, sellerID string, req *dto.RejectOrderRequest) (*dto.OrderResponse, error)
	ProcessRefund(orderID, sellerID string) (*dto.OrderResponse, error)
	GetSellerOrderStats(sellerID string) (*dto.OrderStatsResponse, error)

	// Admin
	GetAllOrders(query *dto.OrderFilterQuery) (*dto.PaginatedOrdersResponse, error)
	AdminUpdateStatus(orderID string, req *dto.UpdateOrderStatusRequest) (*dto.OrderResponse, error)
	AdminMarkAsDelivered(orderID string) (*dto.OrderResponse, error)
	GetOrderStats() (*dto.OrderStatsResponse, error)

	// Payment
	UpdatePaymentStatus(orderID string, status model.PaymentStatus) error
	MarkAsPaid(orderID string) error
}

type orderService struct {
	orderRepo    repo.OrderRepo
	productRepo  repo.ProductRepo
	provinceRepo repo.ProvinceRepo
	userRepo     repo.UserRepo
}

func NewOrderService(
	orderRepo repo.OrderRepo,
	productRepo repo.ProductRepo,
	provinceRepo repo.ProvinceRepo,
	userRepo repo.UserRepo,
) OrderService {
	return &orderService{
		orderRepo:    orderRepo,
		productRepo:  productRepo,
		provinceRepo: provinceRepo,
		userRepo:     userRepo,
	}
}

// ==================== Customer Methods ====================

func (s *orderService) PlaceOrder(customerID string, req *dto.PlaceOrderRequest) (*dto.OrderResponse, error) {
	ctx, cancel := util.NewDefaultDBContext()
	defer cancel()

	// Validate request
	if err := s.validatePlaceOrderRequest(req); err != nil {
		return nil, err
	}

	// Validate customer ID format
	customerObjID, err := primitive.ObjectIDFromHex(customerID)
	if err != nil {
		return nil, apperror.ErrInvalidID
	}

	// Validate customer exists
	if _, err := s.userRepo.GetByID(ctx, customerID); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, apperror.ErrUserNotFound
		}
		return nil, err
	}

	// Validate and build order items
	items, sellerID, subtotal, err := s.validateAndBuildOrderItems(ctx, req.Items)
	if err != nil {
		return nil, err
	}

	// Validate shipping address
	shippingAddress, err := s.validateAndBuildShippingAddress(ctx, &req.ShippingAddress)
	if err != nil {
		return nil, err
	}

	// Validate payment method
	if !req.PaymentMethod.IsValid() {
		return nil, apperror.ErrInvalidPaymentMethod
	}

	// Calculate fees
	shippingFee := s.calculateShippingFee(subtotal)
	discount := 0.0
	tax := 0.0
	total := subtotal + shippingFee + tax - discount

	// Generate order number
	orderNumber, err := s.orderRepo.GenerateOrderNumber(ctx)
	if err != nil {
		return nil, err
	}

	// Create order model
	order := &model.Order{
		OrderNumber:     orderNumber,
		CustomerID:      customerObjID,
		SellerID:        sellerID,
		Items:           items,
		Subtotal:        subtotal,
		ShippingFee:     shippingFee,
		Discount:        discount,
		Tax:             tax,
		Total:           total,
		ShippingAddress: *shippingAddress,
		PaymentMethod:   req.PaymentMethod,
		PaymentStatus:   model.PaymentStatusPending,
		Status:          model.OrderStatusPending,
		StatusHistory: []model.StatusHistory{
			{
				Status:    model.OrderStatusPending,
				Note:      "Đơn hàng đã được tạo",
				UpdatedBy: "customer",
				UpdatedAt: time.Now(),
			},
		},
		CustomerNote: req.CustomerNote,
	}

	// Decrease stock before creating order
	if err := s.decreaseStock(ctx, req.Items); err != nil {
		return nil, err
	}

	// Save order
	createdOrder, err := s.orderRepo.Create(ctx, order)
	if err != nil {
		// Rollback stock on failure
		_ = s.restoreStock(ctx, items)
		return nil, err
	}

	return dto.FromOrder(createdOrder), nil
}

func (s *orderService) GetMyOrders(customerID string, query *dto.OrderFilterQuery) (*dto.PaginatedOrdersResponse, error) {
	ctx, cancel := util.NewDefaultDBContext()
	defer cancel()

	customerObjID, err := primitive.ObjectIDFromHex(customerID)
	if err != nil {
		return nil, apperror.ErrInvalidID
	}

	filter := s.buildOrderFilter(query)
	filter["customer_id"] = customerObjID

	return s.findOrdersWithPagination(ctx, filter, query)
}

func (s *orderService) GetOrderByID(orderID, userID, role string) (*dto.OrderResponse, error) {
	ctx, cancel := util.NewDefaultDBContext()
	defer cancel()

	order, err := s.getOrderByID(ctx, orderID)
	if err != nil {
		return nil, err
	}

	if err := s.checkOrderAccess(order, userID, role); err != nil {
		return nil, err
	}

	return dto.FromOrder(order), nil
}

func (s *orderService) GetOrderByNumber(orderNumber, userID, role string) (*dto.OrderResponse, error) {
	ctx, cancel := util.NewDefaultDBContext()
	defer cancel()

	order, err := s.orderRepo.GetByOrderNumber(ctx, orderNumber)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, apperror.ErrOrderNotFound
		}
		return nil, err
	}

	if err := s.checkOrderAccess(order, userID, role); err != nil {
		return nil, err
	}

	return dto.FromOrder(order), nil
}

func (s *orderService) CancelOrder(orderID, customerID string, req *dto.CancelOrderRequest) (*dto.OrderResponse, error) {
	ctx, cancel := util.NewDefaultDBContext()
	defer cancel()

	order, err := s.getOrderWithCustomerOwnership(ctx, orderID, customerID)
	if err != nil {
		return nil, err
	}

	if !order.CanCancel() {
		return nil, apperror.ErrOrderCannotBeCanceled
	}

	history := model.StatusHistory{
		Status:    model.OrderStatusCanceled,
		Note:      req.Reason,
		UpdatedBy: "customer",
		UpdatedAt: time.Now(),
	}

	if err := s.orderRepo.UpdateStatus(ctx, orderID, model.OrderStatusCanceled, history); err != nil {
		return nil, err
	}

	if err := s.orderRepo.UpdateCancelReason(ctx, orderID, req.Reason); err != nil {
		return nil, err
	}

	// Restore stock
	_ = s.restoreStock(ctx, order.Items)

	return s.getAndReturnOrder(ctx, orderID)
}

func (s *orderService) RequestReturn(orderID, customerID string, req *dto.ReturnOrderRequest) (*dto.OrderResponse, error) {
	ctx, cancel := util.NewDefaultDBContext()
	defer cancel()

	order, err := s.getOrderWithCustomerOwnership(ctx, orderID, customerID)
	if err != nil {
		return nil, err
	}

	if !order.CanReturn() {
		return nil, apperror.ErrOrderCannotBeReturned
	}

	history := model.StatusHistory{
		Status:    model.OrderStatusReturned,
		Note:      req.Reason,
		UpdatedBy: "customer",
		UpdatedAt: time.Now(),
	}

	if err := s.orderRepo.UpdateStatus(ctx, orderID, model.OrderStatusReturned, history); err != nil {
		return nil, err
	}

	return s.getAndReturnOrder(ctx, orderID)
}

// ==================== Seller Methods ====================

func (s *orderService) GetSellerOrders(sellerID string, query *dto.OrderFilterQuery) (*dto.PaginatedOrdersResponse, error) {
	ctx, cancel := util.NewDefaultDBContext()
	defer cancel()

	sellerObjID, err := primitive.ObjectIDFromHex(sellerID)
	if err != nil {
		return nil, apperror.ErrInvalidID
	}

	filter := s.buildOrderFilter(query)
	filter["seller_id"] = sellerObjID

	return s.findOrdersWithPagination(ctx, filter, query)
}

func (s *orderService) ConfirmOrder(orderID, sellerID string) (*dto.OrderResponse, error) {
	return s.updateSellerOrderStatus(orderID, sellerID, model.OrderStatusConfirmed, "Đơn hàng đã được xác nhận")
}

func (s *orderService) PackOrder(orderID, sellerID string) (*dto.OrderResponse, error) {
	return s.updateSellerOrderStatus(orderID, sellerID, model.OrderStatusPacked, "Đơn hàng đã được đóng gói")
}

func (s *orderService) ShipOrder(orderID, sellerID string, req *dto.UpdateTrackingRequest) (*dto.OrderResponse, error) {
	ctx, cancel := util.NewDefaultDBContext()
	defer cancel()

	order, err := s.getOrderWithSellerOwnership(ctx, orderID, sellerID)
	if err != nil {
		return nil, err
	}

	if !order.CanTransitionTo(model.OrderStatusShipped) {
		return nil, apperror.ErrInvalidOrderStatusTransition
	}

	if err := s.orderRepo.UpdateTracking(ctx, orderID, req.TrackingNumber, req.ShippingCarrier, req.EstimatedDelivery); err != nil {
		return nil, err
	}

	note := fmt.Sprintf("Đã giao cho %s, mã vận đơn: %s", req.ShippingCarrier, req.TrackingNumber)
	history := model.StatusHistory{
		Status:    model.OrderStatusShipped,
		Note:      note,
		UpdatedBy: "seller",
		UpdatedAt: time.Now(),
	}

	if err := s.orderRepo.UpdateStatus(ctx, orderID, model.OrderStatusShipped, history); err != nil {
		return nil, err
	}

	return s.getAndReturnOrder(ctx, orderID)
}

func (s *orderService) UpdateTracking(orderID, sellerID string, req *dto.UpdateTrackingRequest) (*dto.OrderResponse, error) {
	ctx, cancel := util.NewDefaultDBContext()
	defer cancel()

	order, err := s.getOrderWithSellerOwnership(ctx, orderID, sellerID)
	if err != nil {
		return nil, err
	}

	if order.Status != model.OrderStatusShipped {
		return nil, apperror.ErrInvalidOrderStatus
	}

	if err := s.orderRepo.UpdateTracking(ctx, orderID, req.TrackingNumber, req.ShippingCarrier, req.EstimatedDelivery); err != nil {
		return nil, err
	}

	return s.getAndReturnOrder(ctx, orderID)
}

func (s *orderService) RejectOrder(orderID, sellerID string, req *dto.RejectOrderRequest) (*dto.OrderResponse, error) {
	ctx, cancel := util.NewDefaultDBContext()
	defer cancel()

	order, err := s.getOrderWithSellerOwnership(ctx, orderID, sellerID)
	if err != nil {
		return nil, err
	}

	if !order.CanCancel() {
		return nil, apperror.ErrOrderCannotBeCanceled
	}

	history := model.StatusHistory{
		Status:    model.OrderStatusCanceled,
		Note:      req.Reason,
		UpdatedBy: "seller",
		UpdatedAt: time.Now(),
	}

	if err := s.orderRepo.UpdateStatus(ctx, orderID, model.OrderStatusCanceled, history); err != nil {
		return nil, err
	}

	if err := s.orderRepo.UpdateCancelReason(ctx, orderID, req.Reason); err != nil {
		return nil, err
	}

	// Restore stock
	_ = s.restoreStock(ctx, order.Items)

	return s.getAndReturnOrder(ctx, orderID)
}

func (s *orderService) ProcessRefund(orderID, sellerID string) (*dto.OrderResponse, error) {
	ctx, cancel := util.NewDefaultDBContext()
	defer cancel()

	order, err := s.getOrderWithSellerOwnership(ctx, orderID, sellerID)
	if err != nil {
		return nil, err
	}

	if order.Status != model.OrderStatusReturned {
		return nil, apperror.ErrInvalidOrderStatusTransition
	}

	history := model.StatusHistory{
		Status:    model.OrderStatusRefunded,
		Note:      "Đã hoàn tiền cho khách hàng",
		UpdatedBy: "seller",
		UpdatedAt: time.Now(),
	}

	if err := s.orderRepo.UpdateStatus(ctx, orderID, model.OrderStatusRefunded, history); err != nil {
		return nil, err
	}

	if err := s.orderRepo.UpdatePaymentStatus(ctx, orderID, model.PaymentStatusRefunded); err != nil {
		return nil, err
	}

	// Restore stock
	_ = s.restoreStock(ctx, order.Items)

	return s.getAndReturnOrder(ctx, orderID)
}

func (s *orderService) GetSellerOrderStats(sellerID string) (*dto.OrderStatsResponse, error) {
	ctx, cancel := util.NewDefaultDBContext()
	defer cancel()

	if _, err := primitive.ObjectIDFromHex(sellerID); err != nil {
		return nil, apperror.ErrInvalidID
	}

	stats, err := s.orderRepo.GetStats(ctx, &sellerID)
	if err != nil {
		return nil, err
	}

	return dto.FromOrderStats(stats), nil
}

func (s *orderService) updateSellerOrderStatus(orderID, sellerID string, status model.OrderStatus, note string) (*dto.OrderResponse, error) {
	ctx, cancel := util.NewDefaultDBContext()
	defer cancel()

	order, err := s.getOrderWithSellerOwnership(ctx, orderID, sellerID)
	if err != nil {
		return nil, err
	}

	if !order.CanTransitionTo(status) {
		return nil, apperror.ErrInvalidOrderStatusTransition
	}

	history := model.StatusHistory{
		Status:    status,
		Note:      note,
		UpdatedBy: "seller",
		UpdatedAt: time.Now(),
	}

	if err := s.orderRepo.UpdateStatus(ctx, orderID, status, history); err != nil {
		return nil, err
	}

	return s.getAndReturnOrder(ctx, orderID)
}

// ==================== Admin Methods ====================

func (s *orderService) GetAllOrders(query *dto.OrderFilterQuery) (*dto.PaginatedOrdersResponse, error) {
	ctx, cancel := util.NewDefaultDBContext()
	defer cancel()

	filter := s.buildOrderFilter(query)

	return s.findOrdersWithPagination(ctx, filter, query)
}

func (s *orderService) AdminUpdateStatus(orderID string, req *dto.UpdateOrderStatusRequest) (*dto.OrderResponse, error) {
	ctx, cancel := util.NewDefaultDBContext()
	defer cancel()

	order, err := s.getOrderByID(ctx, orderID)
	if err != nil {
		return nil, err
	}

	if !req.Status.IsValid() {
		return nil, apperror.ErrInvalidOrderStatus
	}

	oldStatus := order.Status

	history := model.StatusHistory{
		Status:    req.Status,
		Note:      req.Note,
		UpdatedBy: "admin",
		UpdatedAt: time.Now(),
	}

	if err := s.orderRepo.UpdateStatus(ctx, orderID, req.Status, history); err != nil {
		return nil, err
	}

	// Handle stock based on status change
	if req.Status == model.OrderStatusCanceled && oldStatus != model.OrderStatusCanceled {
		_ = s.restoreStock(ctx, order.Items)
	}

	return s.getAndReturnOrder(ctx, orderID)
}

func (s *orderService) AdminMarkAsDelivered(orderID string) (*dto.OrderResponse, error) {
	ctx, cancel := util.NewDefaultDBContext()
	defer cancel()

	order, err := s.getOrderByID(ctx, orderID)
	if err != nil {
		return nil, err
	}

	if order.Status != model.OrderStatusShipped {
		return nil, apperror.ErrInvalidOrderStatusTransition
	}

	history := model.StatusHistory{
		Status:    model.OrderStatusDelivered,
		Note:      "Đã giao hàng thành công",
		UpdatedBy: "admin",
		UpdatedAt: time.Now(),
	}

	if err := s.orderRepo.UpdateStatus(ctx, orderID, model.OrderStatusDelivered, history); err != nil {
		return nil, err
	}

	// If COD, mark as paid
	if order.PaymentMethod == model.PaymentMethodCOD {
		if err := s.orderRepo.UpdatePaymentStatus(ctx, orderID, model.PaymentStatusPaid); err != nil {
			return nil, err
		}
	}

	return s.getAndReturnOrder(ctx, orderID)
}

func (s *orderService) GetOrderStats() (*dto.OrderStatsResponse, error) {
	ctx, cancel := util.NewDefaultDBContext()
	defer cancel()

	stats, err := s.orderRepo.GetStats(ctx, nil)
	if err != nil {
		return nil, err
	}

	return dto.FromOrderStats(stats), nil
}

// ==================== Payment Methods ====================

func (s *orderService) UpdatePaymentStatus(orderID string, status model.PaymentStatus) error {
	ctx, cancel := util.NewDefaultDBContext()
	defer cancel()

	if _, err := s.getOrderByID(ctx, orderID); err != nil {
		return err
	}

	return s.orderRepo.UpdatePaymentStatus(ctx, orderID, status)
}

func (s *orderService) MarkAsPaid(orderID string) error {
	return s.UpdatePaymentStatus(orderID, model.PaymentStatusPaid)
}

// ==================== Helper Methods ====================

func (s *orderService) getOrderByID(ctx context.Context, orderID string) (*model.Order, error) {
	order, err := s.orderRepo.GetByID(ctx, orderID)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, apperror.ErrOrderNotFound
		}
		return nil, err
	}
	return order, nil
}

func (s *orderService) getAndReturnOrder(ctx context.Context, orderID string) (*dto.OrderResponse, error) {
	order, err := s.getOrderByID(ctx, orderID)
	if err != nil {
		return nil, err
	}
	return dto.FromOrder(order), nil
}

func (s *orderService) getOrderWithCustomerOwnership(ctx context.Context, orderID, customerID string) (*model.Order, error) {
	order, err := s.getOrderByID(ctx, orderID)
	if err != nil {
		return nil, err
	}

	customerObjID, err := primitive.ObjectIDFromHex(customerID)
	if err != nil {
		return nil, apperror.ErrInvalidID
	}

	if order.CustomerID != customerObjID {
		return nil, apperror.ErrForbidden
	}

	return order, nil
}

func (s *orderService) getOrderWithSellerOwnership(ctx context.Context, orderID, sellerID string) (*model.Order, error) {
	order, err := s.getOrderByID(ctx, orderID)
	if err != nil {
		return nil, err
	}

	sellerObjID, err := primitive.ObjectIDFromHex(sellerID)
	if err != nil {
		return nil, apperror.ErrInvalidID
	}

	if order.SellerID != sellerObjID {
		return nil, apperror.ErrForbidden
	}

	return order, nil
}

func (s *orderService) checkOrderAccess(order *model.Order, userID, role string) error {
	if role == string(model.AdminRole) {
		return nil
	}

	userObjID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return apperror.ErrInvalidID
	}

	if order.CustomerID != userObjID && order.SellerID != userObjID {
		return apperror.ErrForbidden
	}

	return nil
}

func (s *orderService) validatePlaceOrderRequest(req *dto.PlaceOrderRequest) error {
	if req == nil {
		return apperror.ErrBadRequest
	}

	if len(req.Items) == 0 {
		return apperror.ErrBadRequest
	}

	if len(req.Items) > MaxItemsPerOrder {
		return apperror.ErrBadRequest
	}

	for _, item := range req.Items {
		if item.Quantity <= 0 || item.Quantity > MaxQuantityPerItem {
			return apperror.ErrBadRequest
		}
		if item.ProductID == "" {
			return apperror.ErrInvalidID
		}
	}

	return nil
}

func (s *orderService) validateAndBuildOrderItems(ctx context.Context, items []dto.OrderItemRequest) ([]model.OrderItem, primitive.ObjectID, float64, error) {
	var orderItems []model.OrderItem
	var sellerID primitive.ObjectID
	var subtotal float64

	for i, item := range items {
		productObjID, err := primitive.ObjectIDFromHex(item.ProductID)
		if err != nil {
			return nil, primitive.NilObjectID, 0, apperror.ErrInvalidID
		}

		product, err := s.productRepo.GetByID(ctx, item.ProductID)
		if err != nil {
			if errors.Is(err, mongo.ErrNoDocuments) {
				return nil, primitive.NilObjectID, 0, apperror.ErrProductNotFound
			}
			return nil, primitive.NilObjectID, 0, err
		}

		if product.Status != model.ProductStatusActive {
			return nil, primitive.NilObjectID, 0, apperror.ErrProductNotAvailable
		}

		// Set seller ID from first product (one order = one seller)
		if i == 0 {
			sellerID = product.SellerID
		} else if sellerID != product.SellerID {
			return nil, primitive.NilObjectID, 0, apperror.ErrMultipleSellersNotAllowed
		}

		orderItem, err := s.buildOrderItem(product, productObjID, item)
		if err != nil {
			return nil, primitive.NilObjectID, 0, err
		}

		orderItems = append(orderItems, *orderItem)
		subtotal += orderItem.Subtotal
	}

	return orderItems, sellerID, subtotal, nil
}

func (s *orderService) buildOrderItem(product *model.Product, productObjID primitive.ObjectID, item dto.OrderItemRequest) (*model.OrderItem, error) {
	var price float64
	var stock int
	var sku string
	var image model.Image

	if item.VariantID != "" {
		variant, found := s.findVariant(product.Variants, item.VariantID)
		if !found {
			return nil, apperror.ErrVariantNotFound
		}

		price = variant.GetEffectivePrice()
		stock = variant.StockQuantity
		sku = variant.SKU

		if variant.Image != nil {
			image = *variant.Image
		}
	} else {
		if len(product.Variants) > 0 {
			return nil, apperror.ErrVariantRequired
		}

		price = product.GetEffectivePrice()
		stock = product.StockQuantity
		sku = product.SKU

		if len(product.Images) > 0 {
			image = product.Images[0]
		}
	}

	if stock < item.Quantity {
		return nil, apperror.ErrInsufficientStock
	}

	itemSubtotal := price * float64(item.Quantity)

	return &model.OrderItem{
		ProductID:   productObjID,
		VariantID:   item.VariantID,
		ProductName: product.Name,
		SKU:         sku,
		Image:       image,
		Price:       price,
		Quantity:    item.Quantity,
		Subtotal:    itemSubtotal,
	}, nil
}

func (s *orderService) validateAndBuildShippingAddress(ctx context.Context, req *dto.ShippingAddressRequest) (*model.ShippingAddress, error) {
	if req == nil {
		return nil, apperror.ErrBadRequest
	}

	if req.RecipientName == "" || req.PhoneNumber == "" {
		return nil, apperror.ErrBadRequest
	}

	provinceObjID, err := primitive.ObjectIDFromHex(req.ProvinceID)
	if err != nil {
		return nil, apperror.ErrInvalidID
	}

	province, err := s.provinceRepo.GetByID(ctx, provinceObjID)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, apperror.ErrProvinceNotFound
		}
		return nil, err
	}

	ward, err := s.provinceRepo.GetWardByID(ctx, provinceObjID, req.WardID)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, apperror.ErrWardNotFound
		}
		return nil, err
	}

	address := &model.ShippingAddress{
		RecipientName: req.RecipientName,
		PhoneNumber:   req.PhoneNumber,
		ProvinceID:    provinceObjID,
		ProvinceName:  province.Name,
		WardID:        req.WardID,
		WardName:      ward.Name,
		Detail:        req.Detail,
	}
	address.BuildFullAddress()

	return address, nil
}

func (s *orderService) decreaseStock(ctx context.Context, items []dto.OrderItemRequest) error {
	for i, item := range items {
		product, err := s.productRepo.GetByID(ctx, item.ProductID)
		if err != nil {
			s.rollbackDecreasedStock(ctx, items[:i])
			return err
		}

		var newStock int
		if item.VariantID != "" {
			variant, found := s.findVariant(product.Variants, item.VariantID)
			if !found {
				s.rollbackDecreasedStock(ctx, items[:i])
				return apperror.ErrVariantNotFound
			}
			newStock = variant.StockQuantity - item.Quantity
			if newStock < 0 {
				s.rollbackDecreasedStock(ctx, items[:i])
				return apperror.ErrInsufficientStock
			}
			if err := s.productRepo.UpdateVariantStock(ctx, item.ProductID, item.VariantID, newStock); err != nil {
				s.rollbackDecreasedStock(ctx, items[:i])
				return err
			}
		} else {
			newStock = product.StockQuantity - item.Quantity
			if newStock < 0 {
				s.rollbackDecreasedStock(ctx, items[:i])
				return apperror.ErrInsufficientStock
			}
			if err := s.productRepo.UpdateStock(ctx, item.ProductID, newStock); err != nil {
				s.rollbackDecreasedStock(ctx, items[:i])
				return err
			}
		}
	}
	return nil
}

func (s *orderService) rollbackDecreasedStock(ctx context.Context, items []dto.OrderItemRequest) {
	for _, item := range items {
		product, err := s.productRepo.GetByID(ctx, item.ProductID)
		if err != nil {
			continue
		}

		if item.VariantID != "" {
			variant, found := s.findVariant(product.Variants, item.VariantID)
			if found {
				_ = s.productRepo.UpdateVariantStock(ctx, item.ProductID, item.VariantID, variant.StockQuantity+item.Quantity)
			}
		} else {
			_ = s.productRepo.UpdateStock(ctx, item.ProductID, product.StockQuantity+item.Quantity)
		}
	}
}

func (s *orderService) restoreStock(ctx context.Context, items []model.OrderItem) error {
	var lastErr error
	for _, item := range items {
		productID := item.ProductID.Hex()

		product, err := s.productRepo.GetByID(ctx, productID)
		if err != nil {
			lastErr = err
			continue
		}

		if item.VariantID != "" {
			variant, found := s.findVariant(product.Variants, item.VariantID)
			if found {
				if err := s.productRepo.UpdateVariantStock(ctx, productID, item.VariantID, variant.StockQuantity+item.Quantity); err != nil {
					lastErr = err
				}
			}
		} else {
			if err := s.productRepo.UpdateStock(ctx, productID, product.StockQuantity+item.Quantity); err != nil {
				lastErr = err
			}
		}
	}
	return lastErr
}

func (s *orderService) findOrdersWithPagination(ctx context.Context, filter repo.Filter, query *dto.OrderFilterQuery) (*dto.PaginatedOrdersResponse, error) {
	page, pageSize := s.normalizePagination(query.Page, query.PageSize)

	findOptions := &repo.FindOptions{
		Skip:  int64((page - 1) * pageSize),
		Limit: int64(pageSize),
		Sort:  s.buildSortOptions(query.SortBy, query.SortOrder),
	}

	orders, total, err := s.orderRepo.Find(ctx, filter, findOptions)
	if err != nil {
		return nil, err
	}

	return &dto.PaginatedOrdersResponse{
		Orders: dto.FromOrderList(orders),
		Pagination: dto.Pagination{
			Page:     page,
			PageSize: pageSize,
			Total:    total,
		},
	}, nil
}

func (s *orderService) buildOrderFilter(query *dto.OrderFilterQuery) repo.Filter {
	filter := repo.Filter{}

	if query == nil {
		return filter
	}

	if query.Status != "" {
		filter["status"] = model.OrderStatus(query.Status)
	}

	if query.PaymentStatus != "" {
		filter["payment_status"] = model.PaymentStatus(query.PaymentStatus)
	}

	if query.PaymentMethod != "" {
		filter["payment_method"] = model.PaymentMethod(query.PaymentMethod)
	}

	if query.Search != "" {
		filter["$or"] = []bson.M{
			{"order_number": bson.M{"$regex": primitive.Regex{Pattern: query.Search, Options: "i"}}},
			{"items.product_name": bson.M{"$regex": primitive.Regex{Pattern: query.Search, Options: "i"}}},
		}
	}

	if query.StartDate != "" && query.EndDate != "" {
		startDate, err1 := time.Parse("2006-01-02", query.StartDate)
		endDate, err2 := time.Parse("2006-01-02", query.EndDate)
		if err1 == nil && err2 == nil {
			endDate = endDate.Add(24 * time.Hour)
			filter["created_at"] = bson.M{
				"$gte": startDate,
				"$lt":  endDate,
			}
		}
	}

	if query.CustomerID != "" {
		if customerObjID, err := primitive.ObjectIDFromHex(query.CustomerID); err == nil {
			filter["customer_id"] = customerObjID
		}
	}

	if query.SellerID != "" {
		if sellerObjID, err := primitive.ObjectIDFromHex(query.SellerID); err == nil {
			filter["seller_id"] = sellerObjID
		}
	}

	return filter
}

func (s *orderService) buildSortOptions(sortBy, sortOrder string) map[string]int {
	if sortBy == "" {
		sortBy = "created_at"
	}

	order := -1
	if sortOrder == "asc" {
		order = 1
	}

	return map[string]int{sortBy: order}
}

func (s *orderService) normalizePagination(page, pageSize int) (int, int) {
	if page < 1 {
		page = 1
	}

	if pageSize < 1 {
		pageSize = DefaultPageSize
	}

	if pageSize > MaxPageSize {
		pageSize = MaxPageSize
	}

	return page, pageSize
}

func (s *orderService) calculateShippingFee(subtotal float64) float64 {
	if subtotal >= FreeShippingThreshold {
		return 0
	}
	return DefaultShippingFee
}

func (s *orderService) findVariant(variants []model.ProductVariant, variantID string) (*model.ProductVariant, bool) {
	variantObjID, err := primitive.ObjectIDFromHex(variantID)
	if err != nil {
		return nil, false
	}

	for i := range variants {
		if variants[i].ID == variantObjID {
			return &variants[i], true
		}
	}

	return nil, false
}
