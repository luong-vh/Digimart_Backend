package controller

import (
	"net/http"

	"github.com/DucLove1/SE357-ShoppingManagement-BE/internal/apperror"
	"github.com/DucLove1/SE357-ShoppingManagement-BE/internal/auth"
	"github.com/DucLove1/SE357-ShoppingManagement-BE/internal/dto"
	"github.com/DucLove1/SE357-ShoppingManagement-BE/internal/model"
	"github.com/DucLove1/SE357-ShoppingManagement-BE/internal/service"
	"github.com/gin-gonic/gin"
)

// OrderController handles requests related to order management.
type OrderController struct {
	service service.OrderService
}

// NewOrderController creates a new OrderController.
func NewOrderController(service service.OrderService) *OrderController {
	return &OrderController{service: service}
}

// ==================== Customer Endpoints ====================

// PlaceOrder creates a new order for the authenticated customer.
func (c *OrderController) PlaceOrder(ctx *gin.Context) {
	authUser, exists := ctx.Get("authUser")
	if !exists {
		dto.SendError(ctx, http.StatusUnauthorized, apperror.ErrNotAuthenticated.Message, apperror.ErrNotAuthenticated.Code)
		return
	}

	var req dto.PlaceOrderRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		dto.SendError(ctx, http.StatusBadRequest, apperror.ErrBadRequest.Message, apperror.ErrBadRequest.Code)
		return
	}

	order, err := c.service.PlaceOrder(authUser.(auth.AuthUser).ID, &req)
	if err != nil {
		dto.SendError(ctx, apperror.StatusFromError(err), apperror.Message(err), apperror.Code(err))
		return
	}

	dto.SendSuccess(ctx, http.StatusCreated, "Order placed successfully", order)
}

// GetMyOrders retrieves all orders for the authenticated customer.
func (c *OrderController) GetMyOrders(ctx *gin.Context) {
	authUser, exists := ctx.Get("authUser")
	if !exists {
		dto.SendError(ctx, http.StatusUnauthorized, apperror.ErrNotAuthenticated.Message, apperror.ErrNotAuthenticated.Code)
		return
	}

	var query dto.OrderFilterQuery
	if err := ctx.ShouldBindQuery(&query); err != nil {
		dto.SendError(ctx, http.StatusBadRequest, apperror.ErrBadRequest.Message, apperror.ErrBadRequest.Code)
		return
	}

	orders, err := c.service.GetMyOrders(authUser.(auth.AuthUser).ID, &query)
	if err != nil {
		dto.SendError(ctx, apperror.StatusFromError(err), apperror.Message(err), apperror.Code(err))
		return
	}

	dto.SendSuccess(ctx, http.StatusOK, "Orders retrieved successfully", orders)
}

// GetOrderByID retrieves a specific order by ID.
func (c *OrderController) GetOrderByID(ctx *gin.Context) {
	authUser, exists := ctx.Get("authUser")
	if !exists {
		dto.SendError(ctx, http.StatusUnauthorized, apperror.ErrNotAuthenticated.Message, apperror.ErrNotAuthenticated.Code)
		return
	}

	orderID := ctx.Param("id")
	if orderID == "" {
		dto.SendError(ctx, http.StatusBadRequest, apperror.ErrInvalidID.Message, apperror.ErrInvalidID.Code)
		return
	}

	user := authUser.(auth.AuthUser)
	order, err := c.service.GetOrderByID(orderID, user.ID, user.Role)
	if err != nil {
		dto.SendError(ctx, apperror.StatusFromError(err), apperror.Message(err), apperror.Code(err))
		return
	}

	dto.SendSuccess(ctx, http.StatusOK, "Order retrieved successfully", order)
}

