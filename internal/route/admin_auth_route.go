package route

import (
	"github.com/gin-gonic/gin"
	"github.com/luong-vh/Digimart_Backend/internal/controller"
	"github.com/luong-vh/Digimart_Backend/internal/middleware"
)

func RegisterAdminAuthRoutes(rg *gin.RouterGroup, c *controller.AdminAuthController) {
	auth := rg.Group("/admin/auth")
	{
		// Public routes
		auth.POST("/login", c.AdminLogin)
		auth.POST("/refresh", c.AdminRefreshToken)

		// Protected routes (require admin authentication)
		auth.POST("/logout", middleware.RequireAuth(), middleware.RequireAdmin(), c.AdminLogout)
	}
}
