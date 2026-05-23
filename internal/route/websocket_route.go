package route

import (
	"github.com/gin-gonic/gin"
	"github.com/luong-vh/Digimart_Backend/internal/controller"
)

func RegisterWebSocketRoutes(r *gin.RouterGroup, webSocketController *controller.WebSocketController) {
	r.GET("/ws", webSocketController.Connect)
}
