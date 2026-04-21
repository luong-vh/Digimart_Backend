package route

import (
	"github.com/gin-gonic/gin"
	"github.com/luong-vh/Digimart_Backend/internal/controller"
)

// RegisterAuthRoutes registers all authentication-related routes.
func RegisterAuthRoutes(rg *gin.RouterGroup, authCtrl *controller.AuthController) {
	auth := rg.Group("/auth")

	auth.POST("/refresh", authCtrl.RefreshToken)
	auth.POST("/logout", authCtrl.Logout)

	// Local Authentication - New Flow (Verify Email First)
	local := auth.Group("/local")
	{
		local.POST("/send-verification", authCtrl.SendEmailVerification)
		local.POST("/verify-email", authCtrl.VerifyEmailCode)
		local.POST("/complete-buyer-registration", authCtrl.CompleteBuyerRegistration)
		local.POST("/complete-seller-registration", authCtrl.CompleteSellerRegistration)
		local.POST("/resend-otp", authCtrl.ResendOTP)
		local.POST("/login", authCtrl.Login)

		// Forgot Password Flow (only for local auth)
		local.POST("/forgot-password", authCtrl.ForgotPassword)
		local.POST("/verify-reset-otp", authCtrl.VerifyResetPasswordOTP)
		local.POST("/reset-password", authCtrl.ResetPassword)
	}

	// Google OAuth2
	google := auth.Group("/google")
	{
		google.GET("/login", authCtrl.GoogleLogin)
		google.GET("/callback", authCtrl.GoogleCallback)
		google.POST("/complete-setup", authCtrl.CompleteGoogleSetup)
	}
}
