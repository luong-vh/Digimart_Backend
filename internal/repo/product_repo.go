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
	GetDeletedByID(ctx context.Context, id string) (*model.Product, error)
	GetByIDs(ctx context.Context, ids []string) ([]*model.Product, error)
	GetBySlug(ctx context.Context, slug string) (*model.Product, error)
	GetBySKU(ctx context.Context, sku string) (*model.Product, error)
	GetBySellerID(ctx context.Context, sellerID string) ([]*model.Product, error)
	GetByCategoryID(ctx context.Context, categoryID string) ([]*model.Product, error)
	Find(ctx context.Context, filter Filter, opts *FindOptions) ([]*model.Product, int64, error)

	// Stats methods
	CountTotal(ctx context.Context) (int64, error)
	CountByStatus(ctx context.Context, status model.ProductStatus) (int64, error)
	CountBySeller(ctx context.Context, sellerID string) (int64, error)
	CountByCategory(ctx context.Context, categoryID string) (int64, error)
	CountCreatedAfter(ctx context.Context, since time.Time) (int64, error)
	CountOutOfStock(ctx context.Context) (int64, error)

	// Inventory methods
	UpdateStock(ctx context.Context, productID string, quantity int) error
	UpdateVariantStock(ctx context.Context, productID string, variantID string, quantity int) error
	IncrementSoldCount(ctx context.Context, productID string, count int) error
	IncrementViewCount(ctx context.Context, productID string) error
	UpdateRating(ctx context.Context, productID string, rating float64, ratingCount int) error
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

func (r *productRepo) GetDeletedByID(ctx context.Context, id string) (*model.Product, error) {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, apperror.ErrInvalidID
	}

	filter := bson.M{"_id": objectID, "deleted_at": bson.M{"$exists": true}}
	var product model.Product
	err = r.productCollection.FindOne(ctx, filter).Decode(&product)
	if err != nil {
		return nil, err
	}
	return &product, nil
}

func (r *productRepo) GetByIDs(ctx context.Context, ids []string) ([]*model.Product, error) {
	if len(ids) == 0 {
		return []*model.Product{}, nil
	}

	objIDs := make([]primitive.ObjectID, 0, len(ids))
	for _, id := range ids {
		if objID, err := primitive.ObjectIDFromHex(id); err == nil {
			objIDs = append(objIDs, objID)
		}
	}

	filter := bson.M{"_id": bson.M{"$in": objIDs}, "deleted_at": bson.M{"$exists": false}}
	cursor, err := r.productCollection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var products []*model.Product
	if err := cursor.All(ctx, &products); err != nil {
		return nil, err
	}
	return products, nil
}

func (r *productRepo) GetBySlug(ctx context.Context, slug string) (*model.Product, error) {
	filter := bson.M{"slug": slug, "deleted_at": bson.M{"$exists": false}}
	var product model.Product
	err := r.productCollection.FindOne(ctx, filter).Decode(&product)
	if err != nil {
		return nil, err
	}
	return &product, nil
}

func (r *productRepo) GetBySKU(ctx context.Context, sku string) (*model.Product, error) {
	filter := bson.M{"sku": sku, "deleted_at": bson.M{"$exists": false}}
	var product model.Product
	err := r.productCollection.FindOne(ctx, filter).Decode(&product)
	if err != nil {
		return nil, err
	}
	return &product, nil
}

func (r *productRepo) GetBySellerID(ctx context.Context, sellerID string) ([]*model.Product, error) {
	objectID, err := primitive.ObjectIDFromHex(sellerID)
	if err != nil {
		return nil, apperror.ErrInvalidID
	}

	filter := bson.M{"seller_id": objectID, "deleted_at": bson.M{"$exists": false}}
	cursor, err := r.productCollection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var products []*model.Product
	if err := cursor.All(ctx, &products); err != nil {
		return nil, err
	}
	return products, nil
}

func (r *productRepo) GetByCategoryID(ctx context.Context, categoryID string) ([]*model.Product, error) {
	objectID, err := primitive.ObjectIDFromHex(categoryID)
	if err != nil {
		return nil, apperror.ErrInvalidID
	}

	filter := bson.M{"category_id": objectID, "deleted_at": bson.M{"$exists": false}}
	cursor, err := r.productCollection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var products []*model.Product
	if err := cursor.All(ctx, &products); err != nil {
		return nil, err
	}
	return products, nil
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

func (r *productRepo) CountBySeller(ctx context.Context, sellerID string) (int64, error) {
	objectID, err := primitive.ObjectIDFromHex(sellerID)
	if err != nil {
		return 0, apperror.ErrInvalidID
	}

	filter := bson.M{
		"seller_id":  objectID,
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

func (r *productRepo) CountCreatedAfter(ctx context.Context, since time.Time) (int64, error) {
	filter := bson.M{
		"created_at": bson.M{"$gte": since},
		"deleted_at": bson.M{"$exists": false},
	}
	return r.productCollection.CountDocuments(ctx, filter)
}

func (r *productRepo) CountOutOfStock(ctx context.Context) (int64, error) {
	filter := bson.M{
		"$or": []bson.M{
			{"status": model.ProductStatusOutOfStock},
			{"stock_quantity": bson.M{"$lte": 0}, "has_variants": false},
		},
		"deleted_at": bson.M{"$exists": false},
	}
	return r.productCollection.CountDocuments(ctx, filter)
}

// Inventory methods
func (r *productRepo) UpdateStock(ctx context.Context, productID string, quantity int) error {
	objectID, err := primitive.ObjectIDFromHex(productID)
	if err != nil {
		return apperror.ErrInvalidID
	}

	filter := bson.M{"_id": objectID, "deleted_at": bson.M{"$exists": false}}
	update := bson.M{
		"$set": bson.M{
			"stock_quantity": quantity,
			"updated_at":     time.Now(),
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

func (r *productRepo) UpdateVariantStock(ctx context.Context, productID string, variantID string, quantity int) error {
	productObjID, err := primitive.ObjectIDFromHex(productID)
	if err != nil {
		return apperror.ErrInvalidID
	}

	variantObjID, err := primitive.ObjectIDFromHex(variantID)
	if err != nil {
		return apperror.ErrInvalidID
	}

	filter := bson.M{
		"_id":          productObjID,
		"variants._id": variantObjID,
		"deleted_at":   bson.M{"$exists": false},
	}
	update := bson.M{
		"$set": bson.M{
			"variants.$.stock_quantity": quantity,
			"updated_at":                time.Now(),
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

func (r *productRepo) IncrementViewCount(ctx context.Context, productID string) error {
	objectID, err := primitive.ObjectIDFromHex(productID)
	if err != nil {
		return apperror.ErrInvalidID
	}

	filter := bson.M{"_id": objectID, "deleted_at": bson.M{"$exists": false}}
	update := bson.M{
		"$inc": bson.M{"view_count": 1},
	}

	_, err = r.productCollection.UpdateOne(ctx, filter, update)
	return err
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
