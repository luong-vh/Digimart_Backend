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

type CartRepo interface {
	Create(ctx context.Context, cart *model.Cart) (*model.Cart, error)
	Update(ctx context.Context, cart *model.Cart) (*model.Cart, error)
	Delete(ctx context.Context, id string) error
	GetByID(ctx context.Context, id string) (*model.Cart, error)
	GetByUserID(ctx context.Context, userID string) (*model.Cart, error)
	Find(ctx context.Context, filter Filter, opts *FindOptions) ([]*model.Cart, int64, error)

	// Cart item operations (by item ID)
	AddItem(ctx context.Context, userID string, item model.CartItem) (*model.CartItem, error)
	GetCartItemByID(ctx context.Context, userID string, itemID string) (*model.CartItem, error)
	UpdateItem(ctx context.Context, userID string, itemID string, item model.CartItem) error
	UpdateItemQuantity(ctx context.Context, userID string, itemID string, quantity int) error
	RemoveItem(ctx context.Context, userID string, itemID string) error
	ClearCart(ctx context.Context, userID string) error

	// Stats methods
	CountTotal(ctx context.Context) (int64, error)
	CountNonEmpty(ctx context.Context) (int64, error)
	CountByUser(ctx context.Context, userID string) (int64, error)
	GetAverageItemCount(ctx context.Context) (float64, error)
}

type cartRepo struct {
	cartCollection *mongo.Collection
}

func NewCartRepo(db *mongo.Database) CartRepo {
	return &cartRepo{cartCollection: db.Collection(config.CartColName)}
}

func (r *cartRepo) Create(ctx context.Context, cart *model.Cart) (*model.Cart, error) {
	cart.CreatedAt = time.Now()
	cart.UpdatedAt = time.Now()

	result, err := r.cartCollection.InsertOne(ctx, cart)
	if err != nil {
		return nil, err
	}

	if oid, ok := result.InsertedID.(primitive.ObjectID); ok {
		cart.ID = oid
	}

	return cart, nil
}

func (r *cartRepo) Update(ctx context.Context, cart *model.Cart) (*model.Cart, error) {
	cart.UpdatedAt = time.Now()

	filter := bson.M{"_id": cart.ID}
	update := bson.M{"$set": cart}

	result, err := r.cartCollection.UpdateOne(ctx, filter, update)
	if err != nil {
		return nil, err
	}
	if result.MatchedCount == 0 {
		return nil, mongo.ErrNoDocuments
	}
	return cart, nil
}

func (r *cartRepo) Delete(ctx context.Context, id string) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return apperror.ErrInvalidID
	}

	filter := bson.M{"_id": objectID}
	result, err := r.cartCollection.DeleteOne(ctx, filter)
	if err != nil {
		return err
	}
	if result.DeletedCount == 0 {
		return mongo.ErrNoDocuments
	}
	return nil
}

func (r *cartRepo) GetByID(ctx context.Context, id string) (*model.Cart, error) {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, apperror.ErrInvalidID
	}

	filter := bson.M{"_id": objectID}
	var cart model.Cart
	err = r.cartCollection.FindOne(ctx, filter).Decode(&cart)
	if err != nil {
		return nil, err
	}
	return &cart, nil
}

func (r *cartRepo) GetByUserID(ctx context.Context, userID string) (*model.Cart, error) {
	objectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, apperror.ErrInvalidID
	}

	filter := bson.M{"user_id": objectID}
	var cart model.Cart
	err = r.cartCollection.FindOne(ctx, filter).Decode(&cart)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			// Create new cart if not exists
			newCart := &model.Cart{
				UserID:    objectID,
				Items:     []model.CartItem{},
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}
			return r.Create(ctx, newCart)
		}
		return nil, err
	}
	return &cart, nil
}

func (r *cartRepo) Find(ctx context.Context, filter Filter, opts *FindOptions) ([]*model.Cart, int64, error) {
	mongoFilter := bson.M(filter)

	// Get total count
	countPipeline := mongo.Pipeline{
		{{"$match", mongoFilter}},
		{{"$count", "total"}},
	}
	cursor, err := r.cartCollection.Aggregate(ctx, countPipeline)
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
		return []*model.Cart{}, 0, nil
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

	cursor, err = r.cartCollection.Find(ctx, mongoFilter, findOptions)
	if err != nil {
		return nil, total, err
	}
	defer cursor.Close(ctx)

	var carts []*model.Cart
	if err := cursor.All(ctx, &carts); err != nil {
		return nil, 0, err
	}

	return carts, total, nil
}

// Cart item operations
func (r *cartRepo) AddItem(ctx context.Context, userID string, item model.CartItem) (*model.CartItem, error) {
	userObjID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, apperror.ErrInvalidID
	}

	// Generate new ID for the item
	item.ID = primitive.NewObjectID()
	item.AddedAt = time.Now()

	filter := bson.M{"user_id": userObjID}

	// Add new item to cart, create cart if not exists
	update := bson.M{
		"$push": bson.M{"items": item},
		"$set":  bson.M{"updated_at": time.Now()},
		"$setOnInsert": bson.M{
			"user_id":    userObjID,
			"created_at": time.Now(),
		},
	}

	opts := options.Update().SetUpsert(true)
	_, err = r.cartCollection.UpdateOne(ctx, filter, update, opts)
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *cartRepo) GetCartItemByID(ctx context.Context, userID string, itemID string) (*model.CartItem, error) {
	userObjID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, apperror.ErrInvalidID
	}

	itemObjID, err := primitive.ObjectIDFromHex(itemID)
	if err != nil {
		return nil, apperror.ErrInvalidID
	}

	filter := bson.M{
		"user_id":    userObjID,
		"items._id": itemObjID,
	}

	projection := bson.M{"items.$": 1}
	var result struct {
		Items []model.CartItem `bson:"items"`
	}

	err = r.cartCollection.FindOne(ctx, filter, options.FindOne().SetProjection(projection)).Decode(&result)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, mongo.ErrNoDocuments
		}
		return nil, err
	}

	if len(result.Items) == 0 {
		return nil, mongo.ErrNoDocuments
	}

	return &result.Items[0], nil
}

