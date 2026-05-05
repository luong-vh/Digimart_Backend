package controller

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/luong-vh/Digimart_Backend/internal/apperror"
	"github.com/luong-vh/Digimart_Backend/internal/auth"
	"github.com/luong-vh/Digimart_Backend/internal/dto"
	"github.com/luong-vh/Digimart_Backend/internal/service"
)

type CartController struct {
	service service.CartService
}

func NewCartController(service service.CartService) *CartController {
	return &CartController{service: service}
}

// ==================== CART OPERATIONS ====================

// GetCart retrieves the cart for the authenticated user
func (c *CartController) GetCart(ctx *gin.Context) {
	authUser, exists := ctx.Get("authUser")
	if !exists {
		dto.SendError(ctx, http.StatusUnauthorized, apperror.ErrForbidden.Message, apperror.ErrForbidden.Code)
		return
	}

	userID := authUser.(auth.AuthUser).ID

	cart, err := c.service.GetCart(ctx, userID)
	if err != nil {
		dto.SendError(ctx, apperror.StatusFromError(err), apperror.Message(err), apperror.Code(err))
		return
	}

	dto.SendSuccess(ctx, http.StatusOK, "Cart retrieved successfully", cart)
}

// ==================== ITEM OPERATIONS ====================

// AddItem adds an item to the cart
func (c *CartController) AddItem(ctx *gin.Context) {
	authUser, exists := ctx.Get("authUser")
	if !exists {
		dto.SendError(ctx, http.StatusUnauthorized, apperror.ErrForbidden.Message, apperror.ErrForbidden.Code)
		return
	}

	userID := authUser.(auth.AuthUser).ID

	var req dto.AddCartItemRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		dto.SendError(ctx, http.StatusBadRequest, "Invalid request payload", apperror.ErrBadRequest.Code)
		return
	}

	cart, err := c.service.AddItem(ctx, userID, req)
	if err != nil {
		dto.SendError(ctx, apperror.StatusFromError(err), apperror.Message(err), apperror.Code(err))
		return
	}

	dto.SendSuccess(ctx, http.StatusOK, "Item added to cart successfully", cart)
}

// UpdateItemQuantity updates the quantity of an item in the cart
func (c *CartController) UpdateItemQuantity(ctx *gin.Context) {
	authUser, exists := ctx.Get("authUser")
	if !exists {
		dto.SendError(ctx, http.StatusUnauthorized, apperror.ErrForbidden.Message, apperror.ErrForbidden.Code)
		return
	}

	userID := authUser.(auth.AuthUser).ID

	var req dto.UpdateCartItemRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		dto.SendError(ctx, http.StatusBadRequest, "Invalid request payload", apperror.ErrBadRequest.Code)
		return
	}

	cart, err := c.service.UpdateItemQuantity(ctx, userID, req)
	if err != nil {
		dto.SendError(ctx, apperror.StatusFromError(err), apperror.Message(err), apperror.Code(err))
		return
	}

	dto.SendSuccess(ctx, http.StatusOK, "Item quantity updated successfully", cart)
}

// RemoveItem removes an item from the cart
func (c *CartController) RemoveItem(ctx *gin.Context) {
	authUser, exists := ctx.Get("authUser")
	if !exists {
		dto.SendError(ctx, http.StatusUnauthorized, apperror.ErrForbidden.Message, apperror.ErrForbidden.Code)
		return
	}

	userID := authUser.(auth.AuthUser).ID
	itemID := ctx.Param("itemId")

	cart, err := c.service.RemoveItem(ctx, userID, itemID)
	if err != nil {
		dto.SendError(ctx, apperror.StatusFromError(err), apperror.Message(err), apperror.Code(err))
		return
	}

	dto.SendSuccess(ctx, http.StatusOK, "Item removed from cart successfully", cart)
}

// ==================== BATCH OPERATIONS ====================

// AddItems adds multiple items to the cart
func (c *CartController) AddItems(ctx *gin.Context) {
	authUser, exists := ctx.Get("authUser")
	if !exists {
		dto.SendError(ctx, http.StatusUnauthorized, apperror.ErrForbidden.Message, apperror.ErrForbidden.Code)
		return
	}

	userID := authUser.(auth.AuthUser).ID

	var req dto.AddCartItemsRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		dto.SendError(ctx, http.StatusBadRequest, "Invalid request payload", apperror.ErrBadRequest.Code)
		return
	}

	cart, err := c.service.AddItems(ctx, userID, req)
	if err != nil {
		dto.SendError(ctx, apperror.StatusFromError(err), apperror.Message(err), apperror.Code(err))
		return
	}

	dto.SendSuccess(ctx, http.StatusOK, "Items added to cart successfully", cart)
}

// RemoveItems removes multiple items from the cart
func (c *CartController) RemoveItems(ctx *gin.Context) {
	authUser, exists := ctx.Get("authUser")
	if !exists {
		dto.SendError(ctx, http.StatusUnauthorized, apperror.ErrForbidden.Message, apperror.ErrForbidden.Code)
		return
	}

	userID := authUser.(auth.AuthUser).ID

	var req dto.RemoveCartItemsRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		dto.SendError(ctx, http.StatusBadRequest, "Invalid request payload", apperror.ErrBadRequest.Code)
		return
	}

	cart, err := c.service.RemoveItems(ctx, userID, req)
	if err != nil {
		dto.SendError(ctx, apperror.StatusFromError(err), apperror.Message(err), apperror.Code(err))
		return
	}

	dto.SendSuccess(ctx, http.StatusOK, "Items removed from cart successfully", cart)
}

// ==================== VALIDATION ====================

// ValidateCart validates the cart and returns validation results
func (c *CartController) ValidateCart(ctx *gin.Context) {
	authUser, exists := ctx.Get("authUser")
	if !exists {
		dto.SendError(ctx, http.StatusUnauthorized, apperror.ErrForbidden.Message, apperror.ErrForbidden.Code)
		return
	}

	userID := authUser.(auth.AuthUser).ID

	validation, err := c.service.ValidateCart(ctx, userID)
	if err != nil {
		dto.SendError(ctx, apperror.StatusFromError(err), apperror.Message(err), apperror.Code(err))
		return
	}

	dto.SendSuccess(ctx, http.StatusOK, "Cart validated successfully", validation)
}
