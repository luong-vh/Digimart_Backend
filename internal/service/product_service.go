package service

import (
	"errors"
	"fmt"
	"time"

	"github.com/DucLove1/SE357-ShoppingManagement-BE/internal/apperror"
	"github.com/DucLove1/SE357-ShoppingManagement-BE/internal/dto"
	"github.com/DucLove1/SE357-ShoppingManagement-BE/internal/model"
	"github.com/DucLove1/SE357-ShoppingManagement-BE/internal/platform/bus"
	"github.com/DucLove1/SE357-ShoppingManagement-BE/internal/platform/cloudinary"
	"github.com/DucLove1/SE357-ShoppingManagement-BE/internal/repo"
	"github.com/DucLove1/SE357-ShoppingManagement-BE/internal/util"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type ProductService interface {
	// CRUD operations
	CreateProduct(sellerID string, req *dto.CreateProductRequest) (*dto.ProductResponse, error)
	UpdateProduct(productID string, sellerID string, req *dto.UpdateProductRequest) (*dto.ProductResponse, error)
	DeleteProduct(productID string, sellerID string) error
	GetProductByID(id string) (*dto.ProductResponse, error)
	GetProductBySlug(slug string) (*dto.ProductResponse, error)

	// Query operations
	GetProductsBySellerID(sellerID string) ([]*dto.ProductResponse, error)
	GetProductsByCategoryID(categoryID string) ([]*dto.ProductResponse, error)
	FindProducts(filter repo.Filter, opts *repo.FindOptions) ([]*dto.ProductResponse, int64, error)

	// Variant operations
	AddVariant(productID string, sellerID string, req *dto.AddVariantRequest) (*dto.ProductResponse, error)
	UpdateVariant(productID string, variantID string, sellerID string, req *dto.UpdateVariantRequest) (*dto.ProductResponse, error)
	DeleteVariant(productID string, variantID string, sellerID string) (*dto.ProductResponse, error)

	// Status operations
	UpdateProductStatus(productID string, sellerID string, status model.ProductStatus) (*dto.ProductResponse, error)

	// Inventory operations
	UpdateStock(productID string, sellerID string, quantity int) error
	UpdateVariantStock(productID string, variantID string, sellerID string, quantity int) error

	// Stats operations
	IncrementViewCount(productID string) error
	GetProductStats() (*dto.ProductStatsResponse, error)
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

func (s *productService) CreateProduct(sellerID string, req *dto.CreateProductRequest) (*dto.ProductResponse, error) {
	ctx, cancel := util.NewDefaultDBContext()
	defer cancel()

	// Verify seller exists
	_, err := s.userRepo.GetByID(ctx, sellerID)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, apperror.ErrUserNotFound
		}
		return nil, err
	}

	sellerObjID, err := primitive.ObjectIDFromHex(sellerID)
	if err != nil {
		return nil, apperror.ErrInvalidID
	}

	categoryObjID, err := primitive.ObjectIDFromHex(req.CategoryID)
	if err != nil {
		return nil, apperror.ErrInvalidID
	}

	// Generate slug from name
	slug := util.GenerateSlug(req.Name)

	// Check if slug already exists
	existingProduct, _ := s.productRepo.GetBySlug(ctx, slug)
	if existingProduct != nil {
		slug = slug + "-" + primitive.NewObjectID().Hex()[:8]
	}

	product := &model.Product{
		CategoryID:    categoryObjID,
		SellerID:      sellerObjID,
		Name:          req.Name,
		Slug:          slug,
		Description:   req.Description,
		SKU:           req.SKU,
		Thumbnail:     req.Thumbnail,
		Images:        req.Images,
		Videos:        req.Videos,
		Price:         req.Price,
		SalePrice:     req.SalePrice,
		StockQuantity: req.StockQuantity,
		HasVariants:   req.HasVariants,
		Attributes:    req.Attributes,
		Variants:      req.Variants,
		Status:        model.ProductStatusDraft,
		SoldCount:     0,
		Rating:        0,
		RatingCount:   0,
		ViewCount:     0,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	// Generate IDs for variants
	for i := range product.Variants {
		product.Variants[i].ID = primitive.NewObjectID()
		if product.Variants[i].Status == "" {
			product.Variants[i].Status = model.ProductStatusActive
		}
	}

	createdProduct, err := s.productRepo.Create(ctx, product)
	if err != nil {
		return nil, err
	}

	//// Publish event
	//s.eventBus.Publish(bus.ProductCreatedEventType{
	//	ProductID: createdProduct.ID.Hex(),
	//	SellerID:  sellerID,
	//	Name:      createdProduct.Name,
	//})

	return dto.FromProduct(createdProduct), nil
}

func (s *productService) UpdateProduct(productID string, sellerID string, req *dto.UpdateProductRequest) (*dto.ProductResponse, error) {
	ctx, cancel := util.NewDefaultDBContext()
	defer cancel()

	product, err := s.productRepo.GetByID(ctx, productID)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, apperror.ErrProductNotFound
		}
		return nil, err
	}

	// Verify ownership
	if product.SellerID.Hex() != sellerID {
		return nil, apperror.ErrForbidden
	}

	var oldImagePublicIDs []string

	// Update fields if provided
	if req.Name != nil {
		product.Name = *req.Name
		// Optionally update slug
		newSlug := util.GenerateSlug(*req.Name)
		existingProduct, _ := s.productRepo.GetBySlug(ctx, newSlug)
		if existingProduct == nil || existingProduct.ID == product.ID {
			product.Slug = newSlug
		}
	}

	if req.Description != nil {
		product.Description = *req.Description
	}

	if req.SKU != nil {
		product.SKU = *req.SKU
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

	if req.Images != nil {
		for _, img := range product.Images {
			oldImagePublicIDs = append(oldImagePublicIDs, img.PublicID)
		}
		product.Images = req.Images
	}

	if req.Videos != nil {
		product.Videos = req.Videos
	}

	if req.Price != nil {
		product.Price = *req.Price
	}

	if req.SalePrice != nil {
		product.SalePrice = req.SalePrice
	}

	if req.StockQuantity != nil {
		product.StockQuantity = *req.StockQuantity
	}

	if req.Attributes != nil {
		product.Attributes = req.Attributes
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

	//// Publish event
	//s.eventBus.Publish(bus.ProductUpdatedEventType{
	//	ProductID: productID,
	//	SellerID:  sellerID,
	//})

	return dto.FromProduct(updatedProduct), nil
}

func (s *productService) DeleteProduct(productID string, sellerID string) error {
	ctx, cancel := util.NewDefaultDBContext()
	defer cancel()

	product, err := s.productRepo.GetByID(ctx, productID)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return apperror.ErrProductNotFound
		}
		return err
	}

	// Verify ownership
	if product.SellerID.Hex() != sellerID {
		return apperror.ErrForbidden
	}

	err = s.productRepo.SoftDelete(ctx, productID)
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

	//// Publish event
	//s.eventBus.Publish(bus.ProductDeletedEventType{
	//	ProductID: productID,
	//	SellerID:  sellerID,
	//})

	return nil
}

