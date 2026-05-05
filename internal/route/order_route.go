package route

import (
	"github.com/gin-gonic/gin"
	"github.com/luong-vh/Digimart_Backend/internal/controller"
	"github.com/luong-vh/Digimart_Backend/internal/middleware"
)

func RegisterOrderRoutes(r *gin.RouterGroup, orderController *controller.OrderController) {
	orders := r.Group("/orders")
	{
		// Customer endpoints (authenticated)
		customer := orders.Group("")
		customer.Use(middleware.RequireAuth())
		{
			customer.POST("", orderController.PlaceOrder)
			customer.GET("/my-orders", orderController.GetMyOrders)
			customer.GET("/:id", orderController.GetOrderByID)
			customer.GET("/number/:orderNumber", orderController.GetOrderByNumber)
			customer.POST("/:id/cancel", orderController.CancelOrder)
			customer.POST("/:id/return", orderController.RequestReturn)
		}

		// Admin endpoints
		admin := orders.Group("/admin")
		admin.Use(middleware.RequireAuth(), middleware.RequireAdmin())
		{
			admin.GET("/stats", orderController.GetSellerOrderStats)
			admin.POST("/:id/confirm", orderController.ConfirmOrder)
			admin.POST("/:id/pack", orderController.PackOrder)
			admin.POST("/:id/ship", orderController.ShipOrder)
			admin.PUT("/:id/tracking", orderController.UpdateTracking)
			admin.POST("/:id/reject", orderController.RejectOrder)
			admin.POST("/:id/refund", orderController.ProcessRefund)
			admin.GET("", orderController.GetAllOrders)
			
			admin.PUT("/:id/status", orderController.AdminUpdateStatus)
			admin.POST("/:id/deliver", orderController.AdminMarkAsDelivered)
			admin.POST("/:id/mark-paid", orderController.MarkAsPaid)
		}
	}
}
