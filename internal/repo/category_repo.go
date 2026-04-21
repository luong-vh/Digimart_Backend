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

type CategoryRepo interface {
	Create(ctx context.Context, category *model.Category) (*model.Category, error)
	Update(ctx context.Context, category *model.Category) (*model.Category, error)
	Delete(ctx context.Context, id string) error
	GetByID(ctx context.Context, id string) (*model.Category, error)
	GetByName(ctx context.Context, name string) (*model.Category, error)
	GetAll(ctx context.Context) ([]*model.Category, error)
	GetByParentID(ctx context.Context, parentID string) ([]*model.Category, error)
	GetRootCategories(ctx context.Context) ([]*model.Category, error)
	Find(ctx context.Context, filter Filter, opts *FindOptions) ([]*model.Category, int64, error)
	HasChildren(ctx context.Context, id string) (bool, error)
	HasProducts(ctx context.Context, id string) (bool, error)
	CountTotal(ctx context.Context) (int64, error)
}

type categoryRepo struct {
	categoryCollection *mongo.Collection
	productCollection  *mongo.Collection
}

func NewCategoryRepo(db *mongo.Database) CategoryRepo {
	return &categoryRepo{
		categoryCollection: db.Collection(config.CategoryColName),
		productCollection:  db.Collection(config.ProductColName),
	}
}

func (r *categoryRepo) Create(ctx context.Context, category *model.Category) (*model.Category, error) {
	category.CreatedAt = time.Now()
	category.UpdatedAt = time.Now()

	result, err := r.categoryCollection.InsertOne(ctx, category)
	if err != nil {
		return nil, err
	}

	if oid, ok := result.InsertedID.(primitive.ObjectID); ok {
		category.ID = oid
	}

	return category, nil
}

func (r *categoryRepo) Update(ctx context.Context, category *model.Category) (*model.Category, error) {
	category.UpdatedAt = time.Now()

	filter := bson.M{"_id": category.ID}
	update := bson.M{
		"$set": bson.M{
			"name":        category.Name,
			"description": category.Description,
			"parent_id":   category.ParentID,
			"updated_at":  category.UpdatedAt,
		},
	}

	result, err := r.categoryCollection.UpdateOne(ctx, filter, update)
	if err != nil {
		return nil, err
	}

	if result.MatchedCount == 0 {
		return nil, mongo.ErrNoDocuments
	}

	return category, nil
}

func (r *categoryRepo) Delete(ctx context.Context, id string) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return apperror.ErrInvalidID
	}

	filter := bson.M{"_id": objectID}
	result, err := r.categoryCollection.DeleteOne(ctx, filter)
	if err != nil {
		return err
	}

	if result.DeletedCount == 0 {
		return mongo.ErrNoDocuments
	}

	return nil
}

func (r *categoryRepo) GetByID(ctx context.Context, id string) (*model.Category, error) {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, apperror.ErrInvalidID
	}

	filter := bson.M{"_id": objectID}
	var category model.Category
	err = r.categoryCollection.FindOne(ctx, filter).Decode(&category)
	if err != nil {
		return nil, err
	}

	return &category, nil
}

func (r *categoryRepo) GetByName(ctx context.Context, name string) (*model.Category, error) {
	filter := bson.M{"name": name}
	var category model.Category
	err := r.categoryCollection.FindOne(ctx, filter).Decode(&category)
	if err != nil {
		return nil, err
	}

	return &category, nil
}

func (r *categoryRepo) GetAll(ctx context.Context) ([]*model.Category, error) {
	findOptions := options.Find().SetSort(bson.D{{Key: "name", Value: 1}})

	cursor, err := r.categoryCollection.Find(ctx, bson.M{}, findOptions)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var categories []*model.Category
	if err := cursor.All(ctx, &categories); err != nil {
		return nil, err
	}

	return categories, nil
}

func (r *categoryRepo) GetByParentID(ctx context.Context, parentID string) ([]*model.Category, error) {
	objectID, err := primitive.ObjectIDFromHex(parentID)
	if err != nil {
		return nil, apperror.ErrInvalidID
	}

	filter := bson.M{"parent_id": objectID}
	findOptions := options.Find().SetSort(bson.D{{Key: "name", Value: 1}})

	cursor, err := r.categoryCollection.Find(ctx, filter, findOptions)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var categories []*model.Category
	if err := cursor.All(ctx, &categories); err != nil {
		return nil, err
	}

	return categories, nil
}

func (r *categoryRepo) GetRootCategories(ctx context.Context) ([]*model.Category, error) {
	filter := bson.M{
		"$or": []bson.M{
			{"parent_id": bson.M{"$exists": false}},
			{"parent_id": primitive.NilObjectID},
		},
	}
	findOptions := options.Find().SetSort(bson.D{{Key: "name", Value: 1}})

	cursor, err := r.categoryCollection.Find(ctx, filter, findOptions)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var categories []*model.Category
	if err := cursor.All(ctx, &categories); err != nil {
		return nil, err
	}

	return categories, nil
}

func (r *categoryRepo) Find(ctx context.Context, filter Filter, opts *FindOptions) ([]*model.Category, int64, error) {
	mongoFilter := bson.M(filter)

	// Get total count
	total, err := r.categoryCollection.CountDocuments(ctx, mongoFilter)
	if err != nil {
		return nil, 0, err
	}

	if total == 0 {
		return []*model.Category{}, 0, nil
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

	cursor, err := r.categoryCollection.Find(ctx, mongoFilter, findOptions)
	if err != nil {
		return nil, total, err
	}
	defer cursor.Close(ctx)

	var categories []*model.Category
	if err := cursor.All(ctx, &categories); err != nil {
		return nil, 0, err
	}

	return categories, total, nil
}

func (r *categoryRepo) HasChildren(ctx context.Context, id string) (bool, error) {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return false, apperror.ErrInvalidID
	}

	filter := bson.M{"parent_id": objectID}
	count, err := r.categoryCollection.CountDocuments(ctx, filter)
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

func (r *categoryRepo) HasProducts(ctx context.Context, id string) (bool, error) {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return false, apperror.ErrInvalidID
	}

	filter := bson.M{
		"category_id": objectID,
		"deleted_at":  bson.M{"$exists": false},
	}
	count, err := r.productCollection.CountDocuments(ctx, filter)
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

func (r *categoryRepo) CountTotal(ctx context.Context) (int64, error) {
	return r.categoryCollection.CountDocuments(ctx, bson.M{})
}
