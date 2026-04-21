package route

import (
	"github.com/gin-gonic/gin"
	"github.com/luong-vh/Digimart_Backend/internal/controller"
)

func RegisterMediaRoutes(rg *gin.RouterGroup, c *controller.MediaController) {
	media := rg.Group("/media")
	media.Use()
	{
		// Upload
		media.POST("/image", c.UploadImage)
		media.POST("/images", c.UploadImages)
		media.POST("/video", c.UploadVideo)
		media.POST("/videos", c.UploadVideos)

	}
}
