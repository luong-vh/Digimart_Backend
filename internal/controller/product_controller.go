package controller

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/luong-vh/Digimart_Backend/internal/apperror"
	"github.com/luong-vh/Digimart_Backend/internal/auth"
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

	product, err := c.service.GetProductByID(productID)
	if err != nil {
		dto.SendError(ctx, apperror.StatusFromError(err), apperror.Message(err), apperror.Code(err))
		return
	}

	// Increment view count asynchronously
	go c.service.IncrementViewCount(productID)

	dto.SendSuccess(ctx, http.StatusOK, "Product retrieved successfully", product)
}

// GetProductBySlug retrieves a product by its slug
func (c *ProductController) GetProductBySlug(ctx *gin.Context) {
	slug := ctx.Param("slug")

	product, err := c.service.GetProductBySlug(slug)
	if err != nil {
		dto.SendError(ctx, apperror.StatusFromError(err), apperror.Message(err), apperror.Code(err))
		return
	}

	// Increment view count asynchronously
	go c.service.IncrementViewCount(product.ID)

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

	if sellerID := ctx.Query("seller_id"); sellerID != "" {
		if objID, err := primitive.ObjectIDFromHex(sellerID); err == nil {
			filter["seller_id"] = objID
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

	products, total, err := c.service.FindProducts(filter, opts)
	if err != nil {
		dto.SendError(ctx, apperror.StatusFromError(err), apperror.Message(err), apperror.Code(err))
		return
	}

	dto.SendSuccessWithPagination(ctx, http.StatusOK, "Products retrieved successfully", products, total, opts.Skip, opts.Limit)
}

// GetProductsByCategory retrieves products by category ID
func (c *ProductController) GetProductsByCategory(ctx *gin.Context) {
	categoryID := ctx.Param("categoryId")

	products, err := c.service.GetProductsByCategoryID(categoryID)
	if err != nil {
		dto.SendError(ctx, apperror.StatusFromError(err), apperror.Message(err), apperror.Code(err))
		return
	}

	dto.SendSuccess(ctx, http.StatusOK, "Products retrieved successfully", products)
}

// GetProductsBySeller retrieves products by seller ID
func (c *ProductController) GetProductsBySeller(ctx *gin.Context) {
	sellerID := ctx.Param("sellerId")

	products, err := c.service.GetProductsBySellerID(sellerID)
	if err != nil {
		dto.SendError(ctx, apperror.StatusFromError(err), apperror.Message(err), apperror.Code(err))
		return
	}

	dto.SendSuccess(ctx, http.StatusOK, "Products retrieved successfully", products)
}

// ==================== SELLER ENDPOINTS ====================

// CreateProduct creates a new product (Seller only)
func (c *ProductController) CreateProduct(ctx *gin.Context) {
	authUser, exists := ctx.Get("authUser")
	if !exists {
		dto.SendError(ctx, http.StatusUnauthorized, apperror.ErrForbidden.Message, apperror.ErrForbidden.Code)
		return
	}

	sellerID := authUser.(auth.AuthUser).ID

	var req dto.CreateProductRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		dto.SendError(ctx, http.StatusBadRequest, "Invalid request payload", apperror.ErrBadRequest.Code)
		return
	}

	product, err := c.service.CreateProduct(sellerID, &req)
	if err != nil {
		dto.SendError(ctx, apperror.StatusFromError(err), apperror.Message(err), apperror.Code(err))
		return
	}

	dto.SendSuccess(ctx, http.StatusCreated, "Product created successfully", product)
}

// UpdateProduct updates an existing product (Seller only - own products)
func (c *ProductController) UpdateProduct(ctx *gin.Context) {
	authUser, exists := ctx.Get("authUser")
	if !exists {
		dto.SendError(ctx, http.StatusUnauthorized, apperror.ErrForbidden.Message, apperror.ErrForbidden.Code)
		return
	}

	sellerID := authUser.(auth.AuthUser).ID
	productID := ctx.Param("id")

	var req dto.UpdateProductRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		dto.SendError(ctx, http.StatusBadRequest, "Invalid request payload", apperror.ErrBadRequest.Code)
		return
	}

	product, err := c.service.UpdateProduct(productID, sellerID, &req)
	if err != nil {
		dto.SendError(ctx, apperror.StatusFromError(err), apperror.Message(err), apperror.Code(err))
		return
	}

	dto.SendSuccess(ctx, http.StatusOK, "Product updated successfully", product)
}

