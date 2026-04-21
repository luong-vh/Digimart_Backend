package route

import (
	"github.com/DucLove1/SE357-ShoppingManagement-BE/internal/controller"
	"github.com/DucLove1/SE357-ShoppingManagement-BE/internal/middleware"
	"github.com/gin-gonic/gin"
)

func RegisterProvinceRoutes(rg *gin.RouterGroup, c *controller.ProvinceController) {
	provinces := rg.Group("/provinces")
	{
		// Public endpoints
		provinces.GET("", c.GetAllProvinces)
		provinces.GET("/:id", c.GetProvinceByID)
		provinces.GET("/:id/wards", c.GetWardsByProvinceID)
		provinces.GET("/:id/wards/:wardId", c.GetWardByID)

		// Admin endpoints
		admin := provinces.Group("/admin")
		admin.Use(middleware.RequireAuth(), middleware.RequireAdmin())
		{
			admin.POST("", c.CreateProvince)
			admin.PUT("/:id", c.UpdateProvince)
			admin.DELETE("/:id", c.DeleteProvince)

			// Ward management
			admin.POST("/:id/wards", c.AddWard)
			admin.PUT("/:id/wards/:wardId", c.UpdateWard)
			admin.DELETE("/:id/wards/:wardId", c.DeleteWard)
		}
	}
}