// GetOrderByNumber retrieves a specific order by order number.
func (c *OrderController) GetOrderByNumber(ctx *gin.Context) {
	authUser, exists := ctx.Get("authUser")
	if !exists {
		dto.SendError(ctx, http.StatusUnauthorized, apperror.ErrNotAuthenticated.Message, apperror.ErrNotAuthenticated.Code)
		return
	}

	orderNumber := ctx.Param("orderNumber")
	if orderNumber == "" {
		dto.SendError(ctx, http.StatusBadRequest, apperror.ErrBadRequest.Message, apperror.ErrBadRequest.Code)
		return
	}

	user := authUser.(auth.AuthUser)
	order, err := c.service.GetOrderByNumber(orderNumber, user.ID, user.Role)
	if err != nil {
		dto.SendError(ctx, apperror.StatusFromError(err), apperror.Message(err), apperror.Code(err))
		return
	}

	dto.SendSuccess(ctx, http.StatusOK, "Order retrieved successfully", order)
}

// CancelOrder cancels an order by the customer.
func (c *OrderController) CancelOrder(ctx *gin.Context) {
	authUser, exists := ctx.Get("authUser")
	if !exists {
		dto.SendError(ctx, http.StatusUnauthorized, apperror.ErrNotAuthenticated.Message, apperror.ErrNotAuthenticated.Code)
		return
	}

	orderID := ctx.Param("id")
	if orderID == "" {
		dto.SendError(ctx, http.StatusBadRequest, apperror.ErrInvalidID.Message, apperror.ErrInvalidID.Code)
		return
	}

	var req dto.CancelOrderRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		dto.SendError(ctx, http.StatusBadRequest, apperror.ErrBadRequest.Message, apperror.ErrBadRequest.Code)
		return
	}

	order, err := c.service.CancelOrder(orderID, authUser.(auth.AuthUser).ID, &req)
	if err != nil {
		dto.SendError(ctx, apperror.StatusFromError(err), apperror.Message(err), apperror.Code(err))
		return
	}

	dto.SendSuccess(ctx, http.StatusOK, "Order canceled successfully", order)
}

// RequestReturn requests a return for a delivered order.
func (c *OrderController) RequestReturn(ctx *gin.Context) {
	authUser, exists := ctx.Get("authUser")
	if !exists {
		dto.SendError(ctx, http.StatusUnauthorized, apperror.ErrNotAuthenticated.Message, apperror.ErrNotAuthenticated.Code)
		return
	}

	orderID := ctx.Param("id")
	if orderID == "" {
		dto.SendError(ctx, http.StatusBadRequest, apperror.ErrInvalidID.Message, apperror.ErrInvalidID.Code)
		return
	}

	var req dto.ReturnOrderRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		dto.SendError(ctx, http.StatusBadRequest, apperror.ErrBadRequest.Message, apperror.ErrBadRequest.Code)
		return
	}

	order, err := c.service.RequestReturn(orderID, authUser.(auth.AuthUser).ID, &req)
	if err != nil {
		dto.SendError(ctx, apperror.StatusFromError(err), apperror.Message(err), apperror.Code(err))
		return
	}

	dto.SendSuccess(ctx, http.StatusOK, "Return request submitted successfully", order)
}

// ==================== Seller Endpoints ====================

// GetSellerOrders retrieves all orders for the authenticated seller.
func (c *OrderController) GetSellerOrders(ctx *gin.Context) {
	authUser, exists := ctx.Get("authUser")
	if !exists {
		dto.SendError(ctx, http.StatusUnauthorized, apperror.ErrNotAuthenticated.Message, apperror.ErrNotAuthenticated.Code)
		return
	}

	user := authUser.(auth.AuthUser)
	if user.Role != string(model.SellerRole) {
		dto.SendError(ctx, http.StatusForbidden, apperror.ErrSellerAccessRequired.Message, apperror.ErrSellerAccessRequired.Code)
		return
	}

	var query dto.OrderFilterQuery
	if err := ctx.ShouldBindQuery(&query); err != nil {
		dto.SendError(ctx, http.StatusBadRequest, apperror.ErrBadRequest.Message, apperror.ErrBadRequest.Code)
		return
	}

	orders, err := c.service.GetSellerOrders(user.ID, &query)
	if err != nil {
		dto.SendError(ctx, apperror.StatusFromError(err), apperror.Message(err), apperror.Code(err))
		return
	}

	dto.SendSuccess(ctx, http.StatusOK, "Seller orders retrieved successfully", orders)
}

