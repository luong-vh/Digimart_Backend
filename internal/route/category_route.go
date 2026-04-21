package route

import (
	"github.com/gin-gonic/gin"
	"github.com/luong-vh/Digimart_Backend/internal/controller"
	"github.com/luong-vh/Digimart_Backend/internal/middleware"
)

func RegisterCategoryRoutes(r *gin.RouterGroup, categoryController *controller.CategoryController) {
	categories := r.Group("/categories")
	{
		// Public endpoints - Ai cũng xem được
		categories.GET("", categoryController.GetAllCategories)
		categories.GET("/paginate", categoryController.GetCategoriesWithPagination)
		categories.GET("/tree", categoryController.GetCategoryTree)
		categories.GET("/root", categoryController.GetRootCategories)

		// Đổi từ /:parentId/children thành /children/:id
		categories.GET("/children/:id", categoryController.GetChildCategories)

		// Get by ID phải đặt sau các route cố định
		categories.GET("/:id", categoryController.GetCategoryByID)

		// Seller endpoints - Seller có thể thêm category
		seller := categories.Group("/seller")
		seller.Use(middleware.RequireAuth(), middleware.RequireSeller())
		{
			seller.POST("", categoryController.CreateCategory)
		}

		// Admin endpoints - Admin full quyền
		admin := categories.Group("/admin")
		admin.Use(middleware.RequireAuth(), middleware.RequireAdmin())
		{
			admin.POST("", categoryController.CreateCategory)
			admin.PUT("/:id", categoryController.UpdateCategory)
			admin.DELETE("/:id", categoryController.DeleteCategory)
		}
	}
}
