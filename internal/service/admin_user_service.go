package service

import (
	"errors"

	"github.com/luong-vh/Digimart_Backend/internal/apperror"
	"github.com/luong-vh/Digimart_Backend/internal/dto"
	"github.com/luong-vh/Digimart_Backend/internal/model"
	"github.com/luong-vh/Digimart_Backend/internal/repo"
	"github.com/luong-vh/Digimart_Backend/internal/util"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type AdminUserService interface {
	BanUser(userID string, req *dto.BanUserRequest) error
	UnbanUser(userID string) error
	SoftDeleteUser(userID string) error
	RestoreUser(userID string) error

	ApproveSeller(userID string) error
	RejectSeller(userID string) error
	GetBuyers(query *dto.GetBuyersQuery) (*dto.PaginatedUsersResponse, error)
	GetSellers(query *dto.GetSellersQuery) (*dto.PaginatedUsersResponse, error)
}

type adminUserService struct {
	userRepo repo.UserRepo
}

func NewAdminUserService(userRepo repo.UserRepo) AdminUserService {
	return &adminUserService{
		userRepo: userRepo,
	}
}

func (s *adminUserService) ApproveSeller(userID string) error {
	ctx, cancel := util.NewDefaultDBContext()
	defer cancel()

	// Get user
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return apperror.ErrUserNotFound
		}
		return err
	}

	// Cannot ban admin
	if user.Role != model.SellerRole {
		return apperror.NewError(nil, apperror.ErrForbidden.Code, "This user is not a seller")
	}

	user.RoleContent.Seller.SellerStatus = model.SellerActive

	_, err = s.userRepo.Update(ctx, user)
	return err
}

func (s *adminUserService) RejectSeller(userID string) error {
	ctx, cancel := util.NewDefaultDBContext()
	defer cancel()

	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return apperror.ErrUserNotFound
		}
		return err
	}
	if user.Role != model.SellerRole {
		return apperror.NewError(nil, apperror.ErrForbidden.Code, "This user is not a seller")
	}

	user.RoleContent.Seller.SellerStatus = model.SellerRejected

	_, err = s.userRepo.Update(ctx, user)
	return err
}
func (s *adminUserService) BanUser(userID string, req *dto.BanUserRequest) error {
	ctx, cancel := util.NewDefaultDBContext()
	defer cancel()

	// Get user
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return apperror.ErrUserNotFound
		}
		return err
	}

	// Cannot ban admin
	if user.Role == model.AdminRole {
		return apperror.NewError(nil, apperror.ErrForbidden.Code, "cannot ban admin user")
	}

	// Update ban fields
	user.IsBanned = true
	user.BanUntil = req.BanUntil // null = permanent
	user.BanReason = &req.Reason

	_, err = s.userRepo.Update(ctx, user)
	return err
}

func (s *adminUserService) UnbanUser(userID string) error {
	ctx, cancel := util.NewDefaultDBContext()
	defer cancel()

	// Get user
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return apperror.ErrUserNotFound
		}
		return err
	}

	// Unban user
	user.IsBanned = false
	user.BanUntil = nil
	user.BanReason = nil

	_, err = s.userRepo.Update(ctx, user)
	return err
}

func (s *adminUserService) SoftDeleteUser(userID string) error {
	ctx, cancel := util.NewDefaultDBContext()
	defer cancel()

	// Get user
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return apperror.ErrUserNotFound
		}
		return err
	}
	// Cannot delete admin
	if user.Role == model.AdminRole {
		return apperror.NewError(nil, apperror.ErrForbidden.Code, "cannot delete admin user")
	}

	err = s.userRepo.SoftDelete(ctx, userID)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return apperror.ErrUserNotFound
		}
		return err
	}

	return nil
}

func (s *adminUserService) RestoreUser(userID string) error {
	ctx, cancel := util.NewDefaultDBContext()
	defer cancel()

	// Get user
	user, err := s.userRepo.GetDeletedByID(ctx, userID)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return apperror.ErrUserNotFound
		}
		return err
	}

	// Check if user is deleted
	if user.DeletedAt == nil {
		return apperror.NewError(nil, apperror.ErrBadRequest.Code, "user is not deleted")
	}

	// Restore user
	user.DeletedAt = nil

	_, err = s.userRepo.Update(ctx, user)
	return err
}

