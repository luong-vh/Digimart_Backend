package route

import (
	"github.com/gin-gonic/gin"
	"github.com/luong-vh/Digimart_Backend/internal/controller"
	"github.com/luong-vh/Digimart_Backend/internal/middleware"
)

func RegisterNotificationRoutes(r *gin.RouterGroup, notificationController *controller.NotificationController) {
	notifications := r.Group("/notifications")
	notifications.Use(middleware.RequireAuth())
	{
		notifications.GET("", notificationController.List)
		notifications.GET("/unread-count", notificationController.UnreadCount)
		notifications.PUT("/read-all", notificationController.MarkAllAsRead)
		notifications.PUT("/:id/read", notificationController.MarkAsRead)
	}
}
