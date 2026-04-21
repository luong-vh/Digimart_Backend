package service

import (
	"errors"

	"github.com/DucLove1/SE357-ShoppingManagement-BE/internal/apperror"
	"github.com/DucLove1/SE357-ShoppingManagement-BE/internal/dto"
	"github.com/DucLove1/SE357-ShoppingManagement-BE/internal/model"
	"github.com/DucLove1/SE357-ShoppingManagement-BE/internal/platform/bus"
	"github.com/DucLove1/SE357-ShoppingManagement-BE/internal/repo"
	"github.com/DucLove1/SE357-ShoppingManagement-BE/internal/util"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type CartService interface {
	// Cart operations
	GetCart(userID string) (*dto.CartResponse, error)
	GetCartWithRefresh(userID string) (*dto.CartResponse, error)
	ClearCart(userID string) error

	// Item operations
	AddItem(userID string, req *dto.AddCartItemRequest) (*dto.CartResponse, error)
	UpdateItemQuantity(userID string, req *dto.UpdateCartItemRequest) (*dto.CartResponse, error)
	RemoveItem(userID string, productID string, variantID *string) (*dto.CartResponse, error)

	// Batch operations
	AddItems(userID string, req *dto.AddCartItemsRequest) (*dto.CartResponse, error)
	RemoveItems(userID string, req *dto.RemoveCartItemsRequest) (*dto.CartResponse, error)

	// Validation
	ValidateCart(userID string) (*dto.CartValidationResponse, error)

	// Stats
	GetCartStats() (*dto.CartStatsResponse, error)
}

type cartService struct {
	cartRepo    repo.CartRepo
	productRepo repo.ProductRepo
	userRepo    repo.UserRepo
	eventBus    bus.EventBus
	redisClient *redis.Client
}

func NewCartService(cartRepo repo.CartRepo, productRepo repo.ProductRepo, userRepo repo.UserRepo, eventBus bus.EventBus, redisClient *redis.Client) CartService {
	return &cartService{
		cartRepo:    cartRepo,
		productRepo: productRepo,
		userRepo:    userRepo,
		eventBus:    eventBus,
		redisClient: redisClient,
	}
}

func (s *cartService) GetCart(userID string) (*dto.CartResponse, error) {
	ctx, cancel := util.NewDefaultDBContext()
	defer cancel()

	cart, err := s.cartRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	return dto.FromCart(cart), nil
}

func (s *cartService) GetCartWithRefresh(userID string) (*dto.CartResponse, error) {
	ctx, cancel := util.NewDefaultDBContext()
	defer cancel()

	cart, err := s.cartRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Refresh snapshots for all items
	for i := range cart.Items {
		snapshot, err := s.buildCartItemSnapshot(&cart.Items[i])
		if err != nil {
			// Mark as unavailable if product not found
			if cart.Items[i].Snapshot != nil {
				cart.Items[i].Snapshot.IsAvailable = false
			}
			continue
		}
		cart.Items[i].Snapshot = snapshot

		// Update snapshot in database
		var variantIDStr *string
		if cart.Items[i].VariantID != nil {
			v := cart.Items[i].VariantID.Hex()
			variantIDStr = &v
		}
		_ = s.cartRepo.UpdateItemSnapshot(ctx, userID, cart.Items[i].ProductID.Hex(), variantIDStr, snapshot)
	}

	return dto.FromCart(cart), nil
}

func (s *cartService) ClearCart(userID string) error {
	ctx, cancel := util.NewDefaultDBContext()
	defer cancel()

	err := s.cartRepo.ClearCart(ctx, userID)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return apperror.ErrCartNotFound
		}
		return err
	}

	//// Publish event
	//s.eventBus.Publish(bus.CartClearedEventType{
	//	UserID: userID,
	//})

	return nil
}

