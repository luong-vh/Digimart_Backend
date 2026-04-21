package controller

import (
	"net/http"

	"github.com/DucLove1/SE357-ShoppingManagement-BE/internal/apperror"
	"github.com/DucLove1/SE357-ShoppingManagement-BE/internal/dto"
	"github.com/DucLove1/SE357-ShoppingManagement-BE/internal/service"
	"github.com/gin-gonic/gin"
)

type AdminUserController struct {
	adminService service.AdminUserService
}

func NewAdminUserController(adminService service.AdminUserService) *AdminUserController {
	return &AdminUserController{
		adminService: adminService,
	}
}

// GetUsers retrieves a paginated list of users with optional username search.
func (c *AdminUserController) GetBuyers(ctx *gin.Context) {
	var query dto.GetBuyersQuery
	if err := ctx.ShouldBindQuery(&query); err != nil {
		dto.SendError(ctx, http.StatusBadRequest, "Invalid query parameters", apperror.ErrBadRequest.Code)
		return
	}

	response, err := c.adminService.GetBuyers(&query)
	if err != nil {
		dto.SendError(ctx, apperror.StatusFromError(err), apperror.Message(err), apperror.Code(err))
		return
	}
	dto.SendSuccess(ctx, http.StatusOK, "Users retrieved successfully", response)
}

func (c *AdminUserController) GetSellers(ctx *gin.Context) {
	var query dto.GetSellersQuery
	if err := ctx.ShouldBindQuery(&query); err != nil {
		dto.SendError(ctx, http.StatusBadRequest, "Invalid query parameters", apperror.ErrBadRequest.Code)
		return
	}

	response, err := c.adminService.GetSellers(&query)
	if err != nil {
		dto.SendError(ctx, apperror.StatusFromError(err), apperror.Message(err), apperror.Code(err))
		return
	}
	dto.SendSuccess(ctx, http.StatusOK, "Users retrieved successfully", response)
}

func (c *AdminUserController) ApproveSeller(ctx *gin.Context) {
	userID := ctx.Param("user_id")
	if userID == "" {
		dto.SendError(ctx, http.StatusBadRequest, "User ID is required", apperror.ErrBadRequest.Code)
		return
	}

	err := c.adminService.ApproveSeller(userID)
	if err != nil {
		dto.SendError(ctx, apperror.StatusFromError(err), apperror.Message(err), apperror.Code(err))
		return
	}

	dto.SendSuccess(ctx, http.StatusOK, "Seller approved", gin.H{"user_id": userID})
}

func (c *AdminUserController) RejectSeller(ctx *gin.Context) {
	userID := ctx.Param("user_id")
	if userID == "" {
		dto.SendError(ctx, http.StatusBadRequest, "User ID is required", apperror.ErrBadRequest.Code)
		return
	}
	err := c.adminService.RejectSeller(userID)
	if err != nil {
		dto.SendError(ctx, apperror.StatusFromError(err), apperror.Message(err), apperror.Code(err))
		return
	}

	dto.SendSuccess(ctx, http.StatusOK, "Seller rejected", gin.H{"user_id": userID})
}

// BanUser bans a user
func (c *AdminUserController) BanUser(ctx *gin.Context) {
	userID := ctx.Param("user_id")
	if userID == "" {
		dto.SendError(ctx, http.StatusBadRequest, "User ID is required", apperror.ErrBadRequest.Code)
		return
	}

	var req dto.BanUserRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		dto.SendError(ctx, http.StatusBadRequest, apperror.Message(apperror.ErrBadRequest), apperror.ErrBadRequest.Code)
		return
	}

	err := c.adminService.BanUser(userID, &req)
	if err != nil {
		dto.SendError(ctx, apperror.StatusFromError(err), apperror.Message(err), apperror.Code(err))
		return
	}

	banType := "permanently"
	if req.BanUntil != nil {
		banType = "until " + req.BanUntil.Format("2006-01-02")
	}

	dto.SendSuccess(ctx, http.StatusOK, "User banned "+banType, gin.H{"user_id": userID})
}

// UnbanUser unbans a user
func (c *AdminUserController) UnbanUser(ctx *gin.Context) {
	userID := ctx.Param("user_id")
	if userID == "" {
		dto.SendError(ctx, http.StatusBadRequest, "User ID is required", apperror.ErrBadRequest.Code)
		return
	}

	err := c.adminService.UnbanUser(userID)
	if err != nil {
		dto.SendError(ctx, apperror.StatusFromError(err), apperror.Message(err), apperror.Code(err))
		return
	}

	dto.SendSuccess(ctx, http.StatusOK, "User unbanned successfully", gin.H{"user_id": userID})
}

// DeleteUser soft deletes a user
func (c *AdminUserController) DeleteUser(ctx *gin.Context) {
	userID := ctx.Param("user_id")

	if userID == "" {
		dto.SendError(ctx, http.StatusBadRequest, "User ID is required", apperror.ErrBadRequest.Code)
		return
	}

	err := c.adminService.SoftDeleteUser(userID)
	if err != nil {
		dto.SendError(ctx, apperror.StatusFromError(err), apperror.Message(err), apperror.Code(err))
		return
	}

	dto.SendSuccess(ctx, http.StatusOK, "User deleted successfully", gin.H{"user_id": userID})
}

// RestoreUser restores a soft-deleted user
func (c *AdminUserController) RestoreUser(ctx *gin.Context) {
	userID := ctx.Param("user_id")
	if userID == "" {
		dto.SendError(ctx, http.StatusBadRequest, "User ID is required", apperror.ErrBadRequest.Code)
		return
	}

	err := c.adminService.RestoreUser(userID)
	if err != nil {
		dto.SendError(ctx, apperror.StatusFromError(err), apperror.Message(err), apperror.Code(err))
		return
	}

	dto.SendSuccess(ctx, http.StatusOK, "User restored successfully", gin.H{"user_id": userID})
}
