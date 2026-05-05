package service

import (
	"errors"
	"fmt"
	"time"

	"github.com/luong-vh/Digimart_Backend/internal/apperror"
	"github.com/luong-vh/Digimart_Backend/internal/auth"
	"github.com/luong-vh/Digimart_Backend/internal/dto"
	"github.com/luong-vh/Digimart_Backend/internal/model"
	"github.com/luong-vh/Digimart_Backend/internal/platform/bus"
	"github.com/luong-vh/Digimart_Backend/internal/platform/cloudinary"
	"github.com/luong-vh/Digimart_Backend/internal/repo"
	"github.com/luong-vh/Digimart_Backend/internal/util"
	"github.com/redis/go-redis/v9"

	"go.mongodb.org/mongo-driver/mongo"
	"golang.org/x/crypto/bcrypt"
)

// UserService handles business logic related to user management.
type UserService interface {
	UpdateBuyerProfile(userID string, req *dto.BuyerProfileUpdateRequest) (*dto.UserResponse, error)

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