func (s *cartService) AddItem(userID string, req *dto.AddCartItemRequest) (*dto.CartResponse, error) {
	ctx, cancel := util.NewDefaultDBContext()
	defer cancel()

	// Validate product exists and get info
	product, err := s.productRepo.GetByID(ctx, req.ProductID)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, apperror.ErrProductNotFound
		}
		return nil, err
	}

	// Check if product is available
	if product.Status != model.ProductStatusActive {
		return nil, apperror.ErrProductNotAvailable
	}

	productObjID, _ := primitive.ObjectIDFromHex(req.ProductID)

	var variantObjID *primitive.ObjectID
	var variant *model.ProductVariant

	// If product has variants, variant ID is required
	if product.HasVariants {
		if req.VariantID == nil {
			return nil, apperror.ErrVariantRequired
		}
		vID, err := primitive.ObjectIDFromHex(*req.VariantID)
		if err != nil {
			return nil, apperror.ErrInvalidID
		}
		variantObjID = &vID

		// Find variant
		for i := range product.Variants {
			if product.Variants[i].ID == vID {
				variant = &product.Variants[i]
				break
			}
		}
		if variant == nil {
			return nil, apperror.ErrVariantNotFound
		}

		// Check variant stock
		if variant.StockQuantity < req.Quantity {
			return nil, apperror.ErrInsufficientStock
		}
	} else {
		// Check product stock
		if product.StockQuantity < req.Quantity {
			return nil, apperror.ErrInsufficientStock
		}
	}

	// Build snapshot
	cartItem := model.CartItem{
		ProductID: productObjID,
		VariantID: variantObjID,
		SellerID:  product.SellerID,
		Quantity:  req.Quantity,
	}

	snapshot, err := s.buildCartItemSnapshotFromProduct(product, variant)
	if err != nil {
		return nil, err
	}
	cartItem.Snapshot = snapshot

	// Add item to cart
	err = s.cartRepo.AddItem(ctx, userID, cartItem)
	if err != nil {
		return nil, err
	}

	//// Publish event
	//s.eventBus.Publish(bus.CartItemAddedEventType{
	//	UserID:    userID,
	//	ProductID: req.ProductID,
	//	VariantID: req.VariantID,
	//	Quantity:  req.Quantity,
	//})

	return s.GetCart(userID)
}

func (s *cartService) UpdateItemQuantity(userID string, req *dto.UpdateCartItemRequest) (*dto.CartResponse, error) {
	ctx, cancel := util.NewDefaultDBContext()
	defer cancel()

	if req.Quantity <= 0 {
		return s.RemoveItem(userID, req.ProductID, req.VariantID)
	}

	// Validate stock
	product, err := s.productRepo.GetByID(ctx, req.ProductID)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, apperror.ErrProductNotFound
		}
		return nil, err
	}

	if product.HasVariants && req.VariantID != nil {
		variantObjID, _ := primitive.ObjectIDFromHex(*req.VariantID)
		for _, v := range product.Variants {
			if v.ID == variantObjID {
				if v.StockQuantity < req.Quantity {
					return nil, apperror.ErrInsufficientStock
				}
				break
			}
		}
	} else {
		if product.StockQuantity < req.Quantity {
			return nil, apperror.ErrInsufficientStock
		}
	}

	err = s.cartRepo.UpdateItemQuantity(ctx, userID, req.ProductID, req.VariantID, req.Quantity)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, apperror.ErrCartItemNotFound
		}
		return nil, err
	}

	//// Publish event
	//s.eventBus.Publish(bus.CartItemUpdatedEventType{
	//	UserID:    userID,
	//	ProductID: req.ProductID,
	//	VariantID: req.VariantID,
	//	Quantity:  req.Quantity,
	//})

	return s.GetCart(userID)
}

func (s *cartService) RemoveItem(userID string, productID string, variantID *string) (*dto.CartResponse, error) {
	ctx, cancel := util.NewDefaultDBContext()
	defer cancel()

	err := s.cartRepo.RemoveItem(ctx, userID, productID, variantID)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, apperror.ErrCartItemNotFound
		}
		return nil, err
	}

	// Publish event
	//s.eventBus.Publish(bus.CartItemRemovedEventType{
	//	UserID:    userID,
	//	ProductID: productID,
	//	VariantID: variantID,
	//})

	return s.GetCart(userID)
}

func (s *cartService) AddItems(userID string, req *dto.AddCartItemsRequest) (*dto.CartResponse, error) {
	for _, item := range req.Items {
		_, err := s.AddItem(userID, &item)
		if err != nil {
			return nil, err
		}
	}

	return s.GetCart(userID)
}

func (s *cartService) RemoveItems(userID string, req *dto.RemoveCartItemsRequest) (*dto.CartResponse, error) {
	for _, item := range req.Items {
		_, err := s.RemoveItem(userID, item.ProductID, item.VariantID)
		if err != nil && !errors.Is(err, apperror.ErrCartItemNotFound) {
			return nil, err
		}
	}

	return s.GetCart(userID)
}

