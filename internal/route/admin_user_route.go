package route

import (
	"github.com/DucLove1/SE357-ShoppingManagement-BE/internal/controller"
	"github.com/DucLove1/SE357-ShoppingManagement-BE/internal/middleware"
	"github.com/gin-gonic/gin"
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
