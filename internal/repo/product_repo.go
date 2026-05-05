package repo

import (
	"context"
	"time"

	"github.com/luong-vh/Digimart_Backend/internal/apperror"
	"github.com/luong-vh/Digimart_Backend/internal/config"
	"github.com/luong-vh/Digimart_Backend/internal/model"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type ProductRepo interface {
	Create(ctx context.Context, product *model.Product) (*model.Product, error)
	Update(ctx context.Context, product *model.Product) (*model.Product, error)
	Delete(ctx context.Context, id string) error
	SoftDelete(ctx context.Context, productID string) error
	GetByID(ctx context.Context, id string) (*model.Product, error)

	Find(ctx context.Context, filter Filter, opts *FindOptions) ([]*model.Product, int64, error)

	// Stats methods
	CountTotal(ctx context.Context) (int64, error)
	CountByStatus(ctx context.Context, status model.ProductStatus) (int64, error)
	CountByCategory(ctx context.Context, categoryID string) (int64, error)

	IncrementSoldCount(ctx context.Context, productID string, count int) error
	UpdateRating(ctx context.Context, productID string, rating float64, ratingCount int) error

	// Cart snapshot methods
	GetProductWithVariant(ctx context.Context, productID string, variantID string) (*model.Product, *model.ProductVariant, error)
	CheckAvailability(ctx context.Context, productID string, variantID *string, requestedQty int) (bool, int, error)

	// Media methods
	AddImage(ctx context.Context, productID string, image model.Image) error
	AddImages(ctx context.Context, productID string, images []model.Image) error
	AddVideo(ctx context.Context, productID string, video model.Video) error
	AddVideos(ctx context.Context, productID string, videos []model.Video) error

	// Variant CRUD
	CreateVariant(ctx context.Context, productID string, variant model.ProductVariant) (*model.ProductVariant, error)
	UpdateVariant(ctx context.Context, productID string, variant model.ProductVariant) (*model.ProductVariant, error)
}

type productRepo struct {
	productCollection *mongo.Collection
}

func NewProductRepo(db *mongo.Database) ProductRepo {
	return &productRepo{productCollection: db.Collection(config.ProductColName)}
}

func (r *productRepo) Create(ctx context.Context, product *model.Product) (*model.Product, error) {
	product.CreatedAt = time.Now()
	product.UpdatedAt = time.Now()

	result, err := r.productCollection.InsertOne(ctx, product)
	if err != nil {
		return nil, err
	}

	if oid, ok := result.InsertedID.(primitive.ObjectID); ok {
		product.ID = oid
	}

	return product, nil
}

func (r *productRepo) Update(ctx context.Context, product *model.Product) (*model.Product, error) {
	product.UpdatedAt = time.Now()

	filter := bson.M{"_id": product.ID, "deleted_at": bson.M{"$exists": false}}
	update := bson.M{"$set": product}

	result, err := r.productCollection.UpdateOne(ctx, filter, update)
	if err != nil {
		return nil, err
	}
	if result.MatchedCount == 0 {
		return nil, mongo.ErrNoDocuments
	}
	return product, nil
}

func (r *productRepo) SoftDelete(ctx context.Context, productID string) error {
	objID, err := primitive.ObjectIDFromHex(productID)
	if err != nil {
		return apperror.ErrInvalidID
	}

	filter := bson.M{"_id": objID, "deleted_at": bson.M{"$exists": false}}
	update := bson.M{
		"$set": bson.M{
			"deleted_at": time.Now(),
			"updated_at": time.Now(),
		},
	}

	result, err := r.productCollection.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return mongo.ErrNoDocuments
	}
	return nil
}

func (r *productRepo) Delete(ctx context.Context, id string) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return apperror.ErrInvalidID
	}

	filter := bson.M{"_id": objectID}
	result, err := r.productCollection.DeleteOne(ctx, filter)
	if err != nil {
		return err
	}
	if result.DeletedCount == 0 {
		return mongo.ErrNoDocuments
	}
	return nil
}

func (r *productRepo) GetByID(ctx context.Context, id string) (*model.Product, error) {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, apperror.ErrInvalidID
	}

	filter := bson.M{"_id": objectID, "deleted_at": bson.M{"$exists": false}}
	var product model.Product
	err = r.productCollection.FindOne(ctx, filter).Decode(&product)
	if err != nil {
		return nil, err
	}
	return &product, nil
}

