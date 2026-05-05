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

type ReviewRepo interface {
	// Review CRUD
	Create(ctx context.Context, review *model.Review) (*model.Review, error)
	Update(ctx context.Context, review *model.Review) (*model.Review, error)
	Delete(ctx context.Context, id string) error
	SoftDelete(ctx context.Context, reviewID string) error
	GetByID(ctx context.Context, id string) (*model.Review, error)
	GetByIDWithReplies(ctx context.Context, id string) (*model.Review, []*model.ReviewReply, error)

	// Find reviews
	GetByProductID(ctx context.Context, productID string, opts *FindOptions) ([]*model.Review, int64, error)
	GetByUserID(ctx context.Context, userID string, opts *FindOptions) ([]*model.Review, error)
	Find(ctx context.Context, filter Filter, opts *FindOptions) ([]*model.Review, int64, error)

	// Review replies
	AddReply(ctx context.Context, reply *model.ReviewReply) (*model.ReviewReply, error)
	GetRepliesByReviewID(ctx context.Context, reviewID string, opts *FindOptions) ([]*model.ReviewReply, int64, error)
	UpdateReply(ctx context.Context, reply *model.ReviewReply) error
	DeleteReply(ctx context.Context, replyID string) error

	// Stats
	CountByProductID(ctx context.Context, productID string) (int64, error)
	CountByUserID(ctx context.Context, userID string) (int64, error)
	GetAverageRating(ctx context.Context, productID string) (float64, error)
	GetRatingBreakdown(ctx context.Context, productID string) (map[int]int64, error)
	GetReviewSummary(ctx context.Context, productID string) (*model.ReviewSummary, error)

	// Engagement

}

type reviewRepo struct {
	reviewCollection      *mongo.Collection
	reviewReplyCollection *mongo.Collection
}

func NewReviewRepo(db *mongo.Database) ReviewRepo {
	return &reviewRepo{
		reviewCollection:      db.Collection(config.ReviewColName),
		reviewReplyCollection: db.Collection(config.ReviewReplyColName),
	}
}

// Review CRUD operations
func (r *reviewRepo) Create(ctx context.Context, review *model.Review) (*model.Review, error) {
	review.CreatedAt = time.Now()
	review.UpdatedAt = time.Now()
	review.Status = model.ReviewStatusApproved

	result, err := r.reviewCollection.InsertOne(ctx, review)
	if err != nil {
		return nil, err
	}

	if oid, ok := result.InsertedID.(primitive.ObjectID); ok {
		review.ID = oid
	}

	return review, nil
}

func (r *reviewRepo) Update(ctx context.Context, review *model.Review) (*model.Review, error) {
	review.UpdatedAt = time.Now()

	filter := bson.M{"_id": review.ID, "deleted_at": bson.M{"$exists": false}}
	update := bson.M{"$set": review}

	result, err := r.reviewCollection.UpdateOne(ctx, filter, update)
	if err != nil {
		return nil, err
	}
	if result.MatchedCount == 0 {
		return nil, mongo.ErrNoDocuments
	}
	return review, nil
}

func (r *reviewRepo) Delete(ctx context.Context, id string) error {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return apperror.ErrInvalidID
	}

	filter := bson.M{"_id": objID}
	result, err := r.reviewCollection.DeleteOne(ctx, filter)
	if err != nil {
		return err
	}
	if result.DeletedCount == 0 {
		return mongo.ErrNoDocuments
	}
	return nil
}

func (r *reviewRepo) SoftDelete(ctx context.Context, reviewID string) error {
	objID, err := primitive.ObjectIDFromHex(reviewID)
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

	result, err := r.reviewCollection.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return mongo.ErrNoDocuments
	}
	return nil
}

func (r *reviewRepo) GetByID(ctx context.Context, id string) (*model.Review, error) {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, apperror.ErrInvalidID
	}

	filter := bson.M{"_id": objID, "deleted_at": bson.M{"$exists": false}}
	var review model.Review
	err = r.reviewCollection.FindOne(ctx, filter).Decode(&review)
	if err != nil {
		return nil, err
	}
	return &review, nil
}

func (r *reviewRepo) GetByIDWithReplies(ctx context.Context, id string) (*model.Review, []*model.ReviewReply, error) {
	review, err := r.GetByID(ctx, id)
	if err != nil {
		return nil, nil, err
	}

	replies, _, err := r.GetRepliesByReviewID(ctx, id, &FindOptions{Limit: 1000})
	if err != nil {
		return nil, nil, err
	}

	return review, replies, nil
}

