package dto

import (
	"time"

	"github.com/DucLove1/SE357-ShoppingManagement-BE/internal/model"
)

// ==================== Request DTOs ====================

type CreateCategoryRequest struct {
	ParentID    string `json:"parent_id"`
	Name        string `json:"name" binding:"required,min=1,max=100"`
	Description string `json:"description"`
}

type UpdateCategoryRequest struct {
	ParentID    *string `json:"parent_id"`
	Name        string  `json:"name" binding:"required,min=1,max=100"`
	Description string  `json:"description"`
}

// ==================== Query DTOs ====================

type CategoryFilterQuery struct {
	Search    string `form:"search"`
	ParentID  string `form:"parent_id"`
	Page      int    `form:"page"`
	PageSize  int    `form:"page_size"`
	SortBy    string `form:"sort_by"`
	SortOrder string `form:"sort_order"`
}

// ==================== Response DTOs ====================

type CategoryResponse struct {
	ID          string    `json:"id"`
	ParentID    string    `json:"parent_id,omitempty"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type CategoryTreeResponse struct {
	ID          string                  `json:"id"`
	ParentID    string                  `json:"parent_id,omitempty"`
	Name        string                  `json:"name"`
	Description string                  `json:"description"`
	Children    []*CategoryTreeResponse `json:"children,omitempty"`
	CreatedAt   time.Time               `json:"created_at"`
	UpdatedAt   time.Time               `json:"updated_at"`
}

type PaginatedCategoriesResponse struct {
	Categories []CategoryResponse `json:"categories"`
	Pagination Pagination         `json:"pagination"`
}

// ==================== Converters ====================

func FromCategory(c *model.Category) *CategoryResponse {
	if c == nil {
		return nil
	}

	parentID := ""
	if !c.ParentID.IsZero() {
		parentID = c.ParentID.Hex()
	}

	return &CategoryResponse{
		ID:          c.ID.Hex(),
		ParentID:    parentID,
		Name:        c.Name,
		Description: c.Description,
		CreatedAt:   c.CreatedAt,
		UpdatedAt:   c.UpdatedAt,
	}
}

func FromCategoryList(categories []*model.Category) []CategoryResponse {
	result := make([]CategoryResponse, len(categories))
	for i, c := range categories {
		result[i] = *FromCategory(c)
	}
	return result
}

func FromCategoryToTree(c *model.Category) *CategoryTreeResponse {
	if c == nil {
		return nil
	}

	parentID := ""
	if !c.ParentID.IsZero() {
		parentID = c.ParentID.Hex()
	}

	return &CategoryTreeResponse{
		ID:          c.ID.Hex(),
		ParentID:    parentID,
		Name:        c.Name,
		Description: c.Description,
		Children:    []*CategoryTreeResponse{},
		CreatedAt:   c.CreatedAt,
		UpdatedAt:   c.UpdatedAt,
	}
}
