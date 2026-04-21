package route

import (
	"github.com/DucLove1/SE357-ShoppingManagement-BE/internal/controller"
	"github.com/DucLove1/SE357-ShoppingManagement-BE/internal/middleware"
	"github.com/gin-gonic/gin"
)

func RegisterCartRoutes(r *gin.RouterGroup, cartController *controller.CartController) {
	cart := r.Group("/cart")
	cart.Use(middleware.RequireAuth())
	{
		// Cart operations
		cart.GET("", cartController.GetCart)
		cart.GET("/refresh", cartController.GetCartWithRefresh)
		cart.DELETE("", cartController.ClearCart)

		// Item operations
		cart.POST("/items", cartController.AddItem)
		cart.PUT("/items", cartController.UpdateItemQuantity)
		cart.DELETE("/items/:productId", cartController.RemoveItem)

		// Batch operations
		cart.POST("/items/batch", cartController.AddItems)
		cart.DELETE("/items/batch", cartController.RemoveItems)

		// Validation
		cart.GET("/validate", cartController.ValidateCart)
	}

	// Admin endpoints
	cartAdmin := r.Group("/cart/admin")
	cartAdmin.Use(middleware.RequireAuth(), middleware.RequireAdmin())
	{
		cartAdmin.GET("/stats", cartController.GetCartStats)
	}
}
