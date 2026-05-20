package service

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/luong-vh/Digimart_Backend/internal/apperror"
	"github.com/luong-vh/Digimart_Backend/internal/config"
	"github.com/luong-vh/Digimart_Backend/internal/dto"
	"github.com/luong-vh/Digimart_Backend/internal/model"
	"github.com/luong-vh/Digimart_Backend/internal/repo"
	"github.com/luong-vh/Digimart_Backend/internal/util"
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
	UpdatePaymentMethod(orderID, customerID string, req *dto.UpdatePaymentMethodRequest) (*dto.OrderResponse, error)

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
	CreateZaloPayPayment(orderID, userID, role string) (*dto.ZaloPayPaymentResponse, error)
	SyncZaloPayPayment(orderID, userID, role string, req *dto.ZaloPaySyncRequest) (*dto.OrderResponse, error)
	HandleZaloPayCallback(req *dto.ZaloPayCallbackRequest) error
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

func (s *orderService) UpdatePaymentMethod(orderID, customerID string, req *dto.UpdatePaymentMethodRequest) (*dto.OrderResponse, error) {
	ctx, cancel := util.NewDefaultDBContext()
	defer cancel()

	if req == nil || !req.PaymentMethod.IsValid() {
		return nil, apperror.ErrInvalidPaymentMethod
	}

	order, err := s.getOrderWithCustomerOwnership(ctx, orderID, customerID)
	if err != nil {
		return nil, err
	}

	if order.PaymentStatus == model.PaymentStatusPaid ||
		order.PaymentStatus == model.PaymentStatusRefunded ||
		order.Status == model.OrderStatusCanceled ||
		order.Status == model.OrderStatusReturned ||
		order.Status == model.OrderStatusRefunded {
		return nil, apperror.ErrInvalidOrderStatusTransition
	}

	if err := s.orderRepo.UpdatePaymentMethod(ctx, orderID, req.PaymentMethod); err != nil {
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

	stats, err := s.orderRepo.GetStats(ctx, nil)
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
	ctx, cancel := util.NewDefaultDBContext()
	defer cancel()

	order, err := s.getOrderByID(ctx, orderID)
	if err != nil {
		return err
	}

	if order.Status == model.OrderStatusCanceled ||
		order.Status == model.OrderStatusReturned ||
		order.Status == model.OrderStatusRefunded ||
		order.PaymentStatus == model.PaymentStatusRefunded {
		return apperror.ErrInvalidOrderStatusTransition
	}

	if order.PaymentStatus == model.PaymentStatusPaid {
		return nil
	}

	return s.orderRepo.UpdatePaymentStatus(ctx, orderID, model.PaymentStatusPaid)
}

func (s *orderService) CreateZaloPayPayment(orderID, userID, role string) (*dto.ZaloPayPaymentResponse, error) {
	ctx, cancel := util.NewDefaultDBContext()
	defer cancel()

	order, err := s.getOrderByID(ctx, orderID)
	if err != nil {
		return nil, err
	}
	if err := s.checkOrderAccess(order, userID, role); err != nil {
		return nil, err
	}
	if order.PaymentStatus == model.PaymentStatusPaid {
		return nil, apperror.ErrInvalidOrderStatusTransition
	}
	if order.Status == model.OrderStatusCanceled || order.Status == model.OrderStatusReturned || order.Status == model.OrderStatusRefunded {
		return nil, apperror.ErrInvalidOrderStatusTransition
	}

	appTransID := fmt.Sprintf("%s_%s_%06d", time.Now().Format("060102"), order.ID.Hex(), time.Now().UnixNano()%1000000)
	appTime := time.Now().UnixMilli()
	amount := int64(math.Round(order.Total))

	items := make([]map[string]interface{}, 0, len(order.Items))
	for _, item := range order.Items {
		items = append(items, map[string]interface{}{
			"itemid":       item.ProductID.Hex(),
			"itemname":     item.ProductName,
			"itemprice":    int64(math.Round(item.Price)),
			"itemquantity": item.Quantity,
		})
	}

	itemJSON, err := json.Marshal(items)
	if err != nil {
		return nil, err
	}

	embedData := map[string]interface{}{
		"order_id":    order.ID.Hex(),
		"redirecturl": config.Cfg.ZaloPay.RedirectURL,
	}
	preferredMethods := preferredZaloPayMethods(order.PaymentMethod)
	if len(preferredMethods) > 0 {
		embedData["preferred_payment_method"] = preferredMethods
	}
	embedJSON, err := json.Marshal(embedData)
	if err != nil {
		return nil, err
	}

	appUser := order.CustomerID.Hex()
	description := fmt.Sprintf("DigiMart - Thanh toan don hang %s", order.OrderNumber)
	macInput := fmt.Sprintf("%d|%s|%s|%d|%d|%s|%s",
		config.Cfg.ZaloPay.AppID,
		appTransID,
		appUser,
		amount,
		appTime,
		string(embedJSON),
		string(itemJSON),
	)

	form := url.Values{}
	form.Set("app_id", strconv.Itoa(config.Cfg.ZaloPay.AppID))
	form.Set("app_trans_id", appTransID)
	form.Set("app_user", appUser)
	form.Set("app_time", strconv.FormatInt(appTime, 10))
	form.Set("amount", strconv.FormatInt(amount, 10))
	form.Set("description", description)
	form.Set("item", string(itemJSON))
	form.Set("embed_data", string(embedJSON))
	form.Set("bank_code", "")
	form.Set("mac", hmacSHA256Hex(config.Cfg.ZaloPay.Key1, macInput))
	if config.Cfg.ZaloPay.CallbackURL != "" {
		form.Set("callback_url", config.Cfg.ZaloPay.CallbackURL)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, config.Cfg.ZaloPay.CreateURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	httpResp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer httpResp.Body.Close()

	body, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, err
	}
	if httpResp.StatusCode < 200 || httpResp.StatusCode >= 300 {
		return nil, fmt.Errorf("zalopay create order failed: %s", string(body))
	}

	var result dto.ZaloPayPaymentResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}
	result.OrderID = order.ID.Hex()
	result.AppTransID = appTransID
	if result.ReturnCode != 1 {
		return &result, fmt.Errorf("zalopay create order failed: %s", result.ReturnMessage)
	}

	return &result, nil
}