func (s *cartService) ValidateCart(userID string) (*dto.CartValidationResponse, error) {
	ctx, cancel := util.NewDefaultDBContext()
	defer cancel()

	cart, err := s.cartRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	response := &dto.CartValidationResponse{
		IsValid:       true,
		InvalidItems:  []dto.InvalidCartItem{},
		ValidItems:    []dto.ValidCartItem{},
		TotalAmount:   0,
		TotalQuantity: 0,
	}

	for _, item := range cart.Items {
		product, err := s.productRepo.GetByID(ctx, item.ProductID.Hex())
		if err != nil {
			response.IsValid = false
			response.InvalidItems = append(response.InvalidItems, dto.InvalidCartItem{
				ProductID: item.ProductID.Hex(),
				VariantID: func() *string {
					if item.VariantID != nil {
						v := item.VariantID.Hex()
						return &v
					}
					return nil
				}(),
				Reason: "Product not found",
			})
			continue
		}

		// Check product status
		if product.Status != model.ProductStatusActive {
			response.IsValid = false
			response.InvalidItems = append(response.InvalidItems, dto.InvalidCartItem{
				ProductID: item.ProductID.Hex(),
				Reason:    "Product not available",
			})
			continue
		}

		var availableStock int
		var price float64

		if product.HasVariants && item.VariantID != nil {
			variantFound := false
			for _, v := range product.Variants {
				if v.ID == *item.VariantID {
					variantFound = true
					availableStock = v.StockQuantity
					price = v.GetEffectivePrice()
					break
				}
			}
			if !variantFound {
				response.IsValid = false
				response.InvalidItems = append(response.InvalidItems, dto.InvalidCartItem{
					ProductID: item.ProductID.Hex(),
					VariantID: func() *string { v := item.VariantID.Hex(); return &v }(),
					Reason:    "Variant not found",
				})
				continue
			}
		} else {
			availableStock = product.StockQuantity
			price = product.GetEffectivePrice()
		}

		// Check stock
		if item.Quantity > availableStock {
			response.IsValid = false
			response.InvalidItems = append(response.InvalidItems, dto.InvalidCartItem{
				ProductID: item.ProductID.Hex(),
				VariantID: func() *string {
					if item.VariantID != nil {
						v := item.VariantID.Hex()
						return &v
					}
					return nil
				}(),
				Reason:         "Insufficient stock",
				AvailableStock: availableStock,
				RequestedQty:   item.Quantity,
			})
			continue
		}

		// Item is valid
		response.ValidItems = append(response.ValidItems, dto.ValidCartItem{
			ProductID: item.ProductID.Hex(),
			VariantID: func() *string {
				if item.VariantID != nil {
					v := item.VariantID.Hex()
					return &v
				}
				return nil
			}(),
			Quantity:    item.Quantity,
			Price:       price,
			TotalAmount: price * float64(item.Quantity),
		})

		response.TotalAmount += price * float64(item.Quantity)
		response.TotalQuantity += item.Quantity
	}

	return response, nil
}

func (s *cartService) GetCartStats() (*dto.CartStatsResponse, error) {
	ctx, cancel := util.NewDefaultDBContext()
	defer cancel()

	total, err := s.cartRepo.CountTotal(ctx)
	if err != nil {
		return nil, err
	}

	nonEmpty, err := s.cartRepo.CountNonEmpty(ctx)
	if err != nil {
		return nil, err
	}

	avgItems, err := s.cartRepo.GetAverageItemCount(ctx)
	if err != nil {
		return nil, err
	}

	return &dto.CartStatsResponse{
		TotalCarts:       total,
		NonEmptyCarts:    nonEmpty,
		AverageItemCount: avgItems,
	}, nil
}

// Helper methods

func (s *cartService) buildCartItemSnapshot(item *model.CartItem) (*model.CartItemSnapshot, error) {
	ctx, cancel := util.NewDefaultDBContext()
	defer cancel()

	product, err := s.productRepo.GetByID(ctx, item.ProductID.Hex())
	if err != nil {
		return nil, err
	}

	var variant *model.ProductVariant
	if item.VariantID != nil {
		for i := range product.Variants {
			if product.Variants[i].ID == *item.VariantID {
				variant = &product.Variants[i]
				break
			}
		}
	}

	return s.buildCartItemSnapshotFromProduct(product, variant)
}

func (s *cartService) buildCartItemSnapshotFromProduct(product *model.Product, variant *model.ProductVariant) (*model.CartItemSnapshot, error) {
	ctx, cancel := util.NewDefaultDBContext()
	defer cancel()

	// Get seller info
	seller, err := s.userRepo.GetByID(ctx, product.SellerID.Hex())
	if err != nil {
		return nil, err
	}

	var sellerName string
	if seller.RoleContent.Seller != nil {
		sellerName = seller.FullName
	}

	snapshot := &model.CartItemSnapshot{
		ProductName: product.Name,
		SellerName:  sellerName,
		Image:       product.Thumbnail,
		IsAvailable: product.Status == model.ProductStatusActive,
	}

	if variant != nil {
		snapshot.SKU = variant.SKU
		snapshot.Price = variant.Price
		snapshot.SalePrice = variant.SalePrice
		snapshot.Stock = variant.StockQuantity
		snapshot.Attributes = variant.Attributes
		if variant.Image != nil {
			snapshot.Image = *variant.Image
		}
		snapshot.IsAvailable = snapshot.IsAvailable && variant.Status == model.ProductStatusActive
	} else {
		snapshot.SKU = product.SKU
		snapshot.Price = product.Price
		snapshot.SalePrice = product.SalePrice
		snapshot.Stock = product.StockQuantity
	}

	return snapshot, nil
}
