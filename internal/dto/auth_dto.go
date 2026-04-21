package dto

import "github.com/DucLove1/SE357-ShoppingManagement-BE/internal/model"

type SendEmailVerificationRequest struct {
	Email string `json:"email" binding:"required,email"`
}

type VerifyEmailCodeRequest struct {
	Email string `json:"email" binding:"required,email"`
	OTP   string `json:"otp" binding:"required,len=6"`
}

type CompleteBuyerRegistrationRequest struct {
	VerificationToken string `json:"verification_token" binding:"required"`
	Name              string `json:"name" binding:"required"`
	Password          string `json:"password" binding:"required,min=6"`
}

type CompleteSellerRegistrationRequest struct {
	VerificationToken string `json:"verification_token" binding:"required"`
	Name              string `json:"name" binding:"required"`
	Password          string `json:"password" binding:"required,min=6"`

	PhoneNumber   string           `json:"phone_number" binding:"required,len=10"`
	Categories    []model.Category `json:"categories" `
	PickupAddress model.Address    `json:"pickup_address" binding:"required"`

	IdentityCard      string      `bson:"identity_card,omitempty" json:"identity_card,required"`
	IDFrontImage      model.Image `json:"id_front_image" binding:"required"`
	IDBackImage       model.Image `json:"id_back_image" binding:"required"`
	SelfieWithIDImage model.Image `json:"selfie_with_id_image" binding:"required"`
}
type ResendOTPRequest struct {
	Email string `json:"email" binding:"required,email"`
}

// Login
type UserLoginRequest struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type CompleteGoogleSetupRequest struct {
	SetupToken string `json:"setup_token" binding:"required"`
	Username   string `json:"username" binding:"required,min=3,max=20"`
}
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type LogoutRequest struct {
	AccessToken  string `json:"access_token" binding:"required"`
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// AuthResponse is returned on successful login or registration.
type AuthResponse struct {
	User         *UserResponse `json:"user"`
	AccessToken  string        `json:"access_token"`
	RefreshToken string        `json:"refresh_token"`
}

type RefreshResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

// Forgot Password Flow
type ForgotPasswordRequest struct {
	Email string `json:"email" binding:"required,email"`
}

type VerifyResetPasswordOTPRequest struct {
	Email string `json:"email" binding:"required,email"`
	OTP   string `json:"otp" binding:"required,len=6"`
}

type VerifyResetPasswordOTPResponse struct {
	ResetToken string `json:"reset_token"`
}

type ResetPasswordRequest struct {
	ResetToken  string `json:"reset_token" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=6"`
}