// ConfirmOrder confirms an order by the seller.
func (c *OrderController) ConfirmOrder(ctx *gin.Context) {
	authUser, exists := ctx.Get("authUser")
	if !exists {
		dto.SendError(ctx, http.StatusUnauthorized, apperror.ErrNotAuthenticated.Message, apperror.ErrNotAuthenticated.Code)
		return
	}

	user := authUser.(auth.AuthUser)
	if user.Role != string(model.SellerRole) {
		dto.SendError(ctx, http.StatusForbidden, apperror.ErrSellerAccessRequired.Message, apperror.ErrSellerAccessRequired.Code)
		return
	}

	orderID := ctx.Param("id")
	if orderID == "" {
		dto.SendError(ctx, http.StatusBadRequest, apperror.ErrInvalidID.Message, apperror.ErrInvalidID.Code)
		return
	}

	order, err := c.service.ConfirmOrder(orderID, user.ID)
	if err != nil {
		dto.SendError(ctx, apperror.StatusFromError(err), apperror.Message(err), apperror.Code(err))
		return
	}

	dto.SendSuccess(ctx, http.StatusOK, "Order confirmed successfully", order)
}

// PackOrder marks an order as packed by the seller.
func (c *OrderController) PackOrder(ctx *gin.Context) {
	authUser, exists := ctx.Get("authUser")
	if !exists {
		dto.SendError(ctx, http.StatusUnauthorized, apperror.ErrNotAuthenticated.Message, apperror.ErrNotAuthenticated.Code)
		return
	}

	user := authUser.(auth.AuthUser)
	if user.Role != string(model.SellerRole) {
		dto.SendError(ctx, http.StatusForbidden, apperror.ErrSellerAccessRequired.Message, apperror.ErrSellerAccessRequired.Code)
		return
	}

	orderID := ctx.Param("id")
	if orderID == "" {
		dto.SendError(ctx, http.StatusBadRequest, apperror.ErrInvalidID.Message, apperror.ErrInvalidID.Code)
		return
	}

	order, err := c.service.PackOrder(orderID, user.ID)
	if err != nil {
		dto.SendError(ctx, apperror.StatusFromError(err), apperror.Message(err), apperror.Code(err))
		return
	}

	dto.SendSuccess(ctx, http.StatusOK, "Order packed successfully", order)
}

// ShipOrder marks an order as shipped by the seller.
func (c *OrderController) ShipOrder(ctx *gin.Context) {
	authUser, exists := ctx.Get("authUser")
	if !exists {
		dto.SendError(ctx, http.StatusUnauthorized, apperror.ErrNotAuthenticated.Message, apperror.ErrNotAuthenticated.Code)
		return
	}

	user := authUser.(auth.AuthUser)
	if user.Role != string(model.SellerRole) {
		dto.SendError(ctx, http.StatusForbidden, apperror.ErrSellerAccessRequired.Message, apperror.ErrSellerAccessRequired.Code)
		return
	}

	orderID := ctx.Param("id")
	if orderID == "" {
		dto.SendError(ctx, http.StatusBadRequest, apperror.ErrInvalidID.Message, apperror.ErrInvalidID.Code)
		return
	}

	var req dto.UpdateTrackingRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		dto.SendError(ctx, http.StatusBadRequest, apperror.ErrBadRequest.Message, apperror.ErrBadRequest.Code)
		return
	}

	order, err := c.service.ShipOrder(orderID, user.ID, &req)
	if err != nil {
		dto.SendError(ctx, apperror.StatusFromError(err), apperror.Message(err), apperror.Code(err))
		return
	}

	dto.SendSuccess(ctx, http.StatusOK, "Order shipped successfully", order)
}

