package dto

import (
	"fmt"
	"time"

	"github.com/DucLove1/SE357-ShoppingManagement-BE/internal/model"
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

type GetSellersQuery struct {
	// Pagination
	Page     int `form:"page"`
	PageSize int `form:"pageSize"`

	// Search
	Keyword     string `form:"keyword"` // Tìm theo fullname, email hoặc shop name
	Email       string `form:"email"`
	FullName    string `form:"full_name"`
	PhoneNumber string `form:"phone_number"`

	// Filter
	SellerStatus string              `form:"seller_status"` // pending, active, rejected
	CategoryID   *primitive.ObjectID `form:"category_id"`   // Filter theo danh mục bán

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
type SellerProfileUpdateRequest struct {
	AvatarURL      *string `json:"avatar_url"`
	AvatarPublicID *string `json:"avatar_public_id"`

	BannerURL      *string `json:"banner_url"`
	BannerPublicID *string `json:"banner_public_id"`

	FullName      *string          `json:"full_name"`
	Categories    []model.Category `json:"categories"`
	PickupAddress *model.Address   `json:"pickup_address"`
	PhoneNumber   *string          `json:"phone_number"`
	IdentityCard  *string          `json:"identity_card"`

	IdentityCardURL      *string `json:"identity_card_url"`
	IdentityCardPublicID *string `json:"identity_card_public_id"`

	IDFrontImageURL      *string `json:"id_front_image_url"`
	IDFrontImagePublicID *string `json:"id_front_image_public_id"`

	IDBackImageURL      *string `json:"id_back_image_url"`
	IDBackImagePublicID *string `json:"id_back_image_public_id"`

	SelfieWithIDURL      *string `json:"selfie_with_id_url"`
	SelfieWithIDPublicID *string `json:"selfie_with_id_public_id"`
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

func calculateAge(birthDate time.Time) int {
	now := time.Now()
	age := now.Year() - birthDate.Year()
	if now.YearDay() < birthDate.YearDay() {
		age--
	}
	return age
}

func formatMemberSince(joinedAt time.Time) string {
	monthNames := map[time.Month]string{
		time.January: "Jan", time.February: "Feb", time.March: "Mar",
		time.April: "Apr", time.May: "May", time.June: "Jun",
		time.July: "Jul", time.August: "Aug", time.September: "Sep",
		time.October: "Oct", time.November: "Nov", time.December: "Dec",
	}
	return fmt.Sprintf("Member since %s %d", monthNames[joinedAt.Month()], joinedAt.Year())
}

func formatLastActive(lastActive time.Time) string {
	now := time.Now()
	duration := now.Sub(lastActive)

	switch {
	case duration < time.Minute:
		return "Active now"
	case duration < time.Hour:
		minutes := int(duration.Minutes())
		if minutes == 1 {
			return "Active 1 minute ago"
		}
		return fmt.Sprintf("Active %d minutes ago", minutes)
	case duration < 24*time.Hour:
		hours := int(duration.Hours())
		if hours == 1 {
			return "Active 1 hour ago"
		}
		return fmt.Sprintf("Active %d hours ago", hours)
	case duration < 7*24*time.Hour:
		days := int(duration.Hours() / 24)
		if days == 1 {
			return "Active 1 day ago"
		}
		return fmt.Sprintf("Active %d days ago", days)
	default:
		return fmt.Sprintf("Active on %s %d", lastActive.Month().String()[:3], lastActive.Day())
	}
}
