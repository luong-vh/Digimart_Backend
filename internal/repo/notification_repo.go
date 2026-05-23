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

type NotificationRepo interface {
	Create(ctx context.Context, notification *model.Notification) (*model.Notification, error)
	FindByRecipient(ctx context.Context, recipientID string, opts *FindOptions) ([]*model.Notification, int64, error)
	CountUnread(ctx context.Context, recipientID string) (int64, error)
	MarkAsRead(ctx context.Context, recipientID, notificationID string) error
	MarkAllAsRead(ctx context.Context, recipientID string) error
}

type notificationRepo struct {
	collection *mongo.Collection
}

func NewNotificationRepo(db *mongo.Database) NotificationRepo {
	return &notificationRepo{collection: db.Collection(config.NotificationColName)}
}

func (r *notificationRepo) Create(ctx context.Context, notification *model.Notification) (*model.Notification, error) {
	now := time.Now()
	if notification.ID.IsZero() {
		notification.ID = primitive.NewObjectID()
	}
	if notification.CreatedAt.IsZero() {
		notification.CreatedAt = now
	}

	result, err := r.collection.InsertOne(ctx, notification)
	if err != nil {
		return nil, err
	}
	if oid, ok := result.InsertedID.(primitive.ObjectID); ok {
		notification.ID = oid
	}
	return notification, nil
}

func (r *notificationRepo) FindByRecipient(ctx context.Context, recipientID string, opts *FindOptions) ([]*model.Notification, int64, error) {
	recipientObjID, err := primitive.ObjectIDFromHex(recipientID)
	if err != nil {
		return nil, 0, apperror.ErrInvalidID
	}

	filter := bson.M{"recipient_id": recipientObjID}
	total, err := r.collection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	findOptions := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}})
	if opts != nil {
		if opts.Limit > 0 {
			findOptions.SetLimit(opts.Limit)
		}
		if opts.Skip > 0 {
			findOptions.SetSkip(opts.Skip)
		}
		if len(opts.Sort) > 0 {
			sortDoc := bson.D{}
			for key, value := range opts.Sort {
				sortDoc = append(sortDoc, bson.E{Key: key, Value: value})
			}
			findOptions.SetSort(sortDoc)
		}
	}

	cursor, err := r.collection.Find(ctx, filter, findOptions)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var notifications []*model.Notification
	if err := cursor.All(ctx, &notifications); err != nil {
		return nil, 0, err
	}
	return notifications, total, nil
}

func (r *notificationRepo) CountUnread(ctx context.Context, recipientID string) (int64, error) {
	recipientObjID, err := primitive.ObjectIDFromHex(recipientID)
	if err != nil {
		return 0, apperror.ErrInvalidID
	}
	return r.collection.CountDocuments(ctx, bson.M{"recipient_id": recipientObjID, "is_read": bson.M{"$ne": true}})
}

func (r *notificationRepo) MarkAsRead(ctx context.Context, recipientID, notificationID string) error {
	recipientObjID, err := primitive.ObjectIDFromHex(recipientID)
	if err != nil {
		return apperror.ErrInvalidID
	}
	notificationObjID, err := primitive.ObjectIDFromHex(notificationID)
	if err != nil {
		return apperror.ErrInvalidID
	}

	result, err := r.collection.UpdateOne(ctx, bson.M{
		"_id":          notificationObjID,
		"recipient_id": recipientObjID,
	}, bson.M{"$set": bson.M{"is_read": true}})
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return mongo.ErrNoDocuments
	}
	return nil
}

func (r *notificationRepo) MarkAllAsRead(ctx context.Context, recipientID string) error {
	recipientObjID, err := primitive.ObjectIDFromHex(recipientID)
	if err != nil {
		return apperror.ErrInvalidID
	}
	_, err = r.collection.UpdateMany(ctx, bson.M{
		"recipient_id": recipientObjID,
		"is_read":      bson.M{"$ne": true},
	}, bson.M{"$set": bson.M{"is_read": true}})
	return err
}