// UpdateTracking updates tracking information for a shipped order.
func (c *OrderController) UpdateTracking(ctx *gin.Context) {
	authUser, exists := ctx.Get("authUser")
	if !exists {
		dto.SendError(ctx, http.StatusUnauthorized, apperror.ErrNotAuthenticated.Message, apperror.ErrNotAuthenticated.Code)
		return
	}

	user := authUser.(auth.AuthUser)
	if user.Role != string(model.SellerRole) {
		dto.SendError(ctx, http.StatusForbidden, apperror.ErrSellerAccessRequired.Message, apperror.ErrSellerAccessRequired.Code)
		return
	}

	orderID := ctx.Param("id")
	if orderID == "" {
		dto.SendError(ctx, http.StatusBadRequest, apperror.ErrInvalidID.Message, apperror.ErrInvalidID.Code)
		return
	}

	var req dto.UpdateTrackingRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		dto.SendError(ctx, http.StatusBadRequest, apperror.ErrBadRequest.Message, apperror.ErrBadRequest.Code)
		return
	}

	order, err := c.service.UpdateTracking(orderID, user.ID, &req)
	if err != nil {
		dto.SendError(ctx, apperror.StatusFromError(err), apperror.Message(err), apperror.Code(err))
		return
	}

	dto.SendSuccess(ctx, http.StatusOK, "Tracking updated successfully", order)
}

// RejectOrder rejects an order by the seller.
func (c *OrderController) RejectOrder(ctx *gin.Context) {
	authUser, exists := ctx.Get("authUser")
	if !exists {
		dto.SendError(ctx, http.StatusUnauthorized, apperror.ErrNotAuthenticated.Message, apperror.ErrNotAuthenticated.Code)
		return
	}

	user := authUser.(auth.AuthUser)
	if user.Role != string(model.SellerRole) {
		dto.SendError(ctx, http.StatusForbidden, apperror.ErrSellerAccessRequired.Message, apperror.ErrSellerAccessRequired.Code)
		return
	}

	orderID := ctx.Param("id")
	if orderID == "" {
		dto.SendError(ctx, http.StatusBadRequest, apperror.ErrInvalidID.Message, apperror.ErrInvalidID.Code)
		return
	}

	var req dto.RejectOrderRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		dto.SendError(ctx, http.StatusBadRequest, apperror.ErrBadRequest.Message, apperror.ErrBadRequest.Code)
		return
	}

	order, err := c.service.RejectOrder(orderID, user.ID, &req)
	if err != nil {
		dto.SendError(ctx, apperror.StatusFromError(err), apperror.Message(err), apperror.Code(err))
		return
	}

	dto.SendSuccess(ctx, http.StatusOK, "Order rejected successfully", order)
}

// ProcessRefund processes a refund for a returned order.
func (c *OrderController) ProcessRefund(ctx *gin.Context) {
	authUser, exists := ctx.Get("authUser")
	if !exists {
		dto.SendError(ctx, http.StatusUnauthorized, apperror.ErrNotAuthenticated.Message, apperror.ErrNotAuthenticated.Code)
		return
	}

	user := authUser.(auth.AuthUser)
	if user.Role != string(model.SellerRole) {
		dto.SendError(ctx, http.StatusForbidden, apperror.ErrSellerAccessRequired.Message, apperror.ErrSellerAccessRequired.Code)
		return
	}

	orderID := ctx.Param("id")
	if orderID == "" {
		dto.SendError(ctx, http.StatusBadRequest, apperror.ErrInvalidID.Message, apperror.ErrInvalidID.Code)
		return
	}

	order, err := c.service.ProcessRefund(orderID, user.ID)
	if err != nil {
		dto.SendError(ctx, apperror.StatusFromError(err), apperror.Message(err), apperror.Code(err))
		return
	}

	dto.SendSuccess(ctx, http.StatusOK, "Refund processed successfully", order)
}