func (s *orderService) HandleZaloPayCallback(req *dto.ZaloPayCallbackRequest) error {
	if req == nil || req.Data == "" || req.Mac == "" {
		return apperror.ErrBadRequest
	}

	expectedMac := hmacSHA256Hex(config.Cfg.ZaloPay.Key2, req.Data)
	if !hmac.Equal([]byte(strings.ToLower(expectedMac)), []byte(strings.ToLower(req.Mac))) {
		return apperror.ErrInvalidToken
	}

	var data struct {
		AppTransID string `json:"app_trans_id"`
		Amount     int64  `json:"amount"`
	}
	if err := json.Unmarshal([]byte(req.Data), &data); err != nil {
		return err
	}

	orderID := orderIDFromZaloPayTransID(data.AppTransID)
	if orderID == "" {
		return apperror.ErrInvalidID
	}

	ctx, cancel := util.NewDefaultDBContext()
	defer cancel()

	order, err := s.getOrderByID(ctx, orderID)
	if err != nil {
		return err
	}
	if int64(math.Round(order.Total)) != data.Amount {
		return apperror.ErrBadRequest
	}

	return s.MarkAsPaid(orderID)
}

func (s *orderService) SyncZaloPayPayment(orderID, userID, role string, req *dto.ZaloPaySyncRequest) (*dto.OrderResponse, error) {
	if req == nil || req.AppTransID == "" {
		return nil, apperror.ErrBadRequest
	}

	if parsedOrderID := orderIDFromZaloPayTransID(req.AppTransID); parsedOrderID != orderID {
		return nil, apperror.ErrInvalidID
	}

	ctx, cancel := util.NewDefaultDBContext()
	defer cancel()

	order, err := s.getOrderByID(ctx, orderID)
	if err != nil {
		return nil, err
	}
	if err := s.checkOrderAccess(order, userID, role); err != nil {
		return nil, err
	}
	if order.PaymentStatus == model.PaymentStatusPaid {
		return dto.FromOrder(order), nil
	}

	appID := strconv.Itoa(config.Cfg.ZaloPay.AppID)
	macInput := fmt.Sprintf("%s|%s|%s", appID, req.AppTransID, config.Cfg.ZaloPay.Key1)

	form := url.Values{}
	form.Set("app_id", appID)
	form.Set("app_trans_id", req.AppTransID)
	form.Set("mac", hmacSHA256Hex(config.Cfg.ZaloPay.Key1, macInput))

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, config.Cfg.ZaloPay.QueryURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	httpResp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer httpResp.Body.Close()

	body, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, err
	}
	if httpResp.StatusCode < 200 || httpResp.StatusCode >= 300 {
		return nil, fmt.Errorf("zalopay query order failed: %s", string(body))
	}

	var result struct {
		ReturnCode    int    `json:"return_code"`
		ReturnMessage string `json:"return_message"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	switch result.ReturnCode {
	case 1:
		if err := s.orderRepo.UpdatePaymentStatus(ctx, orderID, model.PaymentStatusPaid); err != nil {
			return nil, err
		}
	case 2:
		if err := s.orderRepo.UpdatePaymentStatus(ctx, orderID, model.PaymentStatusFailed); err != nil {
			return nil, err
		}
	case 3:
		// ZaloPay is still processing; keep the local status as-is.
	default:
		return nil, fmt.Errorf("zalopay query order failed: %s", result.ReturnMessage)
	}

	return s.getAndReturnOrder(ctx, orderID)
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

	if _, err := primitive.ObjectIDFromHex(sellerID); err != nil {
		return nil, apperror.ErrInvalidID
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

	for _, item := range items {
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

		orderItem, err := s.buildOrderItem(product, productObjID, item)
		if err != nil {
			return nil, primitive.NilObjectID, 0, err
		}

		orderItems = append(orderItems, *orderItem)
		subtotal += orderItem.Subtotal
	}

	adminSellerID, err := s.getDefaultSellerID(ctx)
	if err != nil {
		return nil, primitive.NilObjectID, 0, err
	}
	sellerID = adminSellerID

	return orderItems, sellerID, subtotal, nil
}

func (s *orderService) buildOrderItem(product *model.Product, productObjID primitive.ObjectID, item dto.OrderItemRequest) (*model.OrderItem, error) {
	price := product.BasePrice
	sku := productObjID.Hex()

	if product.IsHasVariant {
		if item.VariantID == "" {
			return nil, apperror.ErrVariantRequired
		}

		variant, ok := s.findVariant(product.Variants, item.VariantID)
		if !ok {
			return nil, apperror.ErrVariantNotFound
		}
		if variant.StockQuantity < item.Quantity {
			return nil, apperror.ErrInsufficientStock
		}

		price = variant.FinalPrice
		if price <= 0 {
			price = product.BasePrice + variant.PriceAdjustment
		}
		sku = variant.ID.Hex()
	} else {
		if product.StockQuantity == nil || *product.StockQuantity < item.Quantity {
			return nil, apperror.ErrInsufficientStock
		}
	}

	if product.DiscountPercent > 0 {
		price = price * (100 - product.DiscountPercent) / 100
	}

	orderItem := &model.OrderItem{
		ProductID:   productObjID,
		VariantID:   item.VariantID,
		ProductName: product.Name,
		SKU:         sku,
		Image:       product.Thumbnail,
		Price:       price,
		Quantity:    item.Quantity,
	}
	orderItem.CalculateSubtotal()

	return orderItem, nil
}

func (s *orderService) validateAndBuildShippingAddress(ctx context.Context, req *dto.ShippingAddressRequest) (*model.ShippingAddress, error) {
	if req == nil || req.RecipientName == "" || req.PhoneNumber == "" || req.ProvinceID == "" || req.WardID == "" || req.Detail == "" {
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
		ProvinceID:    province.ID,
		ProvinceName:  province.Name,
		WardID:        ward.ID,
		WardName:      ward.Name,
		Detail:        req.Detail,
	}
	address.BuildFullAddress()

	return address, nil
}

func (s *orderService) decreaseStock(ctx context.Context, items []dto.OrderItemRequest) error {
	decreased := make([]dto.OrderItemRequest, 0, len(items))

	for _, item := range items {
		product, err := s.productRepo.GetByID(ctx, item.ProductID)
		if err != nil {
			s.rollbackDecreasedStock(ctx, decreased)
			return err
		}

		if item.VariantID != "" {
			variantObjID, err := primitive.ObjectIDFromHex(item.VariantID)
			if err != nil {
				s.rollbackDecreasedStock(ctx, decreased)
				return apperror.ErrInvalidID
			}

			found := false
			for i := range product.Variants {
				if product.Variants[i].ID == variantObjID {
					if product.Variants[i].StockQuantity < item.Quantity {
						s.rollbackDecreasedStock(ctx, decreased)
						return apperror.ErrInsufficientStock
					}
					product.Variants[i].StockQuantity -= item.Quantity
					found = true
					break
				}
			}
			if !found {
				s.rollbackDecreasedStock(ctx, decreased)
				return apperror.ErrVariantNotFound
			}
		} else {
			if product.StockQuantity == nil || *product.StockQuantity < item.Quantity {
				s.rollbackDecreasedStock(ctx, decreased)
				return apperror.ErrInsufficientStock
			}
			stock := *product.StockQuantity - item.Quantity
			product.StockQuantity = &stock
		}

		product.SoldCount += item.Quantity
		if _, err := s.productRepo.Update(ctx, product); err != nil {
			s.rollbackDecreasedStock(ctx, decreased)
			return err
		}

		decreased = append(decreased, item)
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
			variantObjID, err := primitive.ObjectIDFromHex(item.VariantID)
			if err != nil {
				continue
			}
			for i := range product.Variants {
				if product.Variants[i].ID == variantObjID {
					product.Variants[i].StockQuantity += item.Quantity
					break
				}
			}
		} else if product.StockQuantity != nil {
			stock := *product.StockQuantity + item.Quantity
			product.StockQuantity = &stock
		}

		product.SoldCount -= item.Quantity
		if product.SoldCount < 0 {
			product.SoldCount = 0
		}
		_, _ = s.productRepo.Update(ctx, product)
	}
}

func (s *orderService) restoreStock(ctx context.Context, items []model.OrderItem) error {
	for _, item := range items {
		product, err := s.productRepo.GetByID(ctx, item.ProductID.Hex())
		if err != nil {
			return err
		}

		if item.VariantID != "" {
			variantObjID, err := primitive.ObjectIDFromHex(item.VariantID)
			if err != nil {
				return apperror.ErrInvalidID
			}
			for i := range product.Variants {
				if product.Variants[i].ID == variantObjID {
					product.Variants[i].StockQuantity += item.Quantity
					break
				}
			}
		} else if product.StockQuantity != nil {
			stock := *product.StockQuantity + item.Quantity
			product.StockQuantity = &stock
		}

		product.SoldCount -= item.Quantity
		if product.SoldCount < 0 {
			product.SoldCount = 0
		}
		if _, err := s.productRepo.Update(ctx, product); err != nil {
			return err
		}
	}

	return nil
}

func (s *orderService) getDefaultSellerID(ctx context.Context) (primitive.ObjectID, error) {
	admins, _, err := s.userRepo.Find(ctx, repo.Filter{
		"role":       model.AdminRole,
		"deleted_at": bson.M{"$exists": false},
	}, &repo.FindOptions{
		Limit: 1,
		Sort:  map[string]int{"created_at": 1},
	})
	if err != nil {
		return primitive.NilObjectID, err
	}
	if len(admins) == 0 {
		return primitive.NilObjectID, apperror.ErrAdminAccessRequired
	}

	return admins[0].ID, nil
}

func hmacSHA256Hex(key, data string) string {
	mac := hmac.New(sha256.New, []byte(key))
	mac.Write([]byte(data))
	return hex.EncodeToString(mac.Sum(nil))
}

func orderIDFromZaloPayTransID(appTransID string) string {
	parts := strings.Split(appTransID, "_")
	if len(parts) < 2 {
		return ""
	}
	return parts[1]
}

func preferredZaloPayMethods(method model.PaymentMethod) []string {
	switch method {
	case model.PaymentMethodBankTransfer:
		return []string{"vietqr", "domestic_card"}
	case model.PaymentMethodEWallet:
		return []string{"zalopay_wallet"}
	case model.PaymentMethodCreditCard:
		return []string{"international_card"}
	default:
		return nil
	}
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
