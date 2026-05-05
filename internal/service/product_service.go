package service

import (
	"context"
	"errors"
	"time"

	"github.com/luong-vh/Digimart_Backend/internal/apperror"
	"github.com/luong-vh/Digimart_Backend/internal/dto"
	"github.com/luong-vh/Digimart_Backend/internal/model"
	"github.com/luong-vh/Digimart_Backend/internal/platform/bus"
	"github.com/luong-vh/Digimart_Backend/internal/platform/cloudinary"
	"github.com/luong-vh/Digimart_Backend/internal/repo"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type ProductService interface {
	// ── CRUD ────────────────────────────────────────────
	CreateProduct(ctx context.Context, req dto.CreateProductRequest) (*dto.ProductResponse, error)
	UpdateProduct(ctx context.Context, productID string, req dto.UpdateProductRequest) (*dto.ProductResponse, error)
	DeleteProduct(ctx context.Context, productID string) error     // hard delete (admin)
	SoftDeleteProduct(ctx context.Context, productID string) error // soft delete (seller)
	GetProductByID(ctx context.Context, productID string) (*dto.ProductResponse, error)

	// ── List / Search ────────────────────────────────────
	FindProducts(ctx context.Context, filter repo.Filter, opts *repo.FindOptions) ([]*dto.ProductResponse, int64, error)

	// ── Media ────────────────────────────────────────────
	AddProductImages(ctx context.Context, productID string, images []model.Image) error
	AddProductVideos(ctx context.Context, productID string, videos []model.Video) error

	// ── Variant ──────────────────────────────────────────
	CreateVariant(ctx context.Context, productID string, req dto.CreateVariantRequest) (*dto.VariantResponse, error)
	UpdateVariant(ctx context.Context, productID string, variantID string, req dto.UpdateVariantRequest) (*dto.VariantResponse, error)

	// ── Cart support ─────────────────────────────────────
	CheckAvailability(ctx context.Context, productID string, variantID *string, qty int) (bool, int, error)

	// ── Order support ────────────────────────────────────
	IncrementSoldCount(ctx context.Context, productID string, count int) error
	UpdateRating(ctx context.Context, productID string, newRating float64) error
}

type productService struct {
	productRepo repo.ProductRepo
	userRepo    repo.UserRepo
	eventBus    bus.EventBus
	redisClient *redis.Client
}

func NewProductService(productRepo repo.ProductRepo, userRepo repo.UserRepo, eventBus bus.EventBus, redisClient *redis.Client) ProductService {
	return &productService{
		productRepo: productRepo,
		userRepo:    userRepo,
		eventBus:    eventBus,
		redisClient: redisClient,
	}
}

func (s *productService) CreateProduct(ctx context.Context, req dto.CreateProductRequest) (*dto.ProductResponse, error) {

	categoryObjID, err := primitive.ObjectIDFromHex(req.CategoryID)
	if err != nil {
		return nil, apperror.ErrInvalidID
	}

	product := &model.Product{
		CategoryID:      categoryObjID,
		Name:            req.Name,
		Description:     req.Description,
		Brand:           req.Brand,
		Thumbnail:       req.Thumbnail,
		Images:          req.Images,
		Videos:          req.Videos,
		BasePrice:       req.BasePrice,
		DiscountPercent: req.DiscountPercent,
		IsHasVariant:    req.IsHasVariant,
		StockQuantity:   req.StockQuantity,
		Variants:        req.Variants,
		Status:          model.ProductStatusDraft,
		SoldCount:       0,
		Rating:          0,
		RatingCount:     0,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}
	for i := range product.Variants {
		product.Variants[i].FinalPrice = (product.BasePrice + product.Variants[i].PriceAdjustment) * (1 - product.DiscountPercent/100)
	}
	createdProduct, err := s.productRepo.Create(ctx, product)
	if err != nil {
		return nil, err
	}

	return dto.FromProduct(createdProduct), nil
}

func (s *productService) UpdateProduct(ctx context.Context, productID string, req dto.UpdateProductRequest) (*dto.ProductResponse, error) {

	product, err := s.productRepo.GetByID(ctx, productID)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, apperror.ErrProductNotFound
		}
		return nil, err
	}

	var oldImagePublicIDs []string

	// Update fields if provided
	if req.Name != nil {
		product.Name = *req.Name
	}
	if req.Description != nil {
		product.Description = *req.Description
	}
	if req.Brand != nil {
		product.Brand = *req.Brand
	}
	if req.CategoryID != nil {
		categoryObjID, err := primitive.ObjectIDFromHex(*req.CategoryID)
		if err != nil {
			return nil, apperror.ErrInvalidID
		}
		product.CategoryID = categoryObjID
	}
	if req.Thumbnail != nil {
		oldImagePublicIDs = append(oldImagePublicIDs, product.Thumbnail.PublicID)
		product.Thumbnail = *req.Thumbnail
	}
	if req.BasePrice != nil {
		product.BasePrice = *req.BasePrice
	}
	if req.DiscountPercent != nil {
		product.DiscountPercent = *req.DiscountPercent
	}
	if req.BasePrice != nil || req.DiscountPercent != nil {
		for i := range product.Variants {
			product.Variants[i].FinalPrice = (product.BasePrice + product.Variants[i].PriceAdjustment) * (1 - product.DiscountPercent/100)
		}
	}
	if req.StockQuantity != nil {
		product.StockQuantity = req.StockQuantity
	}
	if req.Status != nil {
		product.Status = *req.Status
	}

	product.UpdatedAt = time.Now()

	updatedProduct, err := s.productRepo.Update(ctx, product)
	if err != nil {
		return nil, err
	}

	// Delete old images async
	for _, publicID := range oldImagePublicIDs {
		if publicID != "" {
			go cloudinary.Delete(publicID)
		}
	}

	return dto.FromProduct(updatedProduct), nil
}