func (s *productService) GetProductByID(id string) (*dto.ProductResponse, error) {
	ctx, cancel := util.NewDefaultDBContext()
	defer cancel()

	product, err := s.productRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, apperror.ErrProductNotFound
		}
		return nil, err
	}

	return dto.FromProduct(product), nil
}

func (s *productService) GetProductBySlug(slug string) (*dto.ProductResponse, error) {
	ctx, cancel := util.NewDefaultDBContext()
	defer cancel()
	product, err := s.productRepo.GetBySlug(ctx, slug)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		fmt.Printf("Error string: %q\n", err.Error()) // In ra với quotes để thấy exact string
		fmt.Printf("Is ErrNoDocuments: %v\n", errors.Is(err, mongo.ErrNoDocuments))
		if errors.Is(err, mongo.ErrNoDocuments) || err.Error() == "mongo: no documents in result" {
			return nil, apperror.ErrProductNotFound
		}
		return nil, err
	}
	return dto.FromProduct(product), nil
}

func (s *productService) GetProductsBySellerID(sellerID string) ([]*dto.ProductResponse, error) {
	ctx, cancel := util.NewDefaultDBContext()
	defer cancel()

	products, err := s.productRepo.GetBySellerID(ctx, sellerID)
	if err != nil {
		return nil, err
	}

	return dto.FromProducts(products), nil
}

