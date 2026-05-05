package route

import (
	"github.com/gin-gonic/gin"
	"github.com/luong-vh/Digimart_Backend/internal/controller"
	"github.com/luong-vh/Digimart_Backend/internal/middleware"
)

func RegisterProductRoutes(r *gin.RouterGroup, productController *controller.ProductController) {
	products := r.Group("/products")
	{
		// Public endpoints
		products.GET("", productController.GetProducts)
		products.GET("/:id", productController.GetProductByID)

		// Seller endpoints
		seller := products.Group("/seller")
		seller.Use(middleware.RequireAuth(), middleware.RequireAdmin())
		{
			seller.POST("", productController.CreateProduct)
			seller.PUT("/:id", productController.UpdateProduct)
			seller.DELETE("/:id", productController.DeleteProduct)

		}

	}
}