// DeleteProduct deletes a product (Seller only - own products)
func (c *ProductController) DeleteProduct(ctx *gin.Context) {
	authUser, exists := ctx.Get("authUser")
	if !exists {
		dto.SendError(ctx, http.StatusUnauthorized, apperror.ErrForbidden.Message, apperror.ErrForbidden.Code)
		return
	}

	sellerID := authUser.(auth.AuthUser).ID
	productID := ctx.Param("id")

	err := c.service.DeleteProduct(productID, sellerID)
	if err != nil {
		dto.SendError(ctx, apperror.StatusFromError(err), apperror.Message(err), apperror.Code(err))
		return
	}

	dto.SendSuccess(ctx, http.StatusOK, "Product deleted successfully", gin.H{"id": productID})
}

// GetMyProducts retrieves all products of the authenticated seller
func (c *ProductController) GetMyProducts(ctx *gin.Context) {
	authUser, exists := ctx.Get("authUser")
	if !exists {
		dto.SendError(ctx, http.StatusUnauthorized, apperror.ErrForbidden.Message, apperror.ErrForbidden.Code)
		return
	}

	sellerID := authUser.(auth.AuthUser).ID

	products, err := c.service.GetProductsBySellerID(sellerID)
	if err != nil {
		dto.SendError(ctx, apperror.StatusFromError(err), apperror.Message(err), apperror.Code(err))
		return
	}

	dto.SendSuccess(ctx, http.StatusOK, "Products retrieved successfully", products)
}

// UpdateProductStatus updates the status of a product (Seller only - own products)
func (c *ProductController) UpdateProductStatus(ctx *gin.Context) {
	authUser, exists := ctx.Get("authUser")
	if !exists {
		dto.SendError(ctx, http.StatusUnauthorized, apperror.ErrForbidden.Message, apperror.ErrForbidden.Code)
		return
	}

	sellerID := authUser.(auth.AuthUser).ID
	productID := ctx.Param("id")

	var req dto.UpdateProductStatusRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		dto.SendError(ctx, http.StatusBadRequest, "Invalid request payload", apperror.ErrBadRequest.Code)
		return
	}

	product, err := c.service.UpdateProductStatus(productID, sellerID, req.Status)
	if err != nil {
		dto.SendError(ctx, apperror.StatusFromError(err), apperror.Message(err), apperror.Code(err))
		return
	}

	dto.SendSuccess(ctx, http.StatusOK, "Product status updated successfully", product)
}

// ==================== VARIANT ENDPOINTS ====================

// AddVariant adds a new variant to a product (Seller only)
func (c *ProductController) AddVariant(ctx *gin.Context) {
	authUser, exists := ctx.Get("authUser")
	if !exists {
		dto.SendError(ctx, http.StatusUnauthorized, apperror.ErrForbidden.Message, apperror.ErrForbidden.Code)
		return
	}

	sellerID := authUser.(auth.AuthUser).ID
	productID := ctx.Param("id")

	var req dto.AddVariantRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		dto.SendError(ctx, http.StatusBadRequest, "Invalid request payload", apperror.ErrBadRequest.Code)
		return
	}

	product, err := c.service.AddVariant(productID, sellerID, &req)
	if err != nil {
		dto.SendError(ctx, apperror.StatusFromError(err), apperror.Message(err), apperror.Code(err))
		return
	}

	dto.SendSuccess(ctx, http.StatusCreated, "Variant added successfully", product)
}

// UpdateVariant updates a variant of a product (Seller only)
func (c *ProductController) UpdateVariant(ctx *gin.Context) {
	authUser, exists := ctx.Get("authUser")
	if !exists {
		dto.SendError(ctx, http.StatusUnauthorized, apperror.ErrForbidden.Message, apperror.ErrForbidden.Code)
		return
	}

	sellerID := authUser.(auth.AuthUser).ID
	productID := ctx.Param("id")
	variantID := ctx.Param("variantId")

	var req dto.UpdateVariantRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		dto.SendError(ctx, http.StatusBadRequest, "Invalid request payload", apperror.ErrBadRequest.Code)
		return
	}

	product, err := c.service.UpdateVariant(productID, variantID, sellerID, &req)
	if err != nil {
		dto.SendError(ctx, apperror.StatusFromError(err), apperror.Message(err), apperror.Code(err))
		return
	}

	dto.SendSuccess(ctx, http.StatusOK, "Variant updated successfully", product)
}

// DeleteVariant deletes a variant from a product (Seller only)
func (c *ProductController) DeleteVariant(ctx *gin.Context) {
	authUser, exists := ctx.Get("authUser")
	if !exists {
		dto.SendError(ctx, http.StatusUnauthorized, apperror.ErrForbidden.Message, apperror.ErrForbidden.Code)
		return
	}

	sellerID := authUser.(auth.AuthUser).ID
	productID := ctx.Param("id")
	variantID := ctx.Param("variantId")

	product, err := c.service.DeleteVariant(productID, variantID, sellerID)
	if err != nil {
		dto.SendError(ctx, apperror.StatusFromError(err), apperror.Message(err), apperror.Code(err))
		return
	}

	dto.SendSuccess(ctx, http.StatusOK, "Variant deleted successfully", product)
}

