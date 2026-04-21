package model

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type User struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	FullName    string             `bson:"full_name" json:"full_name"`
	Email       string             `bson:"email" json:"email"`
	Password    string             `bson:"password" json:"-"` // Ẩn password
	Role        Role               `bson:"role" json:"role"`
	RoleContent RoleContent        `bson:"role_content,omitempty" json:"role_content,omitempty"`
	UserStatus  UserStatus         `bson:"user_status,omitempty" json:"user_status,omitempty"`
	IsVerified  bool               `bson:"is_verified" json:"is_verified"`
	LastLoginAt *time.Time         `bson:"last_login_at,omitempty" json:"last_login_at,omitempty"`
	CreatedAt   time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt   time.Time          `bson:"updated_at" json:"updated_at"`
	DeletedAt   *time.Time         `bson:"deleted_at,omitempty" json:"deleted_at,omitempty"`
	// Ban fields
	IsBanned  bool       `bson:"is_banned" json:"is_banned"`
	BanUntil  *time.Time `bson:"ban_until,omitempty" json:"ban_until,omitempty"` // null = permanent ban
	BanReason *string    `bson:"ban_reason,omitempty" json:"ban_reason,omitempty"`
}

type UserStatus string

const (
	Banned  UserStatus = "banned"
	Active  UserStatus = "active"
	Deleted UserStatus = "deleted"
)

type Role string

const (
	AdminRole  Role = "admin"
	BuyerRole  Role = "buyer"
	SellerRole Role = "seller"
)

type RoleContent struct {
	Buyer  *BuyerRoleContent  `bson:"buyer,omitempty" json:"buyer,omitempty"`
	Seller *SellerRoleContent `bson:"seller,omitempty" json:"seller,omitempty"`
	Admin  *AdminRoleContent  `bson:"admin,omitempty" json:"admin,omitempty"`
}

// Buyer
type BuyerRoleContent struct {
	Avatar           *Image              `bson:"avatar,omitempty" json:"avatar,omitempty"`
	PhoneNumber      string              `bson:"phone_number,omitempty" json:"phone_number,omitempty"`
	Gender           Gender              `bson:"gender,omitempty" json:"gender,omitempty"`
	DateOfBirth      time.Time           `bson:"date_of_birth,omitempty" json:"date_of_birth,omitempty"`
	Address          []Address           `bson:"address,omitempty" json:"address,omitempty"`
	DefaultAddressID *primitive.ObjectID `bson:"default_address_id,omitempty" json:"default_address_id,omitempty"`

	TotalOrders int `bson:"total_orders,omitempty" json:"total_orders,omitempty"`
	TotalSpent  int `bson:"total_spent,omitempty" json:"total_spent,omitempty"`
}

// Seller
type SellerRoleContent struct {
	Avatar        *Image     `bson:"avatar,omitempty" json:"avatar,omitempty"`
	Banner        *Image     `bson:"banner,omitempty" json:"banner,omitempty"`
	Categories    []Category `bson:"categories,omitempty" json:"categories,omitempty"`
	PickupAddress Address    `bson:"pickup_address,omitempty" json:"pickup_address,omitempty"`
	PhoneNumber   string     `bson:"phone_number,omitempty" json:"phone_number,omitempty"`

	// CMND/CCCD để xác minh
	IdentityCard string `bson:"identity_card,omitempty" json:"identity_card,omitempty"`
	IDFrontImage Image  `bson:"id_front_image,omitempty" json:"id_front_image,omitempty"`
	IDBackImage  Image  `bson:"id_back_image,omitempty" json:"id_back_image,omitempty"`
	SelfieWithID Image  `bson:"selfie_with_id,omitempty" json:"selfie_with_id,omitempty"`

	SellerStatus SellerStatus `bson:"seller_status,omitempty" json:"seller_status,omitempty"`
}
type SellerStatus string

const (
	SellerPending  SellerStatus = "pending"
	SellerActive   SellerStatus = "active"
	SellerRejected SellerStatus = "rejected"
)

// Admin
type AdminRoleContent struct {
	FullName    string             `bson:"full_name" json:"full_name"`
	Permissions []Permission       `bson:"permissions" json:"permissions"`
	CreateAt    time.Time          `bson:"create_at" json:"create_at"`
	CreateBy    primitive.ObjectID `bson:"create_by,omitempty" json:"create_by,omitempty"`
}

type Permission string

const (
	PermissionUserManage    Permission = "user:manage"
	PermissionProductManage Permission = "product:manage"
	PermissionOrderManage   Permission = "order:manage"
	PermissionShopManage    Permission = "shop:manage"
	PermissionReportView    Permission = "report:view"
	PermissionSettingManage Permission = "setting:manage"
)

// Enums
type Gender string

const (
	Male   Gender = "male"
	Female Gender = "female"
	Other  Gender = "other"
)
