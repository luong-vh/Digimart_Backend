package route

import (
	"github.com/DucLove1/SE357-ShoppingManagement-BE/internal/controller"
	"github.com/DucLove1/SE357-ShoppingManagement-BE/internal/middleware"
	"github.com/gin-gonic/gin"
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

		// Seller endpoints
		seller := orders.Group("/seller")
		seller.Use(middleware.RequireAuth(), middleware.RequireSeller())
		{
			seller.GET("", orderController.GetSellerOrders)
			seller.GET("/stats", orderController.GetSellerOrderStats)
			seller.POST("/:id/confirm", orderController.ConfirmOrder)
			seller.POST("/:id/pack", orderController.PackOrder)
			seller.POST("/:id/ship", orderController.ShipOrder)
			seller.PUT("/:id/tracking", orderController.UpdateTracking)
			seller.POST("/:id/reject", orderController.RejectOrder)
			seller.POST("/:id/refund", orderController.ProcessRefund)
		}

		// Admin endpoints
		admin := orders.Group("/admin")
		admin.Use(middleware.RequireAuth(), middleware.RequireAdmin())
		{
			admin.GET("", orderController.GetAllOrders)
			admin.GET("/stats", orderController.GetOrderStats)
			admin.PUT("/:id/status", orderController.AdminUpdateStatus)
			admin.POST("/:id/deliver", orderController.AdminMarkAsDelivered)
			admin.POST("/:id/mark-paid", orderController.MarkAsPaid)
		}
	}
}
