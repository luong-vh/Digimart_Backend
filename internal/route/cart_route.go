package route

import (
	"github.com/gin-gonic/gin"
	"github.com/luong-vh/Digimart_Backend/internal/controller"
	"github.com/luong-vh/Digimart_Backend/internal/middleware"
)

func RegisterCartRoutes(r *gin.RouterGroup, cartController *controller.CartController) {
	cart := r.Group("/cart")
	cart.Use(middleware.RequireAuth())
	{
		// Cart operations
		cart.GET("", cartController.GetCart)
		//cart.DELETE("", cartController.ClearCart)

		// Item operations
		cart.POST("/items", cartController.AddItem)
		cart.PUT("/items", cartController.UpdateItemQuantity)
		cart.DELETE("/items/:itemId", cartController.RemoveItem)

		// Batch operations
		cart.POST("/items/batch", cartController.AddItems)
		cart.DELETE("/items/batch", cartController.RemoveItems)

		// Validation
		cart.GET("/validate", cartController.ValidateCart)
	}

}
