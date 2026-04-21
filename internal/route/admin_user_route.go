package route

import (
	"github.com/gin-gonic/gin"
	"github.com/luong-vh/Digimart_Backend/internal/controller"
	"github.com/luong-vh/Digimart_Backend/internal/middleware"
)

func RegisterAdminUserRoutes(rg *gin.RouterGroup, c *controller.AdminUserController) {
	admin := rg.Group("/admin/users")

	// All admin routes require authentication AND admin role
	admin.Use(middleware.RequireAuth(), middleware.RequireAdmin())
	{
		admin.GET("buyers", c.GetBuyers)
		admin.GET("buyers/", c.GetBuyers)
		admin.GET("sellers", c.GetSellers)
		admin.GET("sellers/", c.GetSellers)
		// User management
		admin.POST("/:user_id/ban", c.BanUser)
		admin.POST("/:user_id/unban", c.UnbanUser)
		admin.DELETE("/:user_id", c.DeleteUser)
		admin.POST("/:user_id/restore", c.RestoreUser)

		admin.POST("/:user_id/approve", c.ApproveSeller)
		admin.POST("/:user_id/reject", c.RejectSeller)
	}
}