func (r *productRepo) Find(ctx context.Context, filter Filter, opts *FindOptions) ([]*model.Product, int64, error) {
	// Add soft delete filter
	mongoFilter := bson.M(filter)
	mongoFilter["deleted_at"] = bson.M{"$exists": false}

	// Get total count
	countPipeline := mongo.Pipeline{
		{{"$match", mongoFilter}},
		{{"$count", "total"}},
	}
	cursor, err := r.productCollection.Aggregate(ctx, countPipeline)
	if err != nil {
		return nil, 0, err
	}

	var countResult []struct {
		Total int64 `bson:"total"`
	}
	if err = cursor.All(ctx, &countResult); err != nil {
		return nil, 0, err
	}

	var total int64
	if len(countResult) > 0 {
		total = countResult[0].Total
	}

	if total == 0 {
		return []*model.Product{}, 0, nil
	}

	// Get paginated data
	findOptions := options.Find()
	if opts != nil {
		if opts.Sort != nil {
			sortDoc := bson.D{}
			for key, value := range opts.Sort {
				sortDoc = append(sortDoc, bson.E{Key: key, Value: value})
			}
			findOptions.SetSort(sortDoc)
		}
		if opts.Skip > 0 {
			findOptions.SetSkip(opts.Skip)
		}
		if opts.Limit > 0 {
			findOptions.SetLimit(opts.Limit)
		}
	}

	cursor, err = r.productCollection.Find(ctx, mongoFilter, findOptions)
	if err != nil {
		return nil, total, err
	}
	defer cursor.Close(ctx)

	var products []*model.Product
	if err := cursor.All(ctx, &products); err != nil {
		return nil, 0, err
	}

	return products, total, nil
}

// Stats methods
func (r *productRepo) CountTotal(ctx context.Context) (int64, error) {
	filter := bson.M{"deleted_at": bson.M{"$exists": false}}
	return r.productCollection.CountDocuments(ctx, filter)
}

func (r *productRepo) CountByStatus(ctx context.Context, status model.ProductStatus) (int64, error) {
	filter := bson.M{
		"status":     status,
		"deleted_at": bson.M{"$exists": false},
	}
	return r.productCollection.CountDocuments(ctx, filter)
}

func (r *productRepo) CountByCategory(ctx context.Context, categoryID string) (int64, error) {
	objectID, err := primitive.ObjectIDFromHex(categoryID)
	if err != nil {
		return 0, apperror.ErrInvalidID
	}

	filter := bson.M{
		"category_id": objectID,
		"deleted_at":  bson.M{"$exists": false},
	}
	return r.productCollection.CountDocuments(ctx, filter)
}

func (r *productRepo) IncrementSoldCount(ctx context.Context, productID string, count int) error {
	objectID, err := primitive.ObjectIDFromHex(productID)
	if err != nil {
		return apperror.ErrInvalidID
	}

	filter := bson.M{"_id": objectID, "deleted_at": bson.M{"$exists": false}}
	update := bson.M{
		"$inc": bson.M{"sold_count": count},
		"$set": bson.M{"updated_at": time.Now()},
	}

	result, err := r.productCollection.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return mongo.ErrNoDocuments
	}
	return nil
}

func (r *productRepo) UpdateRating(ctx context.Context, productID string, rating float64, ratingCount int) error {
	objectID, err := primitive.ObjectIDFromHex(productID)
	if err != nil {
		return apperror.ErrInvalidID
	}

	filter := bson.M{"_id": objectID, "deleted_at": bson.M{"$exists": false}}
	update := bson.M{
		"$set": bson.M{
			"rating":       rating,
			"rating_count": ratingCount,
			"updated_at":   time.Now(),
		},
	}

	result, err := r.productCollection.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return mongo.ErrNoDocuments
	}
	return nil
}

// Cart snapshot methods
func (r *productRepo) GetProductWithVariant(ctx context.Context, productID string, variantID string) (*model.Product, *model.ProductVariant, error) {
	product, err := r.GetByID(ctx, productID)
	if err != nil {
		return nil, nil, err
	}

	if variantID == "" {
		return product, nil, nil
	}

	variantObjID, err := primitive.ObjectIDFromHex(variantID)
	if err != nil {
		return nil, nil, apperror.ErrInvalidID
	}

	for i := range product.Variants {
		if product.Variants[i].ID == variantObjID {
			return product, &product.Variants[i], nil
		}
	}

	return nil, nil, mongo.ErrNoDocuments
}