func (s *productService) GetProductsByCategoryID(categoryID string) ([]*dto.ProductResponse, error) {
	ctx, cancel := util.NewDefaultDBContext()
	defer cancel()

	products, err := s.productRepo.GetByCategoryID(ctx, categoryID)
	if err != nil {
		return nil, err
	}

	return dto.FromProducts(products), nil
}

func (s *productService) FindProducts(filter repo.Filter, opts *repo.FindOptions) ([]*dto.ProductResponse, int64, error) {
	ctx, cancel := util.NewDefaultDBContext()
	defer cancel()

	products, total, err := s.productRepo.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, err
	}

	return dto.FromProducts(products), total, nil
}

func (s *productService) AddVariant(productID string, sellerID string, req *dto.AddVariantRequest) (*dto.ProductResponse, error) {
	ctx, cancel := util.NewDefaultDBContext()
	defer cancel()

	product, err := s.productRepo.GetByID(ctx, productID)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, apperror.ErrProductNotFound
		}
		return nil, err
	}

	// Verify ownership
	if product.SellerID.Hex() != sellerID {
		return nil, apperror.ErrForbidden
	}

	newVariant := model.ProductVariant{
		ID:            primitive.NewObjectID(),
		SKU:           req.SKU,
		Attributes:    req.Attributes,
		Price:         req.Price,
		SalePrice:     req.SalePrice,
		StockQuantity: req.StockQuantity,
		SoldCount:     0,
		Image:         req.Image,
		Status:        model.ProductStatusActive,
	}

	product.Variants = append(product.Variants, newVariant)
	product.HasVariants = true
	product.UpdatedAt = time.Now()

	updatedProduct, err := s.productRepo.Update(ctx, product)
	if err != nil {
		return nil, err
	}

	return dto.FromProduct(updatedProduct), nil
}

func (s *productService) UpdateVariant(productID string, variantID string, sellerID string, req *dto.UpdateVariantRequest) (*dto.ProductResponse, error) {
	ctx, cancel := util.NewDefaultDBContext()
	defer cancel()

	product, err := s.productRepo.GetByID(ctx, productID)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, apperror.ErrProductNotFound
		}
		return nil, err
	}

	// Verify ownership
	if product.SellerID.Hex() != sellerID {
		return nil, apperror.ErrForbidden
	}

	variantObjID, err := primitive.ObjectIDFromHex(variantID)
	if err != nil {
		return nil, apperror.ErrInvalidID
	}

	// Find and update variant
	variantFound := false
	for i := range product.Variants {
		if product.Variants[i].ID == variantObjID {
			variantFound = true

			if req.SKU != nil {
				product.Variants[i].SKU = *req.SKU
			}
			if req.Attributes != nil {
				product.Variants[i].Attributes = req.Attributes
			}
			if req.Price != nil {
				product.Variants[i].Price = *req.Price
			}
			if req.SalePrice != nil {
				product.Variants[i].SalePrice = req.SalePrice
			}
			if req.StockQuantity != nil {
				product.Variants[i].StockQuantity = *req.StockQuantity
			}
			if req.Image != nil {
				if product.Variants[i].Image != nil {
					go cloudinary.Delete(product.Variants[i].Image.PublicID)
				}
				product.Variants[i].Image = req.Image
			}
			if req.Status != nil {
				product.Variants[i].Status = *req.Status
			}
			break
		}
	}

	if !variantFound {
		return nil, apperror.ErrVariantNotFound
	}

	product.UpdatedAt = time.Now()

	updatedProduct, err := s.productRepo.Update(ctx, product)
	if err != nil {
		return nil, err
	}

	return dto.FromProduct(updatedProduct), nil
}

