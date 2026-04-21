package service

import (
	"errors"
	"fmt"
	"time"

	"github.com/DucLove1/SE357-ShoppingManagement-BE/internal/apperror"
	"github.com/DucLove1/SE357-ShoppingManagement-BE/internal/auth"
	"github.com/DucLove1/SE357-ShoppingManagement-BE/internal/dto"
	"github.com/DucLove1/SE357-ShoppingManagement-BE/internal/model"
	"github.com/DucLove1/SE357-ShoppingManagement-BE/internal/platform/bus"
	"github.com/DucLove1/SE357-ShoppingManagement-BE/internal/platform/cloudinary"
	"github.com/DucLove1/SE357-ShoppingManagement-BE/internal/repo"
	"github.com/DucLove1/SE357-ShoppingManagement-BE/internal/util"
	"github.com/redis/go-redis/v9"

	"go.mongodb.org/mongo-driver/mongo"
	"golang.org/x/crypto/bcrypt"
)

// UserService handles business logic related to user management.
type UserService interface {
	UpdateBuyerProfile(userID string, req *dto.BuyerProfileUpdateRequest) (*dto.UserResponse, error)
	UpdateSellerProfile(userID string, req *dto.SellerProfileUpdateRequest) (*dto.UserResponse, error)

	DeleteUser(id string) error
	ChangePassword(userID, oldPassword, newPassword string) error

	GetUserByID(id string) (*dto.UserResponse, error)
	GetUserByEmail(email string) (*dto.UserResponse, error)
}

type userService struct {
	userRepo    repo.UserRepo
	eventBus    bus.EventBus
	redisClient *redis.Client
}

func NewUserService(userRepo repo.UserRepo, bus bus.EventBus, redisClient *redis.Client) UserService {
	return &userService{
		userRepo:    userRepo,
		eventBus:    bus,
		redisClient: redisClient,
	}
}

func (s *userService) UpdateBuyerProfile(userID string, req *dto.BuyerProfileUpdateRequest) (*dto.UserResponse, error) {
	ctx, cancel := util.NewDefaultDBContext()
	defer cancel()

	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	if user.RoleContent.Buyer == nil {
		user.RoleContent.Buyer = &model.BuyerRoleContent{}
	}

	var oldAvatarPublicID string

	// Cập nhật Avatar nếu có
	if req.AvatarURL != nil && req.PublicID != nil {
		if user.RoleContent.Buyer.Avatar != nil {
			oldAvatarPublicID = user.RoleContent.Buyer.Avatar.PublicID
		}
		user.RoleContent.Buyer.Avatar = &model.Image{
			URL:        *req.AvatarURL,
			PublicID:   *req.PublicID,
			UploadedAt: time.Now(),
		}
	}

	// Cập nhật PhoneNumber nếu có
	if req.PhoneNumber != nil {
		user.RoleContent.Buyer.PhoneNumber = *req.PhoneNumber
	}

	if req.FullName != nil {
		user.FullName = *req.FullName
	}
	// Cập nhật Gender nếu có
	if req.Gender != nil {
		user.RoleContent.Buyer.Gender = *req.Gender
	}

	// Cập nhật DateOfBirth nếu có
	if req.DateOfBirth != nil {
		user.RoleContent.Buyer.DateOfBirth = *req.DateOfBirth
	}

	// Cập nhật Address nếu có
	if req.Address != nil && len(req.Address) > 0 {
		user.RoleContent.Buyer.Address = req.Address
	}

	// Cập nhật DefaultAddressID nếu có
	if req.DefaultAddressID != nil {
		user.RoleContent.Buyer.DefaultAddressID = req.DefaultAddressID
	}

	// Lưu vào database
	updatedUser, err := s.userRepo.Update(ctx, user)
	if err != nil {
		return nil, err
	}

	// Xóa avatar cũ trên Cloudinary (async)
	if oldAvatarPublicID != "" {
		go cloudinary.Delete(oldAvatarPublicID)
	}

	// Publish event nếu avatar thay đổi
	if req.AvatarURL != nil && req.PublicID != nil {
		s.eventBus.Publish(bus.UserChangeAvatarEventType{
			UserID:    userID,
			NewAvatar: *req.AvatarURL,
		})
	}

	return dto.FromUser(updatedUser), nil
}

