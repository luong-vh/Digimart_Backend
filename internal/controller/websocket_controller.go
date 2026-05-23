package controller

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/luong-vh/Digimart_Backend/internal/auth"
	"github.com/luong-vh/Digimart_Backend/internal/platform/ws"
)

type WebSocketController struct {
	hub *ws.Hub
}

func NewWebSocketController(hub *ws.Hub) *WebSocketController {
	return &WebSocketController{hub: hub}
}

var wsUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func (c *WebSocketController) Connect(ctx *gin.Context) {
	token := ctx.Query("token")
	if token == "" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"success": false, "message": "Missing token"})
		return
	}

	user, err := auth.ParseAccessToken(token)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"success": false, "message": "Invalid token"})
		return
	}

	conn, err := wsUpgrader.Upgrade(ctx.Writer, ctx.Request, nil)
	if err != nil {
		return
	}

	client := ws.NewClient(c.hub, conn, user.ID)
	c.hub.RegisterClient(client)
	client.Serve()
}
