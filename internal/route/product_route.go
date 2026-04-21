package route

import (
	"github.com/DucLove1/SE357-ShoppingManagement-BE/internal/controller"
	"github.com/DucLove1/SE357-ShoppingManagement-BE/internal/middleware"
	"github.com/gin-gonic/gin"
)

func RegisterProductRoutes(r *gin.RouterGroup, productController *controller.ProductController) {
	products := r.Group("/products")
	{
		// Public endpoints
		products.GET("", productController.GetProducts)
		products.GET("/:id", productController.GetProductByID)
		products.GET("/slug/:slug", productController.GetProductBySlug)
		products.GET("/category/:categoryId", productController.GetProductsByCategory)
		products.GET("/seller/:sellerId", productController.GetProductsBySeller)

		// Seller endpoints
		seller := products.Group("/seller")
		seller.Use(middleware.RequireAuth(), middleware.RequireSeller())
		{
			seller.POST("", productController.CreateProduct)
			seller.GET("/my-products", productController.GetMyProducts)
			seller.PUT("/:id", productController.UpdateProduct)
			seller.DELETE("/:id", productController.DeleteProduct)
			seller.PATCH("/:id/status", productController.UpdateProductStatus)
			seller.PATCH("/:id/stock", productController.UpdateStock)

			// Variant endpoints
			seller.POST("/:id/variants", productController.AddVariant)
			seller.PUT("/:id/variants/:variantId", productController.UpdateVariant)
			seller.DELETE("/:id/variants/:variantId", productController.DeleteVariant)
			seller.PATCH("/:id/variants/:variantId/stock", productController.UpdateVariantStock)
		}

		// Admin endpoints
		admin := products.Group("/admin")
		admin.Use(middleware.RequireAuth(), middleware.RequireAdmin())
		{
			admin.GET("/stats", productController.GetProductStats)
			admin.PATCH("/:id/status", productController.AdminUpdateProductStatus)
			admin.DELETE("/:id", productController.AdminDeleteProduct)
		}
	}
}
