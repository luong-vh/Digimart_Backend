package service

import (
	"context"
	"errors"
	"time"

	"github.com/luong-vh/Digimart_Backend/internal/apperror"
	"github.com/luong-vh/Digimart_Backend/internal/dto"
	"github.com/luong-vh/Digimart_Backend/internal/model"
	"github.com/luong-vh/Digimart_Backend/internal/platform/bus"
	"github.com/luong-vh/Digimart_Backend/internal/repo"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type CartService interface {
	// Cart operations
	GetCart(ctx context.Context, userID string) (*dto.CartResponse, error)
	ClearCart(ctx context.Context, userID string) error
	GetItemCount(ctx context.Context, userID string) (int, error)

	// Item operations
	AddItem(ctx context.Context, userID string, req dto.AddCartItemRequest) (*dto.CartResponse, error)
	UpdateItemQuantity(ctx context.Context, userID string, req dto.UpdateCartItemRequest) (*dto.CartResponse, error)
	RemoveItem(ctx context.Context, userID string, itemID string) (*dto.CartResponse, error)

	// Batch operations
	AddItems(ctx context.Context, userID string, req dto.AddCartItemsRequest) (*dto.CartResponse, error)
	RemoveItems(ctx context.Context, userID string, req dto.RemoveCartItemsRequest) (*dto.CartResponse, error)
	GetSnapshot(ctx context.Context, productID string, variantID *string) (*model.CartItemSnapshot, error)
	// Checkout support
	ValidateCart(ctx context.Context, userID string) (*dto.CartValidationResult, error)
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

func (s *cartService) GetCart(ctx context.Context, userID string) (*dto.CartResponse, error) {
	cart, err := s.cartRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	items := make([]dto.CartItemResponse, 0, len(cart.Items))
	var totalAmount float64
	var totalQuantity int

	for _, item := range cart.Items {
		resp := dto.CartItemResponse{
			ID:        item.ID.Hex(),
			ProductID: item.ProductID.Hex(),
			Quantity:  item.Quantity,
			AddedAt:   item.AddedAt.Format("2006-01-02T15:04:05Z07:00"),
		}

		var variantIDStr *string
		if item.VariantID != nil {
			v := item.VariantID.Hex()
			resp.VariantID = &v
			variantIDStr = &v
		}

		snapshot, err := s.GetSnapshot(ctx, item.ProductID.Hex(), variantIDStr)
		if err != nil {
			resp.IsAvailable = false
			items = append(items, resp)
			continue
		}

		resp.Snapshot = snapshot
		resp.IsAvailable = snapshot.IsAvailable

		if snapshot.IsAvailable {
			totalAmount += snapshot.FinalPrice * float64(item.Quantity)
			totalQuantity += item.Quantity
		}

		items = append(items, resp)
	}

	return &dto.CartResponse{
		ID:            cart.ID.Hex(),
		UserID:        cart.UserID.Hex(),
		Items:         items,
		TotalAmount:   totalAmount,
		TotalQuantity: totalQuantity,
		CreatedAt:     cart.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:     cart.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}, nil
}

func (s *cartService) GetSnapshot(ctx context.Context, productID string, variantID *string) (*model.CartItemSnapshot, error) {
	variantIDStr := ""
	if variantID != nil {
		variantIDStr = *variantID
	}

	product, variant, err := s.productRepo.GetProductWithVariant(ctx, productID, variantIDStr)
	if err != nil {
		return nil, err
	}

	var basePrice float64
	var finalPrice float64
	var priceAdjustment float64
	var stockQuantity int
	var variantTitle *string
	var variantObjID *primitive.ObjectID

	if variant != nil {
		basePrice = product.BasePrice
		priceAdjustment = variant.PriceAdjustment
		finalPrice = variant.FinalPrice
		stockQuantity = variant.StockQuantity
		variantTitle = &variant.Title
		variantObjID = &variant.ID
	} else {
		basePrice = product.BasePrice
		finalPrice = basePrice * (1 - product.DiscountPercent/100)
		if product.StockQuantity != nil {
			stockQuantity = *product.StockQuantity
		}
	}

	snapshot := &model.CartItemSnapshot{
		ProductID:       product.ID,
		ProductName:     product.Name,
		Brand:           product.Brand,
		Thumbnail:       product.Thumbnail,
		VariantID:       variantObjID,
		VariantTitle:    variantTitle,
		BasePrice:       basePrice,
		DiscountPercent: product.DiscountPercent,
		PriceAdjustment: priceAdjustment,
		FinalPrice:      finalPrice,
		StockQuantity:   stockQuantity,
		IsAvailable:     product.Status == model.ProductStatusActive && stockQuantity > 0,
	}

	return snapshot, nil
}

func (s *cartService) ClearCart(ctx context.Context, userID string) error {
	err := s.cartRepo.ClearCart(ctx, userID)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return apperror.ErrCartNotFound
		}
		return err
	}
	return nil
}

