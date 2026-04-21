package service

import (
	"context"
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

const (
	DefaultCategoryPageSize = 20
	MaxCategoryPageSize     = 100
)

type CategoryService interface {
	// CRUD
	CreateCategory(req *dto.CreateCategoryRequest) (*dto.CategoryResponse, error)
	UpdateCategory(id string, req *dto.UpdateCategoryRequest) (*dto.CategoryResponse, error)
	DeleteCategory(id string) error

	// Read
	GetCategoryByID(id string) (*dto.CategoryResponse, error)
	GetAllCategories() ([]dto.CategoryResponse, error)
	GetCategoriesWithPagination(query *dto.CategoryFilterQuery) (*dto.PaginatedCategoriesResponse, error)
	GetRootCategories() ([]dto.CategoryResponse, error)
	GetChildCategories(parentID string) ([]dto.CategoryResponse, error)
	GetCategoryTree() ([]*dto.CategoryTreeResponse, error)
}

type categoryService struct {
	categoryRepo repo.CategoryRepo
}

func NewCategoryService(categoryRepo repo.CategoryRepo) CategoryService {
	return &categoryService{
		categoryRepo: categoryRepo,
	}
}

// ==================== CRUD Methods ====================

func (s *categoryService) CreateCategory(req *dto.CreateCategoryRequest) (*dto.CategoryResponse, error) {
	ctx, cancel := util.NewDefaultDBContext()
	defer cancel()

	// Check if name already exists
	if _, err := s.categoryRepo.GetByName(ctx, req.Name); err == nil {
		return nil, apperror.ErrCategoryNameExists
	}

	// Validate parent category if provided
	var parentID primitive.ObjectID
	if req.ParentID != "" {
		var err error
		parentID, err = primitive.ObjectIDFromHex(req.ParentID)
		if err != nil {
			return nil, apperror.ErrInvalidID
		}

		// Check parent exists
		if _, err := s.categoryRepo.GetByID(ctx, req.ParentID); err != nil {
			if errors.Is(err, mongo.ErrNoDocuments) {
				return nil, apperror.ErrInvalidParentCategory
			}
			return nil, err
		}
	}

	category := &model.Category{
		ParentID:    parentID,
		Name:        req.Name,
		Description: req.Description,
	}

	createdCategory, err := s.categoryRepo.Create(ctx, category)
	if err != nil {
		return nil, err
	}

	return dto.FromCategory(createdCategory), nil
}

func (s *categoryService) UpdateCategory(id string, req *dto.UpdateCategoryRequest) (*dto.CategoryResponse, error) {
	ctx, cancel := util.NewDefaultDBContext()
	defer cancel()

	// Get existing category
	category, err := s.getCategoryByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Check if new name conflicts with existing (excluding self)
	if existing, err := s.categoryRepo.GetByName(ctx, req.Name); err == nil {
		if existing.ID != category.ID {
			return nil, apperror.ErrCategoryNameExists
		}
	}

	// Validate and set parent category
	if req.ParentID != nil {
		if *req.ParentID == "" {
			category.ParentID = primitive.NilObjectID
		} else {
			parentID, err := primitive.ObjectIDFromHex(*req.ParentID)
			if err != nil {
				return nil, apperror.ErrInvalidID
			}

			// Prevent setting self as parent
			if parentID == category.ID {
				return nil, apperror.ErrInvalidParentCategory
			}

			// Check parent exists
			if _, err := s.categoryRepo.GetByID(ctx, *req.ParentID); err != nil {
				if errors.Is(err, mongo.ErrNoDocuments) {
					return nil, apperror.ErrInvalidParentCategory
				}
				return nil, err
			}

			// Prevent circular reference
			if err := s.checkCircularReference(ctx, category.ID.Hex(), *req.ParentID); err != nil {
				return nil, err
			}

			category.ParentID = parentID
		}
	}

	category.Name = req.Name
	category.Description = req.Description

	updatedCategory, err := s.categoryRepo.Update(ctx, category)
	if err != nil {
		return nil, err
	}

	return dto.FromCategory(updatedCategory), nil
}

func (s *categoryService) DeleteCategory(id string) error {
	ctx, cancel := util.NewDefaultDBContext()
	defer cancel()

	// Check category exists
	if _, err := s.getCategoryByID(ctx, id); err != nil {
		return err
	}

	// Check if has children
	hasChildren, err := s.categoryRepo.HasChildren(ctx, id)
	if err != nil {
		return err
	}
	if hasChildren {
		return apperror.ErrCategoryHasChildren
	}

	// Check if has products
	hasProducts, err := s.categoryRepo.HasProducts(ctx, id)
	if err != nil {
		return err
	}
	if hasProducts {
		return apperror.ErrCategoryHasProducts
	}

	return s.categoryRepo.Delete(ctx, id)
}

// ==================== Read Methods ====================

func (s *categoryService) GetCategoryByID(id string) (*dto.CategoryResponse, error) {
	ctx, cancel := util.NewDefaultDBContext()
	defer cancel()

	category, err := s.getCategoryByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return dto.FromCategory(category), nil
}

func (s *categoryService) GetAllCategories() ([]dto.CategoryResponse, error) {
	ctx, cancel := util.NewDefaultDBContext()
	defer cancel()

	categories, err := s.categoryRepo.GetAll(ctx)
	if err != nil {
		return nil, err
	}

	return dto.FromCategoryList(categories), nil
}

func (s *categoryService) GetCategoriesWithPagination(query *dto.CategoryFilterQuery) (*dto.PaginatedCategoriesResponse, error) {
	ctx, cancel := util.NewDefaultDBContext()
	defer cancel()

	filter := s.buildCategoryFilter(query)
	page, pageSize := s.normalizePagination(query.Page, query.PageSize)

	findOptions := &repo.FindOptions{
		Skip:  int64((page - 1) * pageSize),
		Limit: int64(pageSize),
		Sort:  s.buildSortOptions(query.SortBy, query.SortOrder),
	}

	categories, total, err := s.categoryRepo.Find(ctx, filter, findOptions)
	if err != nil {
		return nil, err
	}

	return &dto.PaginatedCategoriesResponse{
		Categories: dto.FromCategoryList(categories),
		Pagination: dto.Pagination{
			Page:     page,
			PageSize: pageSize,
			Total:    total,
		},
	}, nil
}

func (s *categoryService) GetRootCategories() ([]dto.CategoryResponse, error) {
	ctx, cancel := util.NewDefaultDBContext()
	defer cancel()

	categories, err := s.categoryRepo.GetRootCategories(ctx)
	if err != nil {
		return nil, err
	}

	return dto.FromCategoryList(categories), nil
}

func (s *categoryService) GetChildCategories(parentID string) ([]dto.CategoryResponse, error) {
	ctx, cancel := util.NewDefaultDBContext()
	defer cancel()

	// Validate parent exists
	if _, err := s.getCategoryByID(ctx, parentID); err != nil {
		return nil, err
	}

	categories, err := s.categoryRepo.GetByParentID(ctx, parentID)
	if err != nil {
		return nil, err
	}

	return dto.FromCategoryList(categories), nil
}

func (s *categoryService) GetCategoryTree() ([]*dto.CategoryTreeResponse, error) {
	ctx, cancel := util.NewDefaultDBContext()
	defer cancel()

	// Get all categories
	allCategories, err := s.categoryRepo.GetAll(ctx)
	if err != nil {
		return nil, err
	}

	// Build tree structure
	return s.buildCategoryTree(allCategories), nil
}

// ==================== Helper Methods ====================

func (s *categoryService) getCategoryByID(ctx context.Context, id string) (*model.Category, error) {
	category, err := s.categoryRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, apperror.ErrCategoryNotFound
		}
		return nil, err
	}
	return category, nil
}