// Find reviews
func (r *reviewRepo) GetByProductID(ctx context.Context, productID string, opts *FindOptions) ([]*model.Review, int64, error) {
	productObjID, err := primitive.ObjectIDFromHex(productID)
	if err != nil {
		return nil, 0, apperror.ErrInvalidID
	}

	filter := bson.M{
		"product_id": productObjID,
		"status":     model.ReviewStatusApproved,
		"deleted_at": bson.M{"$exists": false},
	}

	// Get count
	total, err := r.reviewCollection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	if total == 0 {
		return []*model.Review{}, 0, nil
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

	cursor, err := r.reviewCollection.Find(ctx, filter, findOptions)
	if err != nil {
		return nil, total, err
	}
	defer cursor.Close(ctx)

	var reviews []*model.Review
	if err := cursor.All(ctx, &reviews); err != nil {
		return nil, 0, err
	}

	return reviews, total, nil
}

func (r *reviewRepo) GetByUserID(ctx context.Context, userID string, opts *FindOptions) ([]*model.Review, error) {
	userObjID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, apperror.ErrInvalidID
	}

	filter := bson.M{
		"user_id":    userObjID,
		"deleted_at": bson.M{"$exists": false},
	}

	findOptions := options.Find()
	if opts != nil {
		if opts.Sort != nil {
			sortDoc := bson.D{}
			for key, value := range opts.Sort {
				sortDoc = append(sortDoc, bson.E{Key: key, Value: value})
			}
			findOptions.SetSort(sortDoc)
		}
		if opts.Limit > 0 {
			findOptions.SetLimit(opts.Limit)
		}
	}

	cursor, err := r.reviewCollection.Find(ctx, filter, findOptions)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var reviews []*model.Review
	if err := cursor.All(ctx, &reviews); err != nil {
		return nil, err
	}

	return reviews, nil
}

func (r *reviewRepo) Find(ctx context.Context, filter Filter, opts *FindOptions) ([]*model.Review, int64, error) {
	mongoFilter := bson.M(filter)
	mongoFilter["deleted_at"] = bson.M{"$exists": false}

	// Get count
	total, err := r.reviewCollection.CountDocuments(ctx, mongoFilter)
	if err != nil {
		return nil, 0, err
	}

	if total == 0 {
		return []*model.Review{}, 0, nil
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

	cursor, err := r.reviewCollection.Find(ctx, mongoFilter, findOptions)
	if err != nil {
		return nil, total, err
	}
	defer cursor.Close(ctx)

	var reviews []*model.Review
	if err := cursor.All(ctx, &reviews); err != nil {
		return nil, 0, err
	}

	return reviews, total, nil
}

// Review replies
func (r *reviewRepo) AddReply(ctx context.Context, reply *model.ReviewReply) (*model.ReviewReply, error) {
	reply.ID = primitive.NewObjectID()
	reply.CreatedAt = time.Now()
	reply.UpdatedAt = time.Now()

	result, err := r.reviewReplyCollection.InsertOne(ctx, reply)
	if err != nil {
		return nil, err
	}

	if oid, ok := result.InsertedID.(primitive.ObjectID); ok {
		reply.ID = oid
	}

	// Increment reply count on review
	_ = r.IncrementReplyCount(ctx, reply.ReviewID.Hex())

	return reply, nil
}

func (r *reviewRepo) GetRepliesByReviewID(ctx context.Context, reviewID string, opts *FindOptions) ([]*model.ReviewReply, int64, error) {
	reviewObjID, err := primitive.ObjectIDFromHex(reviewID)
	if err != nil {
		return nil, 0, apperror.ErrInvalidID
	}

	filter := bson.M{"review_id": reviewObjID}

	// Get count
	total, err := r.reviewReplyCollection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	if total == 0 {
		return []*model.ReviewReply{}, 0, nil
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

	cursor, err := r.reviewReplyCollection.Find(ctx, filter, findOptions)
	if err != nil {
		return nil, total, err
	}
	defer cursor.Close(ctx)

	var replies []*model.ReviewReply
	if err := cursor.All(ctx, &replies); err != nil {
		return nil, 0, err
	}

	return replies, total, nil
}

func (r *reviewRepo) UpdateReply(ctx context.Context, reply *model.ReviewReply) error {
	reply.UpdatedAt = time.Now()

	filter := bson.M{"_id": reply.ID}
	update := bson.M{"$set": reply}

	result, err := r.reviewReplyCollection.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return mongo.ErrNoDocuments
	}
	return nil
}

func (r *reviewRepo) DeleteReply(ctx context.Context, replyID string) error {
	objID, err := primitive.ObjectIDFromHex(replyID)
	if err != nil {
		return apperror.ErrInvalidID
	}

	// Get reply to get review_id before deleting
	var reply model.ReviewReply
	err = r.reviewReplyCollection.FindOne(ctx, bson.M{"_id": objID}).Decode(&reply)
	if err != nil {
		return err
	}

	filter := bson.M{"_id": objID}
	result, err := r.reviewReplyCollection.DeleteOne(ctx, filter)
	if err != nil {
		return err
	}
	if result.DeletedCount == 0 {
		return mongo.ErrNoDocuments
	}

	return nil
}

// Stats methods
func (r *reviewRepo) CountByProductID(ctx context.Context, productID string) (int64, error) {
	productObjID, err := primitive.ObjectIDFromHex(productID)
	if err != nil {
		return 0, apperror.ErrInvalidID
	}

	filter := bson.M{
		"product_id": productObjID,
		"status":     model.ReviewStatusApproved,
		"deleted_at": bson.M{"$exists": false},
	}
	return r.reviewCollection.CountDocuments(ctx, filter)
}

func (r *reviewRepo) CountByUserID(ctx context.Context, userID string) (int64, error) {
	userObjID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return 0, apperror.ErrInvalidID
	}

	filter := bson.M{
		"user_id":    userObjID,
		"deleted_at": bson.M{"$exists": false},
	}
	return r.reviewCollection.CountDocuments(ctx, filter)
}