func (s *cartService) GetItemCount(ctx context.Context, userID string) (int, error) {
	cart, err := s.cartRepo.GetByUserID(ctx, userID)
	if err != nil {
		return 0, err
	}
	return len(cart.Items), nil
}

func (s *cartService) AddItem(ctx context.Context, userID string, req dto.AddCartItemRequest) (*dto.CartResponse, error) {
	// Check if product requires variant
	product, err := s.productRepo.GetByID(ctx, req.ProductID)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, apperror.ErrProductNotFound
		}
		return nil, err
	}

	// Require variant_id for products with variants
	if product.IsHasVariant && req.VariantID == "" {
		return nil, apperror.ErrVariantRequired
	}

	productObjID, _ := primitive.ObjectIDFromHex(req.ProductID)
	var variantObjID *primitive.ObjectID
	if req.VariantID != "" {
		vid, _ := primitive.ObjectIDFromHex(req.VariantID)
		variantObjID = &vid
	}

	variantIDPtr := (*string)(nil)
	if req.VariantID != "" {
		variantIDPtr = &req.VariantID
	}

	// Check if item already exists in cart
	cart, err := s.cartRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	for _, item := range cart.Items {
		// Match by product_id and variant_id (both nil or both equal)
		if item.ProductID == productObjID {
			if (item.VariantID == nil && variantObjID == nil) ||
				(item.VariantID != nil && variantObjID != nil && *item.VariantID == *variantObjID) {
				// Item exists, update quantity
				newQuantity := item.Quantity + req.Quantity
				isAvailable, _, err := s.productRepo.CheckAvailability(ctx, req.ProductID, variantIDPtr, newQuantity)
				if err != nil {
					return nil, err
				}
				if !isAvailable {
					return nil, apperror.ErrInsufficientStock
				}
				err = s.cartRepo.UpdateItemQuantity(ctx, userID, item.ID.Hex(), newQuantity)
				if err != nil {
					return nil, err
				}
				return s.GetCart(ctx, userID)
			}
		}
	}

	isAvailable, _, err := s.productRepo.CheckAvailability(ctx, req.ProductID, variantIDPtr, req.Quantity)
	if err != nil {
		return nil, err
	}
	if !isAvailable {
		return nil, apperror.ErrInsufficientStock
	}

	// Item not found, add new item
	cartItem := model.CartItem{
		ID:        primitive.NewObjectID(),
		ProductID: productObjID,
		VariantID: variantObjID,
		Quantity:  req.Quantity,
		AddedAt:   time.Now(),
		UpdatedAt: time.Now(),
	}

	_, err = s.cartRepo.AddItem(ctx, userID, cartItem)
	if err != nil {
		return nil, err
	}

	return s.GetCart(ctx, userID)
}