func (s *productService) DeleteVariant(productID string, variantID string, sellerID string) (*dto.ProductResponse, error) {
	ctx, cancel := util.NewDefaultDBContext()
	defer cancel()

	product, err := s.productRepo.GetByID(ctx, productID)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, apperror.ErrProductNotFound
		}
		return nil, err
	}

	// Verify ownership
	if product.SellerID.Hex() != sellerID {
		return nil, apperror.ErrForbidden
	}

	variantObjID, err := primitive.ObjectIDFromHex(variantID)
	if err != nil {
		return nil, apperror.ErrInvalidID
	}

	// Find and remove variant
	variantFound := false
	newVariants := make([]model.ProductVariant, 0)
	for _, v := range product.Variants {
		if v.ID == variantObjID {
			variantFound = true
			// Delete variant image async
			if v.Image != nil {
				go cloudinary.Delete(v.Image.PublicID)
			}
			continue
		}
		newVariants = append(newVariants, v)
	}

	if !variantFound {
		return nil, apperror.ErrVariantNotFound
	}

	product.Variants = newVariants
	if len(product.Variants) == 0 {
		product.HasVariants = false
	}
	product.UpdatedAt = time.Now()

	updatedProduct, err := s.productRepo.Update(ctx, product)
	if err != nil {
		return nil, err
	}

	return dto.FromProduct(updatedProduct), nil
}

func (s *productService) UpdateProductStatus(productID string, sellerID string, status model.ProductStatus) (*dto.ProductResponse, error) {
	ctx, cancel := util.NewDefaultDBContext()
	defer cancel()

	product, err := s.productRepo.GetByID(ctx, productID)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, apperror.ErrProductNotFound
		}
		return nil, err
	}

	// Verify ownership
	if product.SellerID.Hex() != sellerID {
		return nil, apperror.ErrForbidden
	}

	product.Status = status
	product.UpdatedAt = time.Now()

	updatedProduct, err := s.productRepo.Update(ctx, product)
	if err != nil {
		return nil, err
	}

	//// Publish event
	//s.eventBus.Publish(bus.ProductStatusChangedEventType{
	//	ProductID: productID,
	//	SellerID:  sellerID,
	//	Status:    string(status),
	//})

	return dto.FromProduct(updatedProduct), nil
}

func (s *productService) UpdateStock(productID string, sellerID string, quantity int) error {
	ctx, cancel := util.NewDefaultDBContext()
	defer cancel()

	product, err := s.productRepo.GetByID(ctx, productID)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return apperror.ErrProductNotFound
		}
		return err
	}

	// Verify ownership
	if product.SellerID.Hex() != sellerID {
		return apperror.ErrForbidden
	}

	return s.productRepo.UpdateStock(ctx, productID, quantity)
}

func (s *productService) UpdateVariantStock(productID string, variantID string, sellerID string, quantity int) error {
	ctx, cancel := util.NewDefaultDBContext()
	defer cancel()

	product, err := s.productRepo.GetByID(ctx, productID)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return apperror.ErrProductNotFound
		}
		return err
	}

	// Verify ownership
	if product.SellerID.Hex() != sellerID {
		return apperror.ErrForbidden
	}

	return s.productRepo.UpdateVariantStock(ctx, productID, variantID, quantity)
}

func (s *productService) IncrementViewCount(productID string) error {
	ctx, cancel := util.NewDefaultDBContext()
	defer cancel()

	return s.productRepo.IncrementViewCount(ctx, productID)
}

func (s *productService) GetProductStats() (*dto.ProductStatsResponse, error) {
	ctx, cancel := util.NewDefaultDBContext()
	defer cancel()

	total, err := s.productRepo.CountTotal(ctx)
	if err != nil {
		return nil, err
	}

	active, err := s.productRepo.CountByStatus(ctx, model.ProductStatusActive)
	if err != nil {
		return nil, err
	}

	outOfStock, err := s.productRepo.CountOutOfStock(ctx)
	if err != nil {
		return nil, err
	}

	newThisWeek, err := s.productRepo.CountCreatedAfter(ctx, time.Now().AddDate(0, 0, -7))
	if err != nil {
		return nil, err
	}

	return &dto.ProductStatsResponse{
		Total:       total,
		Active:      active,
		OutOfStock:  outOfStock,
		NewThisWeek: newThisWeek,
	}, nil
}