func (r *reviewRepo) GetAverageRating(ctx context.Context, productID string) (float64, error) {
	productObjID, err := primitive.ObjectIDFromHex(productID)
	if err != nil {
		return 0, apperror.ErrInvalidID
	}

	pipeline := mongo.Pipeline{
		{{"$match", bson.D{
			{"$and", bson.A{
				bson.D{{"product_id", productObjID}},
				bson.D{{"status", model.ReviewStatusApproved}},
				bson.D{{"deleted_at", bson.M{"$exists": false}}},
			}},
		}}},
		{{"$group", bson.M{
			"_id":            nil,
			"average_rating": bson.M{"$avg": "$rating"},
		}}},
	}

	cursor, err := r.reviewCollection.Aggregate(ctx, pipeline)
	if err != nil {
		return 0, err
	}
	defer cursor.Close(ctx)

	var result []struct {
		AverageRating float64 `bson:"average_rating"`
	}
	if err = cursor.All(ctx, &result); err != nil {
		return 0, err
	}

	if len(result) > 0 {
		return result[0].AverageRating, nil
	}
	return 0, nil
}

func (r *reviewRepo) GetRatingBreakdown(ctx context.Context, productID string) (map[int]int64, error) {
	productObjID, err := primitive.ObjectIDFromHex(productID)
	if err != nil {
		return nil, apperror.ErrInvalidID
	}

	pipeline := mongo.Pipeline{
		{{"$match", bson.M{
			"product_id": productObjID,
			"status":     model.ReviewStatusApproved,
			"deleted_at": bson.M{"$exists": false},
		}}},
		{{"$group", bson.M{
			"_id":   "$rating",
			"count": bson.M{"$sum": 1},
		}}},
	}

	cursor, err := r.reviewCollection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var results []struct {
		Rating int   `bson:"_id"`
		Count  int64 `bson:"count"`
	}
	if err = cursor.All(ctx, &results); err != nil {
		return nil, err
	}

	breakdown := make(map[int]int64)
	for _, r := range results {
		breakdown[r.Rating] = r.Count
	}
	return breakdown, nil
}

func (r *reviewRepo) GetReviewSummary(ctx context.Context, productID string) (*model.ReviewSummary, error) {
	productObjID, err := primitive.ObjectIDFromHex(productID)
	if err != nil {
		return nil, apperror.ErrInvalidID
	}

	avgRating, err := r.GetAverageRating(ctx, productID)
	if err != nil {
		return nil, err
	}

	breakdown, err := r.GetRatingBreakdown(ctx, productID)
	if err != nil {
		return nil, err
	}

	total, err := r.CountByProductID(ctx, productID)
	if err != nil {
		return nil, err
	}

	// Convert breakdown map to string keys for JSON
	breakdownStr := make(map[string]int64)
	for k, v := range breakdown {
		breakdownStr[string(rune(48+k))] = v // Convert int to string
	}

	return &model.ReviewSummary{
		ProductID:       productObjID,
		AverageRating:   avgRating,
		TotalReviews:    total,
		RatingBreakdown: breakdownStr,
	}, nil
}

func (r *reviewRepo) IncrementReplyCount(ctx context.Context, reviewID string) error {
	objID, err := primitive.ObjectIDFromHex(reviewID)
	if err != nil {
		return apperror.ErrInvalidID
	}

	filter := bson.M{"_id": objID}
	update := bson.M{
		"$inc": bson.M{"reply_count": 1},
	}

	_, err = r.reviewCollection.UpdateOne(ctx, filter, update)
	return err
}