func (s *cartService) UpdateItemQuantity(ctx context.Context, userID string, req dto.UpdateCartItemRequest) (*dto.CartResponse, error) {
	if req.Quantity <= 0 {
		return s.RemoveItem(ctx, userID, req.ItemID)
	}

	// Get cart item to find product and variant
	cartItem, err := s.cartRepo.GetCartItemByID(ctx, userID, req.ItemID)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, apperror.ErrCartItemNotFound
		}
		return nil, err
	}

	// Check availability with new quantity
	variantIDPtr := (*string)(nil)
	if cartItem.VariantID != nil {
		vid := cartItem.VariantID.Hex()
		variantIDPtr = &vid
	}

	isAvailable, _, err := s.productRepo.CheckAvailability(ctx, cartItem.ProductID.Hex(), variantIDPtr, req.Quantity)
	if err != nil {
		return nil, err
	}
	if !isAvailable {
		return nil, apperror.ErrInsufficientStock
	}

	err = s.cartRepo.UpdateItemQuantity(ctx, userID, req.ItemID, req.Quantity)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, apperror.ErrCartItemNotFound
		}
		return nil, err
	}

	return s.GetCart(ctx, userID)
}

func (s *cartService) RemoveItem(ctx context.Context, userID string, itemID string) (*dto.CartResponse, error) {
	err := s.cartRepo.RemoveItem(ctx, userID, itemID)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, apperror.ErrCartItemNotFound
		}
		return nil, err
	}

	return s.GetCart(ctx, userID)
}

func (s *cartService) AddItems(ctx context.Context, userID string, req dto.AddCartItemsRequest) (*dto.CartResponse, error) {
	for _, item := range req.Items {
		_, err := s.AddItem(ctx, userID, item)
		if err != nil {
			return nil, err
		}
	}

	return s.GetCart(ctx, userID)
}

func (s *cartService) RemoveItems(ctx context.Context, userID string, req dto.RemoveCartItemsRequest) (*dto.CartResponse, error) {
	for _, item := range req.Items {
		_, err := s.RemoveItem(ctx, userID, item.ID)
		if err != nil && !errors.Is(err, apperror.ErrCartItemNotFound) {
			return nil, err
		}
	}

	return s.GetCart(ctx, userID)
}
func (s *cartService) ValidateCart(ctx context.Context, userID string) (*dto.CartValidationResult, error) {
	cart, err := s.cartRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	result := &dto.CartValidationResult{
		IsValid:      true,
		InvalidItems: []dto.CartItemInvalidReason{},
	}

	var totalPrice float64

	for _, item := range cart.Items {
		var variantIDStr *string
		if item.VariantID != nil {
			v := item.VariantID.Hex()
			variantIDStr = &v
		}

		invalidReason := dto.CartItemInvalidReason{
			ItemID: item.ID.Hex(),
		}

		// ── 1. Lấy snapshot — product có tồn tại không ──────────────
		snapshot, err := s.GetSnapshot(ctx, item.ProductID.Hex(), variantIDStr)
		if err != nil {
			result.IsValid = false
			invalidReason.Reason = "product_not_found"
			result.InvalidItems = append(result.InvalidItems, invalidReason)
			continue
		}

		// ── 2. Product có đang active không ─────────────────────────
		if !snapshot.IsAvailable && snapshot.StockQuantity == 0 {
			result.IsValid = false
			invalidReason.Reason = "product_inactive"
			result.InvalidItems = append(result.InvalidItems, invalidReason)
			continue
		}

		// ── 3. Còn đủ hàng không ────────────────────────────────────
		if snapshot.StockQuantity <= 0 {
			result.IsValid = false
			invalidReason.Reason = "out_of_stock"
			result.InvalidItems = append(result.InvalidItems, invalidReason)
			continue
		}

		if snapshot.StockQuantity < item.Quantity {
			result.IsValid = false
			invalidReason.Reason = "insufficient_stock"
			result.InvalidItems = append(result.InvalidItems, invalidReason)
			continue
		}

		// ── 4. Giá có thay đổi không ────────────────────────────────
		// Không block checkout — chỉ thông báo để user biết
		// (nếu muốn block → đổi thành continue và set IsValid = false)

		// ── 5. Item hợp lệ → cộng vào tổng ─────────────────────────
		totalPrice += snapshot.FinalPrice * float64(item.Quantity)
	}

	result.TotalPrice = totalPrice
	return result, nil
}
