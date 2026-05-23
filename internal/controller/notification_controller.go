package controller

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/luong-vh/Digimart_Backend/internal/apperror"
	"github.com/luong-vh/Digimart_Backend/internal/auth"
	"github.com/luong-vh/Digimart_Backend/internal/dto"
	"github.com/luong-vh/Digimart_Backend/internal/service"
)

type NotificationController struct {
	service service.NotificationService
}

func NewNotificationController(service service.NotificationService) *NotificationController {
	return &NotificationController{service: service}
}

func (c *NotificationController) List(ctx *gin.Context) {
	user, ok := currentAuthUser(ctx)
	if !ok {
		dto.SendError(ctx, http.StatusUnauthorized, apperror.ErrForbidden.Message, apperror.ErrForbidden.Code)
		return
	}

	page := parseInt64Query(ctx, "page", 1)
	pageSize := parseInt64Query(ctx, "page_size", 20)

	notifications, total, err := c.service.List(user.ID, page, pageSize)
	if err != nil {
		dto.SendError(ctx, apperror.StatusFromError(err), apperror.Message(err), apperror.Code(err))
		return
	}

	dto.SendSuccessWithPagination(ctx, http.StatusOK, "Notifications retrieved successfully", notifications, (page-1)*pageSize, pageSize, total)
}

func (c *NotificationController) UnreadCount(ctx *gin.Context) {
	user, ok := currentAuthUser(ctx)
	if !ok {
		dto.SendError(ctx, http.StatusUnauthorized, apperror.ErrForbidden.Message, apperror.ErrForbidden.Code)
		return
	}

	count, err := c.service.CountUnread(user.ID)
	if err != nil {
		dto.SendError(ctx, apperror.StatusFromError(err), apperror.Message(err), apperror.Code(err))
		return
	}

	dto.SendSuccess(ctx, http.StatusOK, "Unread notification count retrieved successfully", gin.H{"count": count})
}

func (c *NotificationController) MarkAsRead(ctx *gin.Context) {
	user, ok := currentAuthUser(ctx)
	if !ok {
		dto.SendError(ctx, http.StatusUnauthorized, apperror.ErrForbidden.Message, apperror.ErrForbidden.Code)
		return
	}

	if err := c.service.MarkAsRead(user.ID, ctx.Param("id")); err != nil {
		dto.SendError(ctx, apperror.StatusFromError(err), apperror.Message(err), apperror.Code(err))
		return
	}

	dto.SendSuccess(ctx, http.StatusOK, "Notification marked as read", gin.H{"id": ctx.Param("id")})
}

func (c *NotificationController) MarkAllAsRead(ctx *gin.Context) {
	user, ok := currentAuthUser(ctx)
	if !ok {
		dto.SendError(ctx, http.StatusUnauthorized, apperror.ErrForbidden.Message, apperror.ErrForbidden.Code)
		return
	}

	if err := c.service.MarkAllAsRead(user.ID); err != nil {
		dto.SendError(ctx, apperror.StatusFromError(err), apperror.Message(err), apperror.Code(err))
		return
	}

	dto.SendSuccess(ctx, http.StatusOK, "All notifications marked as read", gin.H{"ok": true})
}

func currentAuthUser(ctx *gin.Context) (auth.AuthUser, bool) {
	value, exists := ctx.Get("authUser")
	if !exists {
		return auth.AuthUser{}, false
	}
	user, ok := value.(auth.AuthUser)
	return user, ok
}

func parseInt64Query(ctx *gin.Context, key string, fallback int64) int64 {
	raw := ctx.Query(key)
	if raw == "" {
		return fallback
	}
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || value < 1 {
		return fallback
	}
	return value
}