func (r *productRepo) CheckAvailability(ctx context.Context, productID string, variantID *string, requestedQty int) (bool, int, error) {
	variantIDStr := ""
	if variantID != nil {
		variantIDStr = *variantID
	}

	product, variant, err := r.GetProductWithVariant(ctx, productID, variantIDStr)
	if err != nil {
		return false, 0, err
	}

	if product.Status != model.ProductStatusActive {
		return false, 0, nil
	}

	var availableStock int
	if variant != nil {
		availableStock = variant.StockQuantity
	} else if product.StockQuantity != nil {
		availableStock = *product.StockQuantity
	} else {
		return false, 0, nil
	}

	isAvailable := availableStock >= requestedQty
	return isAvailable, availableStock, nil
}

// Media methods
func (r *productRepo) AddImage(ctx context.Context, productID string, image model.Image) error {
	return r.AddImages(ctx, productID, []model.Image{image})
}

func (r *productRepo) AddImages(ctx context.Context, productID string, images []model.Image) error {
	objectID, err := primitive.ObjectIDFromHex(productID)
	if err != nil {
		return apperror.ErrInvalidID
	}

	filter := bson.M{"_id": objectID, "deleted_at": bson.M{"$exists": false}}
	update := bson.M{
		"$push": bson.M{"images": bson.M{"$each": images}},
		"$set":  bson.M{"updated_at": time.Now()},
	}

	result, err := r.productCollection.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return mongo.ErrNoDocuments
	}
	return nil
}

func (r *productRepo) AddVideo(ctx context.Context, productID string, video model.Video) error {
	return r.AddVideos(ctx, productID, []model.Video{video})
}

func (r *productRepo) AddVideos(ctx context.Context, productID string, videos []model.Video) error {
	objectID, err := primitive.ObjectIDFromHex(productID)
	if err != nil {
		return apperror.ErrInvalidID
	}

	filter := bson.M{"_id": objectID, "deleted_at": bson.M{"$exists": false}}
	update := bson.M{
		"$push": bson.M{"videos": bson.M{"$each": videos}},
		"$set":  bson.M{"updated_at": time.Now()},
	}

	result, err := r.productCollection.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return mongo.ErrNoDocuments
	}
	return nil
}

// Variant CRUD
func (r *productRepo) CreateVariant(ctx context.Context, productID string, variant model.ProductVariant) (*model.ProductVariant, error) {
	objectID, err := primitive.ObjectIDFromHex(productID)
	if err != nil {
		return nil, apperror.ErrInvalidID
	}

	variant.ID = primitive.NewObjectID()

	filter := bson.M{"_id": objectID, "deleted_at": bson.M{"$exists": false}}
	update := bson.M{
		"$push": bson.M{"variants": variant},
		"$set": bson.M{
			"is_has_variant": true,
			"updated_at":     time.Now(),
		},
	}

	result, err := r.productCollection.UpdateOne(ctx, filter, update)
	if err != nil {
		return nil, err
	}
	if result.MatchedCount == 0 {
		return nil, mongo.ErrNoDocuments
	}

	return &variant, nil
}

func (r *productRepo) UpdateVariant(ctx context.Context, productID string, variant model.ProductVariant) (*model.ProductVariant, error) {
	objectID, err := primitive.ObjectIDFromHex(productID)
	if err != nil {
		return nil, apperror.ErrInvalidID
	}

	filter := bson.M{
		"_id":          objectID,
		"deleted_at":   bson.M{"$exists": false},
		"variants._id": variant.ID,
	}

	updateFields := bson.M{
		"variants.$.title":            variant.Title,
		"variants.$.description":      variant.Description,
		"variants.$.price_adjustment": variant.PriceAdjustment,
		"variants.$.final_price":      variant.FinalPrice,
		"variants.$.stock_quantity":   variant.StockQuantity,
		"updated_at":                  time.Now(),
	}

	update := bson.M{"$set": updateFields}

	result, err := r.productCollection.UpdateOne(ctx, filter, update)
	if err != nil {
		return nil, err
	}
	if result.MatchedCount == 0 {
		return nil, mongo.ErrNoDocuments
	}

	return &variant, nil
}