// GetSellerOrderStats retrieves order statistics for the seller.
func (c *OrderController) GetSellerOrderStats(ctx *gin.Context) {
	authUser, exists := ctx.Get("authUser")
	if !exists {
		dto.SendError(ctx, http.StatusUnauthorized, apperror.ErrNotAuthenticated.Message, apperror.ErrNotAuthenticated.Code)
		return
	}

	user := authUser.(auth.AuthUser)
	if user.Role != string(model.SellerRole) {
		dto.SendError(ctx, http.StatusForbidden, apperror.ErrSellerAccessRequired.Message, apperror.ErrSellerAccessRequired.Code)
		return
	}

	stats, err := c.service.GetSellerOrderStats(user.ID)
	if err != nil {
		dto.SendError(ctx, apperror.StatusFromError(err), apperror.Message(err), apperror.Code(err))
		return
	}

	dto.SendSuccess(ctx, http.StatusOK, "Seller order stats retrieved successfully", stats)
}

// ==================== Admin Endpoints ====================

// GetAllOrders retrieves all orders (admin only).
func (c *OrderController) GetAllOrders(ctx *gin.Context) {
	var query dto.OrderFilterQuery
	if err := ctx.ShouldBindQuery(&query); err != nil {
		dto.SendError(ctx, http.StatusBadRequest, apperror.ErrBadRequest.Message, apperror.ErrBadRequest.Code)
		return
	}

	orders, err := c.service.GetAllOrders(&query)
	if err != nil {
		dto.SendError(ctx, apperror.StatusFromError(err), apperror.Message(err), apperror.Code(err))
		return
	}

	dto.SendSuccess(ctx, http.StatusOK, "All orders retrieved successfully", orders)
}

// AdminUpdateStatus updates an order status (admin only).
func (c *OrderController) AdminUpdateStatus(ctx *gin.Context) {
	orderID := ctx.Param("id")
	if orderID == "" {
		dto.SendError(ctx, http.StatusBadRequest, apperror.ErrInvalidID.Message, apperror.ErrInvalidID.Code)
		return
	}

	var req dto.UpdateOrderStatusRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		dto.SendError(ctx, http.StatusBadRequest, apperror.ErrBadRequest.Message, apperror.ErrBadRequest.Code)
		return
	}

	order, err := c.service.AdminUpdateStatus(orderID, &req)
	if err != nil {
		dto.SendError(ctx, apperror.StatusFromError(err), apperror.Message(err), apperror.Code(err))
		return
	}

	dto.SendSuccess(ctx, http.StatusOK, "Order status updated successfully", order)
}

// AdminMarkAsDelivered marks an order as delivered (admin only).
func (c *OrderController) AdminMarkAsDelivered(ctx *gin.Context) {
	orderID := ctx.Param("id")
	if orderID == "" {
		dto.SendError(ctx, http.StatusBadRequest, apperror.ErrInvalidID.Message, apperror.ErrInvalidID.Code)
		return
	}

	order, err := c.service.AdminMarkAsDelivered(orderID)
	if err != nil {
		dto.SendError(ctx, apperror.StatusFromError(err), apperror.Message(err), apperror.Code(err))
		return
	}

	dto.SendSuccess(ctx, http.StatusOK, "Order marked as delivered successfully", order)
}

// GetOrderStats retrieves overall order statistics (admin only).
func (c *OrderController) GetOrderStats(ctx *gin.Context) {
	stats, err := c.service.GetOrderStats()
	if err != nil {
		dto.SendError(ctx, apperror.StatusFromError(err), apperror.Message(err), apperror.Code(err))
		return
	}

	dto.SendSuccess(ctx, http.StatusOK, "Order stats retrieved successfully", stats)
}

// ==================== Payment Endpoints ====================

// MarkAsPaid marks an order as paid (for webhook/admin use).
func (c *OrderController) MarkAsPaid(ctx *gin.Context) {
	orderID := ctx.Param("id")
	if orderID == "" {
		dto.SendError(ctx, http.StatusBadRequest, apperror.ErrInvalidID.Message, apperror.ErrInvalidID.Code)
		return
	}

	err := c.service.MarkAsPaid(orderID)
	if err != nil {
		dto.SendError(ctx, apperror.StatusFromError(err), apperror.Message(err), apperror.Code(err))
		return
	}

	dto.SendSuccess(ctx, http.StatusOK, "Order marked as paid successfully", gin.H{"id": orderID})
}
