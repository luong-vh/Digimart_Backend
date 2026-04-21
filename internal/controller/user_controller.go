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

// UserController handles requests related to user management.
type UserController struct {
	service service.UserService
}

// NewUserController creates a new UserController.
func NewUserController(service service.UserService) *UserController {
	return &UserController{service: service}
}

// GetMyProfile retrieves the profile of the currently authenticated user.
func (c *UserController) GetMyProfile(ctx *gin.Context) {
	authUser, exists := ctx.Get("authUser")
	if !exists {
		dto.SendError(ctx, http.StatusUnauthorized, apperror.ErrForbidden.Message, apperror.ErrForbidden.Code)
		return
	}

	user, err := c.service.GetUserByID(authUser.(auth.AuthUser).ID)
	if err != nil {
		dto.SendError(ctx, apperror.StatusFromError(err), apperror.Message(err), apperror.Code(err))
		return
	}

	dto.SendSuccess(ctx, http.StatusOK, "Profile retrieved successfully", user)
}

// UpdateProfile allows a user to update their own profile information.
func (c *UserController) UpdateProfile(ctx *gin.Context) {
	authUser, exists := ctx.Get("authUser")
	if !exists {
		dto.SendError(ctx, http.StatusUnauthorized, apperror.ErrForbidden.Message, apperror.ErrForbidden.Code)
		return
	}

	user, err := c.service.GetUserByID(authUser.(auth.AuthUser).ID)
	if err != nil {
		dto.SendError(ctx, apperror.StatusFromError(err), apperror.Message(err), apperror.Code(err))
		return
	}

	if user.Role == model.SellerRole {
		var req dto.SellerProfileUpdateRequest
		if err := ctx.ShouldBind(&req); err != nil {
			dto.SendError(ctx, http.StatusBadRequest, "Invalid request payload", apperror.ErrBadRequest.Code)
			return
		}
		updatedUser, err := c.service.UpdateSellerProfile(user.ID, &req)
		if err != nil {
			dto.SendError(ctx, apperror.StatusFromError(err), apperror.Message(err), apperror.Code(err))
			return
		}

		dto.SendSuccess(ctx, http.StatusOK, "Profile updated successfully", updatedUser)
	} else {
		var req dto.BuyerProfileUpdateRequest
		if err := ctx.ShouldBind(&req); err != nil {
			dto.SendError(ctx, http.StatusBadRequest, "Invalid request payload", apperror.ErrBadRequest.Code)
			return
		}
		updatedUser, err := c.service.UpdateBuyerProfile(user.ID, &req)
		if err != nil {
			dto.SendError(ctx, apperror.StatusFromError(err), apperror.Message(err), apperror.Code(err))
			return
		}

		dto.SendSuccess(ctx, http.StatusOK, "Profile updated successfully", updatedUser)
	}

}

func (c *UserController) ChangePassword(ctx *gin.Context) {
	authUser, exists := ctx.Get("authUser")
	if !exists {
		dto.SendError(ctx, http.StatusUnauthorized, apperror.ErrForbidden.Message, apperror.ErrForbidden.Code)
		return
	}

	var req dto.ChangePasswordRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		dto.SendError(ctx, http.StatusBadRequest, apperror.Message(apperror.ErrBadRequest), apperror.ErrBadRequest.Code)
		return
	}

	err := c.service.ChangePassword(authUser.(auth.AuthUser).ID, req.OldPassword, req.NewPassword)
	if err != nil {
		dto.SendError(ctx, apperror.StatusFromError(err), apperror.Message(err), apperror.Code(err))
		return
	}

	dto.SendSuccess(ctx, http.StatusOK, "Password changed successfully", nil)
}

// --- Admin-only actions ---

func (c *UserController) DeleteUser(ctx *gin.Context) {
	userID := ctx.Param("id")
	err := c.service.DeleteUser(userID)
	if err != nil {
		dto.SendError(ctx, apperror.StatusFromError(err), apperror.Message(err), apperror.Code(err))
		return
	}

	dto.SendSuccess(ctx, http.StatusOK, "User deleted successfully", gin.H{"id": userID})
}