func (s *categoryService) checkCircularReference(ctx context.Context, categoryID, newParentID string) error {
	currentID := newParentID

	for currentID != "" {
		if currentID == categoryID {
			return apperror.ErrInvalidParentCategory
		}

		parent, err := s.categoryRepo.GetByID(ctx, currentID)
		if err != nil {
			if errors.Is(err, mongo.ErrNoDocuments) {
				break
			}
			return err
		}

		if parent.ParentID.IsZero() {
			break
		}
		currentID = parent.ParentID.Hex()
	}

	return nil
}

func (s *categoryService) buildCategoryTree(categories []*model.Category) []*dto.CategoryTreeResponse {
	// Create map for quick lookup
	categoryMap := make(map[string]*dto.CategoryTreeResponse)
	for _, c := range categories {
		categoryMap[c.ID.Hex()] = dto.FromCategoryToTree(c)
	}

	// Build tree
	var roots []*dto.CategoryTreeResponse
	for _, c := range categories {
		node := categoryMap[c.ID.Hex()]

		if c.ParentID.IsZero() {
			roots = append(roots, node)
		} else {
			parentID := c.ParentID.Hex()
			if parent, exists := categoryMap[parentID]; exists {
				parent.Children = append(parent.Children, node)
			} else {
				// Parent not found, treat as root
				roots = append(roots, node)
			}
		}
	}

	return roots
}

func (s *categoryService) buildCategoryFilter(query *dto.CategoryFilterQuery) repo.Filter {
	filter := repo.Filter{}

	if query == nil {
		return filter
	}

	if query.Search != "" {
		filter["$or"] = []bson.M{
			{"name": bson.M{"$regex": primitive.Regex{Pattern: query.Search, Options: "i"}}},
			{"description": bson.M{"$regex": primitive.Regex{Pattern: query.Search, Options: "i"}}},
		}
	}

	if query.ParentID != "" {
		if parentObjID, err := primitive.ObjectIDFromHex(query.ParentID); err == nil {
			filter["parent_id"] = parentObjID
		}
	}

	return filter
}

func (s *categoryService) buildSortOptions(sortBy, sortOrder string) map[string]int {
	if sortBy == "" {
		sortBy = "name"
	}

	order := 1 // Default ascending for name
	if sortOrder == "desc" {
		order = -1
	}

	return map[string]int{sortBy: order}
}

func (s *categoryService) normalizePagination(page, pageSize int) (int, int) {
	if page < 1 {
		page = 1
	}

	if pageSize < 1 {
		pageSize = DefaultCategoryPageSize
	}

	if pageSize > MaxCategoryPageSize {
		pageSize = MaxCategoryPageSize
	}

	return page, pageSize
}