func (s *adminUserService) GetBuyers(query *dto.GetBuyersQuery) (*dto.PaginatedUsersResponse, error) {
	ctx, cancel := util.NewDefaultDBContext()
	defer cancel()

	// Base filter
	filter := repo.Filter{
		"deleted_at": bson.M{"$exists": false},
		"is_banned":  false,
		"role":       model.BuyerRole,
	}

	// ============ SEARCH FILTERS ============

	// Keyword search (fullname OR email)
	if query.Keyword != "" {
		filter["$or"] = []bson.M{
			{"full_name": bson.M{"$regex": primitive.Regex{Pattern: query.Keyword, Options: "i"}}},
			{"email": bson.M{"$regex": primitive.Regex{Pattern: query.Keyword, Options: "i"}}},
		}
	}

	// Specific field search
	if query.Email != "" {
		filter["email"] = bson.M{"$regex": primitive.Regex{Pattern: query.Email, Options: "i"}}
	}

	if query.FullName != "" {
		filter["full_name"] = bson.M{"$regex": primitive.Regex{Pattern: query.FullName, Options: "i"}}
	}

	if query.PhoneNumber != "" {
		filter["role_content.buyer.phone_number"] = bson.M{"$regex": primitive.Regex{Pattern: query.PhoneNumber, Options: "i"}}
	}

	// ============ FILTER BY FIELDS ============

	// Gender
	if query.Gender != "" {
		filter["role_content.buyer.gender"] = query.Gender
	}

	// ============ PAGINATION ============
	page, pageSize := s.normalizePagination(query.Page, query.PageSize)

	findOptions := &repo.FindOptions{
		Skip:  int64((page - 1) * pageSize),
		Limit: int64(pageSize),
	}

	users, total, err := s.userRepo.Find(ctx, filter, findOptions)
	if err != nil {
		return nil, err
	}

	return &dto.PaginatedUsersResponse{
		Users: dto.FromUsers(users),
		Pagination: dto.Pagination{
			Page:     page,
			PageSize: pageSize,
			Total:    total,
		},
	}, nil
}

// ============ GET SELLERS ============
func (s *adminUserService) GetSellers(query *dto.GetSellersQuery) (*dto.PaginatedUsersResponse, error) {
	ctx, cancel := util.NewDefaultDBContext()
	defer cancel()

	// Base filter
	filter := repo.Filter{
		"deleted_at": bson.M{"$exists": false},
		"is_banned":  false,
		"role":       model.SellerRole,
	}

	// ============ SEARCH FILTERS ============

	// Keyword search (fullname OR email)
	if query.Keyword != "" {
		filter["$or"] = []bson.M{
			{"full_name": bson.M{"$regex": primitive.Regex{Pattern: query.Keyword, Options: "i"}}},
			{"email": bson.M{"$regex": primitive.Regex{Pattern: query.Keyword, Options: "i"}}},
		}
	}

	// Specific field search
	if query.Email != "" {
		filter["email"] = bson.M{"$regex": primitive.Regex{Pattern: query.Email, Options: "i"}}
	}

	if query.FullName != "" {
		filter["full_name"] = bson.M{"$regex": primitive.Regex{Pattern: query.FullName, Options: "i"}}
	}

	if query.PhoneNumber != "" {
		filter["role_content.seller.phone_number"] = bson.M{"$regex": primitive.Regex{Pattern: query.PhoneNumber, Options: "i"}}
	}

	// ============ FILTER BY FIELDS ============

	// Seller Status
	if query.SellerStatus != "" {
		filter["role_content.seller.seller_status"] = query.SellerStatus
	}

	// Category
	if query.CategoryID != nil {
		filter["role_content.seller.categories._id"] = *query.CategoryID
	}

	// ============ PAGINATION ============
	page, pageSize := s.normalizePagination(query.Page, query.PageSize)

	findOptions := &repo.FindOptions{
		Skip:  int64((page - 1) * pageSize),
		Limit: int64(pageSize),
	}

	users, total, err := s.userRepo.Find(ctx, filter, findOptions)
	if err != nil {
		return nil, err
	}

	return &dto.PaginatedUsersResponse{
		Users: dto.FromUsers(users),
		Pagination: dto.Pagination{
			Page:     page,
			PageSize: pageSize,
			Total:    total,
		},
	}, nil
}
func (s *adminUserService) normalizePagination(page, pageSize int) (int, int) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}
	if pageSize > 100 {
		pageSize = 100
	}
	return page, pageSize
}