func (r *cartRepo) UpdateItem(ctx context.Context, userID string, itemID string, item model.CartItem) error {
	userObjID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return apperror.ErrInvalidID
	}

	itemObjID, err := primitive.ObjectIDFromHex(itemID)
	if err != nil {
		return apperror.ErrInvalidID
	}

	item.ID = itemObjID
	item.AddedAt = time.Now()

	filter := bson.M{
		"user_id":   userObjID,
		"items._id": itemObjID,
	}

	update := bson.M{
		"$set": bson.M{
			"items.$":    item,
			"updated_at": time.Now(),
		},
	}

	result, err := r.cartCollection.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return mongo.ErrNoDocuments
	}
	return nil
}

func (r *cartRepo) UpdateItemQuantity(ctx context.Context, userID string, itemID string, quantity int) error {
	userObjID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return apperror.ErrInvalidID
	}

	itemObjID, err := primitive.ObjectIDFromHex(itemID)
	if err != nil {
		return apperror.ErrInvalidID
	}

	filter := bson.M{
		"user_id":   userObjID,
		"items._id": itemObjID,
	}

	update := bson.M{
		"$set": bson.M{
			"items.$.quantity": quantity,
			"updated_at":     time.Now(),
		},
	}

	result, err := r.cartCollection.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return mongo.ErrNoDocuments
	}
	return nil
}

func (r *cartRepo) RemoveItem(ctx context.Context, userID string, itemID string) error {
	userObjID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return apperror.ErrInvalidID
	}

	itemObjID, err := primitive.ObjectIDFromHex(itemID)
	if err != nil {
		return apperror.ErrInvalidID
	}

	filter := bson.M{"user_id": userObjID}

	pullFilter := bson.M{
		"_id": itemObjID,
	}

	update := bson.M{
		"$pull": bson.M{"items": pullFilter},
		"$set":  bson.M{"updated_at": time.Now()},
	}

	result, err := r.cartCollection.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return mongo.ErrNoDocuments
	}
	return nil
}

func (r *cartRepo) ClearCart(ctx context.Context, userID string) error {
	userObjID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return apperror.ErrInvalidID
	}

	filter := bson.M{"user_id": userObjID}
	update := bson.M{
		"$set": bson.M{
			"items":      []model.CartItem{},
			"updated_at": time.Now(),
		},
	}

	result, err := r.cartCollection.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return mongo.ErrNoDocuments
	}
	return nil
}

func (r *cartRepo) UpdateItemSnapshot(ctx context.Context, userID string, productID string, variantID *string, snapshot *model.CartItemSnapshot) error {
	userObjID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return apperror.ErrInvalidID
	}

	productObjID, err := primitive.ObjectIDFromHex(productID)
	if err != nil {
		return apperror.ErrInvalidID
	}

	var filter bson.M
	if variantID != nil {
		variantObjID, err := primitive.ObjectIDFromHex(*variantID)
		if err != nil {
			return apperror.ErrInvalidID
		}
		filter = bson.M{
			"user_id":          userObjID,
			"items.product_id": productObjID,
			"items.variant_id": variantObjID,
		}
	} else {
		filter = bson.M{
			"user_id":          userObjID,
			"items.product_id": productObjID,
			"items.variant_id": bson.M{"$exists": false},
		}
	}

	update := bson.M{
		"$set": bson.M{
			"items.$.snapshot": snapshot,
			"updated_at":       time.Now(),
		},
	}

	_, err = r.cartCollection.UpdateOne(ctx, filter, update)
	return err
}

// Stats methods
func (r *cartRepo) CountTotal(ctx context.Context) (int64, error) {
	return r.cartCollection.CountDocuments(ctx, bson.M{})
}

func (r *cartRepo) CountNonEmpty(ctx context.Context) (int64, error) {
	filter := bson.M{
		"items": bson.M{"$exists": true, "$ne": []model.CartItem{}},
	}
	return r.cartCollection.CountDocuments(ctx, filter)
}

func (r *cartRepo) CountByUser(ctx context.Context, userID string) (int64, error) {
	userObjID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return 0, apperror.ErrInvalidID
	}

	filter := bson.M{"user_id": userObjID}
	return r.cartCollection.CountDocuments(ctx, filter)
}

func (r *cartRepo) GetAverageItemCount(ctx context.Context) (float64, error) {
	pipeline := mongo.Pipeline{
		{{"$match", bson.M{"items": bson.M{"$exists": true}}}},
		{{"$project", bson.M{"itemCount": bson.M{"$size": "$items"}}}},
		{{"$group", bson.M{
			"_id":     nil,
			"average": bson.M{"$avg": "$itemCount"},
		}}},
	}

	cursor, err := r.cartCollection.Aggregate(ctx, pipeline)
	if err != nil {
		return 0, err
	}
	defer cursor.Close(ctx)

	var result []struct {
		Average float64 `bson:"average"`
	}
	if err = cursor.All(ctx, &result); err != nil {
		return 0, err
	}

	if len(result) > 0 {
		return result[0].Average, nil
	}
	return 0, nil
}