// ==================== INVENTORY ENDPOINTS ====================

// UpdateStock updates the stock quantity of a product (Seller only)
func (c *ProductController) UpdateStock(ctx *gin.Context) {
	authUser, exists := ctx.Get("authUser")
	if !exists {
		dto.SendError(ctx, http.StatusUnauthorized, apperror.ErrForbidden.Message, apperror.ErrForbidden.Code)
		return
	}

	sellerID := authUser.(auth.AuthUser).ID
	productID := ctx.Param("id")

	var req dto.UpdateStockRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		dto.SendError(ctx, http.StatusBadRequest, "Invalid request payload", apperror.ErrBadRequest.Code)
		return
	}

	err := c.service.UpdateStock(productID, sellerID, req.Quantity)
	if err != nil {
		dto.SendError(ctx, apperror.StatusFromError(err), apperror.Message(err), apperror.Code(err))
		return
	}

	dto.SendSuccess(ctx, http.StatusOK, "Stock updated successfully", gin.H{"product_id": productID, "quantity": req.Quantity})
}

// UpdateVariantStock updates the stock quantity of a variant (Seller only)
func (c *ProductController) UpdateVariantStock(ctx *gin.Context) {
	authUser, exists := ctx.Get("authUser")
	if !exists {
		dto.SendError(ctx, http.StatusUnauthorized, apperror.ErrForbidden.Message, apperror.ErrForbidden.Code)
		return
	}

	sellerID := authUser.(auth.AuthUser).ID
	productID := ctx.Param("id")
	variantID := ctx.Param("variantId")

	var req dto.UpdateStockRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		dto.SendError(ctx, http.StatusBadRequest, "Invalid request payload", apperror.ErrBadRequest.Code)
		return
	}

	err := c.service.UpdateVariantStock(productID, variantID, sellerID, req.Quantity)
	if err != nil {
		dto.SendError(ctx, apperror.StatusFromError(err), apperror.Message(err), apperror.Code(err))
		return
	}

	dto.SendSuccess(ctx, http.StatusOK, "Variant stock updated successfully", gin.H{
		"product_id": productID,
		"variant_id": variantID,
		"quantity":   req.Quantity,
	})
}

// ==================== ADMIN ENDPOINTS ====================

// GetProductStats retrieves product statistics (Admin only)
func (c *ProductController) GetProductStats(ctx *gin.Context) {
	stats, err := c.service.GetProductStats()
	if err != nil {
		dto.SendError(ctx, apperror.StatusFromError(err), apperror.Message(err), apperror.Code(err))
		return
	}

	dto.SendSuccess(ctx, http.StatusOK, "Product stats retrieved successfully", stats)
}

// AdminUpdateProductStatus allows admin to update any product's status
func (c *ProductController) AdminUpdateProductStatus(ctx *gin.Context) {
	productID := ctx.Param("id")

	var req dto.UpdateProductStatusRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		dto.SendError(ctx, http.StatusBadRequest, "Invalid request payload", apperror.ErrBadRequest.Code)
		return
	}

	// Get product first to find seller ID
	product, err := c.service.GetProductByID(productID)
	if err != nil {
		dto.SendError(ctx, apperror.StatusFromError(err), apperror.Message(err), apperror.Code(err))
		return
	}

	updatedProduct, err := c.service.UpdateProductStatus(productID, product.SellerID, req.Status)
	if err != nil {
		dto.SendError(ctx, apperror.StatusFromError(err), apperror.Message(err), apperror.Code(err))
		return
	}

	dto.SendSuccess(ctx, http.StatusOK, "Product status updated successfully", updatedProduct)
}

// AdminDeleteProduct allows admin to delete any product
func (c *ProductController) AdminDeleteProduct(ctx *gin.Context) {
	productID := ctx.Param("id")

	// Get product first to find seller ID
	product, err := c.service.GetProductByID(productID)
	if err != nil {
		dto.SendError(ctx, apperror.StatusFromError(err), apperror.Message(err), apperror.Code(err))
		return
	}

	err = c.service.DeleteProduct(productID, product.SellerID)
	if err != nil {
		dto.SendError(ctx, apperror.StatusFromError(err), apperror.Message(err), apperror.Code(err))
		return
	}

	dto.SendSuccess(ctx, http.StatusOK, "Product deleted successfully", gin.H{"id": productID})
}
