package dto

import (
	"time"

	"github.com/luong-vh/Digimart_Backend/internal/model"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// --- Request DTOs ---

// New Registration Flow (Verify Email First)

// GetUsersQuery contains query parameters for searching and paginating users
type GetUsersQuery struct {
	Page     int `form:"page"`
	PageSize int `form:"pageSize"`
}
type GetBuyersQuery struct {
	// Pagination
	Page     int `form:"page"`
	PageSize int `form:"pageSize"`

	// Search
	Keyword     string `form:"keyword"` // Tìm theo fullname hoặc email
	Email       string `form:"email"`
	FullName    string `form:"full_name"`
	PhoneNumber string `form:"phone_number"`

	// Filter
	Gender string `form:"gender"` // male, female, other

}
type BuyerProfileUpdateRequest struct {
	FullName         *string             `json:"full_name"`
	AvatarURL        *string             `json:"avatar_url"`
	PublicID         *string             `json:"public_id"`
	PhoneNumber      *string             `json:"phone_number,omitempty"`
	Gender           *model.Gender       `json:"gender,omitempty"`
	DateOfBirth      *time.Time          `json:"date_of_birth,omitempty"`
	Address          []model.Address     `json:"address,omitempty"`
	DefaultAddressID *primitive.ObjectID `json:"default_address_id,omitempty"`
}

type ChangePasswordRequest struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=6"`
}

// --- Response DTOs ---

// UserResponse is the main user object returned in API responses.
type UserResponse struct {
	ID          string            `json:"id"`
	FullName    string            `json:"full_name"`
	Email       string            `json:"email,omitempty"`
	Role        model.Role        `json:"role"`
	IsVerified  bool              `json:"is_verified"`
	RoleContent model.RoleContent `json:"role_content"`
}

func FromUser(u *model.User) *UserResponse {
	if u == nil {
		return nil
	}
	resp := &UserResponse{
		ID:          u.ID.Hex(),
		FullName:    u.FullName,
		Email:       u.Email,
		Role:        u.Role,
		IsVerified:  u.IsVerified,
		RoleContent: u.RoleContent,
	}

	return resp
}

func FromUsers(users []*model.User) []*UserResponse {
	responses := make([]*UserResponse, len(users))
	for i, u := range users {
		userResponse := FromUser(u)
		//	userResponse.Email = ""
		responses[i] = userResponse
	}
	return responses
}
