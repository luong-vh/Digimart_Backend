package controller

import (
	"net/http"
	"regexp"
	"strconv"
	"strings"

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
	andFilters := make([]bson.M, 0, 2)

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
		escapedSearch := regexp.QuoteMeta(search)
		andFilters = append(andFilters, bson.M{
			"$or": []bson.M{
				{"name": bson.M{"$regex": escapedSearch, "$options": "i"}},
				{"description": bson.M{"$regex": escapedSearch, "$options": "i"}},
				{"brand": bson.M{"$regex": escapedSearch, "$options": "i"}},
				{"specs.label": bson.M{"$regex": escapedSearch, "$options": "i"}},
				{"specs.value": bson.M{"$regex": escapedSearch, "$options": "i"}},
			},
		})
	}

	if brand := strings.TrimSpace(ctx.Query("brand")); brand != "" {
		filter["brand"] = bson.M{"$regex": regexp.QuoteMeta(brand), "$options": "i"}
	}

	if specKey := strings.TrimSpace(ctx.Query("spec_key")); specKey != "" {
		specValue := strings.TrimSpace(ctx.Query("spec_value"))
		specMatch := bson.M{
			"label": bson.M{"$regex": regexp.QuoteMeta(specKey), "$options": "i"},
		}
		if specValue != "" {
			specMatch["value"] = bson.M{"$regex": regexp.QuoteMeta(specValue), "$options": "i"}
		}
		andFilters = append(andFilters, bson.M{"specs": bson.M{"$elemMatch": specMatch}})
	}

	if specs := strings.TrimSpace(ctx.Query("specs")); specs != "" {
		for _, pair := range strings.Split(specs, ",") {
			key, value, ok := strings.Cut(pair, ":")
			key = strings.TrimSpace(key)
			value = strings.TrimSpace(value)
			if !ok || key == "" || value == "" {
				continue
			}
			andFilters = append(andFilters, bson.M{
				"specs": bson.M{"$elemMatch": bson.M{
					"label": bson.M{"$regex": regexp.QuoteMeta(key), "$options": "i"},
					"value": bson.M{"$regex": regexp.QuoteMeta(value), "$options": "i"},
				}},
			})
		}
	}

	// Price range filter
	if minPrice := ctx.Query("min_price"); minPrice != "" {
		if price, err := strconv.ParseFloat(minPrice, 64); err == nil {
			filter["base_price"] = bson.M{"$gte": price}
		}
	}

	if maxPrice := ctx.Query("max_price"); maxPrice != "" {
		if price, err := strconv.ParseFloat(maxPrice, 64); err == nil {
			if existing, ok := filter["base_price"].(bson.M); ok {
				existing["$lte"] = price
			} else {
				filter["base_price"] = bson.M{"$lte": price}
			}
		}
	}

	if inStock := strings.ToLower(strings.TrimSpace(ctx.Query("in_stock"))); inStock == "true" || inStock == "1" {
		andFilters = append(andFilters, bson.M{
			"$or": []bson.M{
				{"is_has_variant": false, "stock_quantity": bson.M{"$gt": 0}},
				{"is_has_variant": true, "variants.stock_quantity": bson.M{"$gt": 0}},
			},
		})
	}

	if len(andFilters) > 0 {
		filter["$and"] = andFilters
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
	if sortField == "price" {
		sortField = "base_price"
	}
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

	dto.SendSuccessWithPagination(ctx, http.StatusOK, "Products retrieved successfully", products, opts.Skip, opts.Limit, total)
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