func (s *productService) DeleteProduct(ctx context.Context, productID string) error {
	product, err := s.productRepo.GetByID(ctx, productID)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return apperror.ErrProductNotFound
		}
		return err
	}

	err = s.productRepo.Delete(ctx, productID)
	if err != nil {
		return err
	}

	// Delete images async
	go func() {
		cloudinary.Delete(product.Thumbnail.PublicID)
		for _, img := range product.Images {
			cloudinary.Delete(img.PublicID)
		}
	}()

	return nil
}

func (s *productService) SoftDeleteProduct(ctx context.Context, productID string) error {
	_, err := s.productRepo.GetByID(ctx, productID)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return apperror.ErrProductNotFound
		}
		return err
	}

	return s.productRepo.SoftDelete(ctx, productID)
}

func (s *productService) GetProductByID(ctx context.Context, productID string) (*dto.ProductResponse, error) {
	product, err := s.productRepo.GetByID(ctx, productID)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, apperror.ErrProductNotFound
		}
		return nil, err
	}

	return dto.FromProduct(product), nil
}

func (s *productService) FindProducts(ctx context.Context, filter repo.Filter, opts *repo.FindOptions) ([]*dto.ProductResponse, int64, error) {
	products, total, err := s.productRepo.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, err
	}

	return dto.FromProducts(products), total, nil
}

func (s *productService) AddProductImages(ctx context.Context, productID string, images []model.Image) error {
	return s.productRepo.AddImages(ctx, productID, images)
}

func (s *productService) AddProductVideos(ctx context.Context, productID string, videos []model.Video) error {
	return s.productRepo.AddVideos(ctx, productID, videos)
}

func (s *productService) CreateVariant(ctx context.Context, productID string, req dto.CreateVariantRequest) (*dto.VariantResponse, error) {
	product, err := s.productRepo.GetByID(ctx, productID)
	if err != nil {
		return nil, err
	}

	variant := model.ProductVariant{
		Title:           req.Title,
		Description:     req.Description,
		PriceAdjustment: req.PriceAdjustment,
		FinalPrice:      (product.BasePrice + req.PriceAdjustment) * (1 - product.DiscountPercent/100),
		StockQuantity:   req.StockQuantity,
	}

	createdVariant, err := s.productRepo.CreateVariant(ctx, productID, variant)
	if err != nil {
		return nil, err
	}

	return dto.FromVariant(createdVariant), nil
}

func (s *productService) UpdateVariant(ctx context.Context, productID string, variantID string, req dto.UpdateVariantRequest) (*dto.VariantResponse, error) {
	variantObjID, err := primitive.ObjectIDFromHex(variantID)
	if err != nil {
		return nil, apperror.ErrInvalidID
	}

	variant := model.ProductVariant{
		ID: variantObjID,
	}
	product, err := s.productRepo.GetByID(ctx, productID)
	if err != nil {
		return nil, err
	}

	if req.Title != nil {
		variant.Title = *req.Title
	}
	if req.Description != nil {
		variant.Description = *req.Description
	}
	if req.PriceAdjustment != nil {
		variant.PriceAdjustment = *req.PriceAdjustment
	}
	if req.StockQuantity != nil {
		variant.StockQuantity = *req.StockQuantity
	}
	variant.FinalPrice = (product.BasePrice + variant.PriceAdjustment) * (1 - product.DiscountPercent/100)
	updatedVariant, err := s.productRepo.UpdateVariant(ctx, productID, variant)
	if err != nil {
		return nil, err
	}

	return dto.FromVariant(updatedVariant), nil
}

func (s *productService) CheckAvailability(ctx context.Context, productID string, variantID *string, qty int) (bool, int, error) {
	return s.productRepo.CheckAvailability(ctx, productID, variantID, qty)
}

func (s *productService) IncrementSoldCount(ctx context.Context, productID string, count int) error {
	return s.productRepo.IncrementSoldCount(ctx, productID, count)
}

func (s *productService) UpdateRating(ctx context.Context, productID string, newRating float64) error {
	product, err := s.productRepo.GetByID(ctx, productID)
	if err != nil {
		return err
	}

	// Calculate new average rating
	newRatingCount := product.RatingCount + 1
	averageRating := (product.Rating*float64(product.RatingCount) + newRating) / float64(newRatingCount)

	return s.productRepo.UpdateRating(ctx, productID, averageRating, newRatingCount)
}
