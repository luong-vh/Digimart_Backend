package controller

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/luong-vh/Digimart_Backend/internal/apperror"
	"github.com/luong-vh/Digimart_Backend/internal/dto"
	"github.com/luong-vh/Digimart_Backend/internal/service"
)

type CategoryController struct {
	service service.CategoryService
}

func NewCategoryController(service service.CategoryService) *CategoryController {
	return &CategoryController{service: service}
}

// ==================== Public Endpoints ====================

// GetAllCategories retrieves all categories
func (c *CategoryController) GetAllCategories(ctx *gin.Context) {
	categories, err := c.service.GetAllCategories()
	if err != nil {
		dto.SendError(ctx, apperror.StatusFromError(err), apperror.Message(err), apperror.Code(err))
		return
	}

	dto.SendSuccess(ctx, http.StatusOK, "Categories retrieved successfully", categories)
}

// GetCategoriesWithPagination retrieves categories with pagination
func (c *CategoryController) GetCategoriesWithPagination(ctx *gin.Context) {
	var query dto.CategoryFilterQuery
	if err := ctx.ShouldBindQuery(&query); err != nil {
		dto.SendError(ctx, http.StatusBadRequest, apperror.ErrBadRequest.Message, apperror.ErrBadRequest.Code)
		return
	}

	result, err := c.service.GetCategoriesWithPagination(&query)
	if err != nil {
		dto.SendError(ctx, apperror.StatusFromError(err), apperror.Message(err), apperror.Code(err))
		return
	}

	dto.SendSuccess(ctx, http.StatusOK, "Categories retrieved successfully", result)
}

// GetCategoryByID retrieves a category by ID
func (c *CategoryController) GetCategoryByID(ctx *gin.Context) {
	id := ctx.Param("id")
	if id == "" {
		dto.SendError(ctx, http.StatusBadRequest, apperror.ErrInvalidID.Message, apperror.ErrInvalidID.Code)
		return
	}

	category, err := c.service.GetCategoryByID(id)
	if err != nil {
		dto.SendError(ctx, apperror.StatusFromError(err), apperror.Message(err), apperror.Code(err))
		return
	}

	dto.SendSuccess(ctx, http.StatusOK, "Category retrieved successfully", category)
}

// GetRootCategories retrieves root categories (no parent)
func (c *CategoryController) GetRootCategories(ctx *gin.Context) {
	categories, err := c.service.GetRootCategories()
	if err != nil {
		dto.SendError(ctx, apperror.StatusFromError(err), apperror.Message(err), apperror.Code(err))
		return
	}

	dto.SendSuccess(ctx, http.StatusOK, "Root categories retrieved successfully", categories)
}

// GetChildCategories retrieves child categories of a parent
func (c *CategoryController) GetChildCategories(ctx *gin.Context) {
	// Đổi từ "parentId" thành "id"
	parentID := ctx.Param("id")
	if parentID == "" {
		dto.SendError(ctx, http.StatusBadRequest, apperror.ErrInvalidID.Message, apperror.ErrInvalidID.Code)
		return
	}

	categories, err := c.service.GetChildCategories(parentID)
	if err != nil {
		dto.SendError(ctx, apperror.StatusFromError(err), apperror.Message(err), apperror.Code(err))
		return
	}

	dto.SendSuccess(ctx, http.StatusOK, "Child categories retrieved successfully", categories)
}

// GetCategoryTree retrieves category tree structure
func (c *CategoryController) GetCategoryTree(ctx *gin.Context) {
	tree, err := c.service.GetCategoryTree()
	if err != nil {
		dto.SendError(ctx, apperror.StatusFromError(err), apperror.Message(err), apperror.Code(err))
		return
	}

	dto.SendSuccess(ctx, http.StatusOK, "Category tree retrieved successfully", tree)
}

// ==================== Seller/Admin Endpoints ====================

// CreateCategory creates a new category (Seller & Admin)
func (c *CategoryController) CreateCategory(ctx *gin.Context) {
	var req dto.CreateCategoryRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		dto.SendError(ctx, http.StatusBadRequest, apperror.ErrBadRequest.Message, apperror.ErrBadRequest.Code)
		return
	}

	category, err := c.service.CreateCategory(&req)
	if err != nil {
		dto.SendError(ctx, apperror.StatusFromError(err), apperror.Message(err), apperror.Code(err))
		return
	}

	dto.SendSuccess(ctx, http.StatusCreated, "Category created successfully", category)
}

// ==================== Admin Only Endpoints ====================

// UpdateCategory updates an existing category (Admin only)
func (c *CategoryController) UpdateCategory(ctx *gin.Context) {
	id := ctx.Param("id")
	if id == "" {
		dto.SendError(ctx, http.StatusBadRequest, apperror.ErrInvalidID.Message, apperror.ErrInvalidID.Code)
		return
	}

	var req dto.UpdateCategoryRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		dto.SendError(ctx, http.StatusBadRequest, apperror.ErrBadRequest.Message, apperror.ErrBadRequest.Code)
		return
	}

	category, err := c.service.UpdateCategory(id, &req)
	if err != nil {
		dto.SendError(ctx, apperror.StatusFromError(err), apperror.Message(err), apperror.Code(err))
		return
	}

	dto.SendSuccess(ctx, http.StatusOK, "Category updated successfully", category)
}

// DeleteCategory deletes a category (Admin only)
func (c *CategoryController) DeleteCategory(ctx *gin.Context) {
	id := ctx.Param("id")
	if id == "" {
		dto.SendError(ctx, http.StatusBadRequest, apperror.ErrInvalidID.Message, apperror.ErrInvalidID.Code)
		return
	}

	err := c.service.DeleteCategory(id)
	if err != nil {
		dto.SendError(ctx, apperror.StatusFromError(err), apperror.Message(err), apperror.Code(err))
		return
	}

	dto.SendSuccess(ctx, http.StatusOK, "Category deleted successfully", gin.H{"id": id})
}
