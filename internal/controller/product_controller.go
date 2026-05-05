package controller

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/luong-vh/Digimart_Backend/internal/apperror"
	"github.com/luong-vh/Digimart_Backend/internal/dto"
	"github.com/luong-vh/Digimart_Backend/internal/model"
	"github.com/luong-vh/Digimart_Backend/internal/repo"
	"github.com/luong-vh/Digimart_Backend/internal/service"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ProductController struct {
	service service.ProductService
}

func NewProductController(service service.ProductService) *ProductController {
	return &ProductController{service: service}
}

// ==================== PUBLIC ENDPOINTS ====================

// GetProductByID retrieves a product by its ID
func (c *ProductController) GetProductByID(ctx *gin.Context) {
	productID := ctx.Param("id")

	product, err := c.service.GetProductByID(ctx, productID)
	if err != nil {
		dto.SendError(ctx, apperror.StatusFromError(err), apperror.Message(err), apperror.Code(err))
		return
	}

	// Increment view count asynchronously

	dto.SendSuccess(ctx, http.StatusOK, "Product retrieved successfully", product)
}

// GetProducts retrieves products with filtering and pagination
func (c *ProductController) GetProducts(ctx *gin.Context) {
	filter := repo.Filter{}

	// Parse query parameters
	if categoryID := ctx.Query("category_id"); categoryID != "" {
		if objID, err := primitive.ObjectIDFromHex(categoryID); err == nil {
			filter["category_id"] = objID
		}
	}

	if status := ctx.Query("status"); status != "" {
		filter["status"] = model.ProductStatus(status)
	} else {
		// Default to active products for public listing
		filter["status"] = model.ProductStatusActive
	}

	if search := ctx.Query("search"); search != "" {
		filter["$or"] = []bson.M{
			{"name": bson.M{"$regex": search, "$options": "i"}},
			{"description": bson.M{"$regex": search, "$options": "i"}},
		}
	}

	// Price range filter
	if minPrice := ctx.Query("min_price"); minPrice != "" {
		if price, err := strconv.ParseFloat(minPrice, 64); err == nil {
			filter["price"] = bson.M{"$gte": price}
		}
	}

	if maxPrice := ctx.Query("max_price"); maxPrice != "" {
		if price, err := strconv.ParseFloat(maxPrice, 64); err == nil {
			if existing, ok := filter["price"].(bson.M); ok {
				existing["$lte"] = price
			} else {
				filter["price"] = bson.M{"$lte": price}
			}
		}
	}

	// Pagination
	opts := &repo.FindOptions{}

	if page := ctx.Query("page"); page != "" {
		if p, err := strconv.ParseInt(page, 10, 64); err == nil && p > 0 {
			limit := int64(20) // Default limit
			if l := ctx.Query("limit"); l != "" {
				if parsedLimit, err := strconv.ParseInt(l, 10, 64); err == nil && parsedLimit > 0 {
					limit = parsedLimit
				}
			}
			opts.Limit = limit
			opts.Skip = (p - 1) * limit
		}
	} else if limit := ctx.Query("limit"); limit != "" {
		if l, err := strconv.ParseInt(limit, 10, 64); err == nil && l > 0 {
			opts.Limit = l
		}
	}

	// Sorting
	sortField := ctx.DefaultQuery("sort_by", "created_at")
	sortOrder := ctx.DefaultQuery("sort_order", "desc")
	order := -1
	if sortOrder == "asc" {
		order = 1
	}
	opts.Sort = map[string]int{sortField: order}

	products, total, err := c.service.FindProducts(ctx, filter, opts)
	if err != nil {
		dto.SendError(ctx, apperror.StatusFromError(err), apperror.Message(err), apperror.Code(err))
		return
	}

	dto.SendSuccessWithPagination(ctx, http.StatusOK, "Products retrieved successfully", products, total, opts.Skip, opts.Limit)
}

// GetProductsBySeller retrieves products by seller ID

// ==================== SELLER ENDPOINTS ====================

// CreateProduct creates a new product (Seller only)
func (c *ProductController) CreateProduct(ctx *gin.Context) {

	var req dto.CreateProductRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		dto.SendError(ctx, http.StatusBadRequest, "Invalid request payload", apperror.ErrBadRequest.Code)
		return
	}

	product, err := c.service.CreateProduct(ctx, req)
	if err != nil {
		dto.SendError(ctx, apperror.StatusFromError(err), apperror.Message(err), apperror.Code(err))
		return
	}

	dto.SendSuccess(ctx, http.StatusCreated, "Product created successfully", product)
}

// UpdateProduct updates an existing product (Seller only - own products)
func (c *ProductController) UpdateProduct(ctx *gin.Context) {

	productID := ctx.Param("id")

	var req dto.UpdateProductRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		dto.SendError(ctx, http.StatusBadRequest, "Invalid request payload", apperror.ErrBadRequest.Code)
		return
	}

	product, err := c.service.UpdateProduct(ctx, productID, req)
	if err != nil {
		dto.SendError(ctx, apperror.StatusFromError(err), apperror.Message(err), apperror.Code(err))
		return
	}

	dto.SendSuccess(ctx, http.StatusOK, "Product updated successfully", product)
}

// DeleteProduct deletes a product (Seller only - own products)
func (c *ProductController) DeleteProduct(ctx *gin.Context) {

	productID := ctx.Param("id")

	err := c.service.DeleteProduct(ctx, productID)
	if err != nil {
		dto.SendError(ctx, apperror.StatusFromError(err), apperror.Message(err), apperror.Code(err))
		return
	}

	dto.SendSuccess(ctx, http.StatusOK, "Product deleted successfully", gin.H{"id": productID})
}