func (s *userService) UpdateSellerProfile(userID string, req *dto.SellerProfileUpdateRequest) (*dto.UserResponse, error) {
	ctx, cancel := util.NewDefaultDBContext()
	defer cancel()

	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	if user.RoleContent.Seller == nil {
		user.RoleContent.Seller = &model.SellerRoleContent{}
	}

	if req.IdentityCard != nil {
		user.RoleContent.Seller.IdentityCard = *req.IdentityCard
	}

	var oldAvatarPublicID string

	// Cập nhật Avatar nếu có
	if req.AvatarURL != nil && req.AvatarPublicID != nil {
		if user.RoleContent.Seller.Avatar != nil {
			oldAvatarPublicID = user.RoleContent.Seller.Avatar.PublicID
		}
		user.RoleContent.Seller.Avatar = &model.Image{
			URL:        *req.AvatarURL,
			PublicID:   *req.AvatarPublicID,
			UploadedAt: time.Now(),
		}
	}

	if req.BannerURL != nil && req.BannerPublicID != nil {
		if user.RoleContent.Seller.Banner != nil {
			go cloudinary.Delete(user.RoleContent.Seller.Banner.PublicID)
		}
		user.RoleContent.Seller.Banner = &model.Image{
			URL:        *req.BannerURL,
			PublicID:   *req.BannerPublicID,
			UploadedAt: time.Now(),
		}
	}

	if req.IDFrontImageURL != nil && req.IDFrontImagePublicID != nil {
		go cloudinary.Delete(user.RoleContent.Seller.IDFrontImage.PublicID)
		user.RoleContent.Seller.IDFrontImage = model.Image{
			URL:        *req.IDFrontImageURL,
			PublicID:   *req.IDFrontImagePublicID,
			UploadedAt: time.Now(),
		}
	}

	if req.IDBackImageURL != nil && req.IDBackImagePublicID != nil {
		go cloudinary.Delete(user.RoleContent.Seller.IDBackImage.PublicID)
		user.RoleContent.Seller.IDBackImage = model.Image{
			URL:        *req.IDBackImageURL,
			PublicID:   *req.IDBackImagePublicID,
			UploadedAt: time.Now(),
		}
	}

	if req.SelfieWithIDURL != nil && req.SelfieWithIDPublicID != nil {
		go cloudinary.Delete(user.RoleContent.Seller.SelfieWithID.PublicID)
		user.RoleContent.Seller.SelfieWithID = model.Image{
			URL:        *req.SelfieWithIDURL,
			PublicID:   *req.SelfieWithIDPublicID,
			UploadedAt: time.Now(),
		}
	}

	// Cập nhật PhoneNumber nếu có
	if req.PhoneNumber != nil {
		user.RoleContent.Seller.PhoneNumber = *req.PhoneNumber
	}

	if req.FullName != nil {
		user.FullName = *req.FullName
	}

	// Cập nhật Address nếu có
	if req.PickupAddress != nil {
		user.RoleContent.Seller.PickupAddress = *req.PickupAddress
	}

	// Lưu vào database
	updatedUser, err := s.userRepo.Update(ctx, user)
	if err != nil {
		return nil, err
	}

	// Xóa avatar cũ trên Cloudinary (async)
	if oldAvatarPublicID != "" {
		go cloudinary.Delete(oldAvatarPublicID)
	}

	// Publish event nếu avatar thay đổi
	if req.AvatarURL != nil && req.AvatarPublicID != nil {
		s.eventBus.Publish(bus.UserChangeAvatarEventType{
			UserID:    userID,
			NewAvatar: *req.AvatarURL,
		})
	}

	return dto.FromUser(updatedUser), nil
}

func (s *userService) DeleteUser(id string) error {
	ctx, cancel := util.NewDefaultDBContext()
	defer cancel()

	if auth.TokenSvc != nil {
		if err := auth.TokenSvc.InvalidateAllUserTokens(ctx, id); err != nil {
			fmt.Printf("Failed to invalidate tokens for user %s: %v\n", id, err)
		}
	}

	err := s.userRepo.Delete(ctx, id)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return apperror.ErrUserNotFound
		}
		return err
	}
	return nil
}

func (s *userService) ChangePassword(userID, oldPassword, newPassword string) error {
	ctx, cancel := util.NewDefaultDBContext()
	defer cancel()
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return apperror.ErrUserNotFound
		}
		return err
	}

	if bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(oldPassword)) != nil {
		return apperror.ErrInvalidCredentials
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	user.Password = string(hashedPassword)
	_, err = s.userRepo.Update(ctx, user)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return apperror.ErrUserNotFound
		}
		return err
	}
	return nil
}

func (s *userService) GetUserByID(id string) (*dto.UserResponse, error) {
	ctx, cancel := util.NewDefaultDBContext()
	defer cancel()
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, apperror.ErrUserNotFound
		}
		return nil, err
	}
	return dto.FromUser(user), nil
}

func (s *userService) GetUserByEmail(email string) (*dto.UserResponse, error) {
	ctx, cancel := util.NewDefaultDBContext()
	defer cancel()
	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, apperror.ErrUserNotFound
		}
		return nil, err
	}
	return dto.FromUser(user), nil
}
