package repo

import (
	"context"
	"fmt"
	"time"

	"github.com/DucLove1/SE357-ShoppingManagement-BE/internal/apperror"
	"github.com/DucLove1/SE357-ShoppingManagement-BE/internal/config"
	"github.com/DucLove1/SE357-ShoppingManagement-BE/internal/model"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// OrderStats holds order statistics
type OrderStats struct {
	TotalOrders     int64   `bson:"total_orders"`
	PendingOrders   int64   `bson:"pending_orders"`
	ConfirmedOrders int64   `bson:"confirmed_orders"`
	PackedOrders    int64   `bson:"packed_orders"`
	ShippedOrders   int64   `bson:"shipped_orders"`
	DeliveredOrders int64   `bson:"delivered_orders"`
	CanceledOrders  int64   `bson:"canceled_orders"`
	ReturnedOrders  int64   `bson:"returned_orders"`
	RefundedOrders  int64   `bson:"refunded_orders"`
	TotalRevenue    float64 `bson:"total_revenue"`
}

type OrderRepo interface {
	// CRUD
	Create(ctx context.Context, order *model.Order) (*model.Order, error)
	Update(ctx context.Context, order *model.Order) (*model.Order, error)
	Delete(ctx context.Context, id string) error
	SoftDelete(ctx context.Context, id string) error

	// Get methods
	GetByID(ctx context.Context, id string) (*model.Order, error)
	GetByOrderNumber(ctx context.Context, orderNumber string) (*model.Order, error)
	GetByCustomerID(ctx context.Context, customerID string) ([]*model.Order, error)
	GetBySellerID(ctx context.Context, sellerID string) ([]*model.Order, error)
	GetDeletedByID(ctx context.Context, id string) (*model.Order, error)

	// Find with filter and pagination
	Find(ctx context.Context, filter Filter, opts *FindOptions) ([]*model.Order, int64, error)

	// Status updates
	UpdateStatus(ctx context.Context, id string, status model.OrderStatus, history model.StatusHistory) error
	UpdatePaymentStatus(ctx context.Context, id string, status model.PaymentStatus) error
	UpdateTracking(ctx context.Context, id string, trackingNumber, carrier string, estimatedDelivery *time.Time) error
	UpdateCancelReason(ctx context.Context, id string, reason string) error

	// Stats methods
	GetStats(ctx context.Context, sellerID *string) (*OrderStats, error)
	CountTotal(ctx context.Context) (int64, error)
	CountByStatus(ctx context.Context, status model.OrderStatus) (int64, error)
	CountByCustomerID(ctx context.Context, customerID string) (int64, error)
	CountBySellerID(ctx context.Context, sellerID string) (int64, error)
	CountCreatedAfter(ctx context.Context, since time.Time) (int64, error)
	GetRevenueBySellerID(ctx context.Context, sellerID string) (float64, error)
	GetTotalRevenue(ctx context.Context) (float64, error)

	// Order number
	GenerateOrderNumber(ctx context.Context) (string, error)
}

type orderRepo struct {
	orderCollection *mongo.Collection
}

func NewOrderRepo(db *mongo.Database) OrderRepo {
	return &orderRepo{
		orderCollection: db.Collection(config.OrderColName),
	}
}

// ==================== CRUD Methods ====================

func (r *orderRepo) Create(ctx context.Context, order *model.Order) (*model.Order, error) {
	order.CreatedAt = time.Now()
	order.UpdatedAt = time.Now()

	result, err := r.orderCollection.InsertOne(ctx, order)
	if err != nil {
		return nil, err
	}

	if oid, ok := result.InsertedID.(primitive.ObjectID); ok {
		order.ID = oid
	}

	return order, nil
}

func (r *orderRepo) Update(ctx context.Context, order *model.Order) (*model.Order, error) {
	order.UpdatedAt = time.Now()

	filter := bson.M{"_id": order.ID, "deleted_at": bson.M{"$exists": false}}
	update := bson.M{"$set": order}

	result, err := r.orderCollection.UpdateOne(ctx, filter, update)
	if err != nil {
		return nil, err
	}

	if result.MatchedCount == 0 {
		return nil, mongo.ErrNoDocuments
	}

	return order, nil
}

func (r *orderRepo) Delete(ctx context.Context, id string) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return apperror.ErrInvalidID
	}

	filter := bson.M{"_id": objectID}
	result, err := r.orderCollection.DeleteOne(ctx, filter)
	if err != nil {
		return err
	}

	if result.DeletedCount == 0 {
		return mongo.ErrNoDocuments
	}

	return nil
}

func (r *orderRepo) SoftDelete(ctx context.Context, id string) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return apperror.ErrInvalidID
	}

	filter := bson.M{"_id": objectID, "deleted_at": bson.M{"$exists": false}}
	update := bson.M{
		"$set": bson.M{
			"deleted_at": time.Now(),
			"updated_at": time.Now(),
		},
	}

	result, err := r.orderCollection.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}

	if result.MatchedCount == 0 {
		return mongo.ErrNoDocuments
	}

	return nil
}

// ==================== Get Methods ====================

func (r *orderRepo) GetByID(ctx context.Context, id string) (*model.Order, error) {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, apperror.ErrInvalidID
	}

	filter := bson.M{
		"_id":        objectID,
		"deleted_at": bson.M{"$exists": false},
	}

	var order model.Order
	err = r.orderCollection.FindOne(ctx, filter).Decode(&order)
	if err != nil {
		return nil, err
	}

	return &order, nil
}

func (r *orderRepo) GetDeletedByID(ctx context.Context, id string) (*model.Order, error) {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, apperror.ErrInvalidID
	}

	filter := bson.M{
		"_id":        objectID,
		"deleted_at": bson.M{"$exists": true},
	}

	var order model.Order
	err = r.orderCollection.FindOne(ctx, filter).Decode(&order)
	if err != nil {
		return nil, err
	}

	return &order, nil
}

func (r *orderRepo) GetByOrderNumber(ctx context.Context, orderNumber string) (*model.Order, error) {
	filter := bson.M{
		"order_number": orderNumber,
		"deleted_at":   bson.M{"$exists": false},
	}

	var order model.Order
	err := r.orderCollection.FindOne(ctx, filter).Decode(&order)
	if err != nil {
		return nil, err
	}

	return &order, nil
}

func (r *orderRepo) GetByCustomerID(ctx context.Context, customerID string) ([]*model.Order, error) {
	objectID, err := primitive.ObjectIDFromHex(customerID)
	if err != nil {
		return nil, apperror.ErrInvalidID
	}

	filter := bson.M{
		"customer_id": objectID,
		"deleted_at":  bson.M{"$exists": false},
	}

	findOptions := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}})

	cursor, err := r.orderCollection.Find(ctx, filter, findOptions)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var orders []*model.Order
	if err := cursor.All(ctx, &orders); err != nil {
		return nil, err
	}

	return orders, nil
}

func (r *orderRepo) GetBySellerID(ctx context.Context, sellerID string) ([]*model.Order, error) {
	objectID, err := primitive.ObjectIDFromHex(sellerID)
	if err != nil {
		return nil, apperror.ErrInvalidID
	}

	filter := bson.M{
		"seller_id":  objectID,
		"deleted_at": bson.M{"$exists": false},
	}

	findOptions := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}})

	cursor, err := r.orderCollection.Find(ctx, filter, findOptions)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var orders []*model.Order
	if err := cursor.All(ctx, &orders); err != nil {
		return nil, err
	}

	return orders, nil
}

// ==================== Find Method ====================

func (r *orderRepo) Find(ctx context.Context, filter Filter, opts *FindOptions) ([]*model.Order, int64, error) {
	// Add soft delete filter
	mongoFilter := bson.M(filter)
	mongoFilter["deleted_at"] = bson.M{"$exists": false}

	// Get total count
	countPipeline := mongo.Pipeline{
		{{Key: "$match", Value: mongoFilter}},
		{{Key: "$count", Value: "total"}},
	}
	cursor, err := r.orderCollection.Aggregate(ctx, countPipeline)
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
		return []*model.Order{}, 0, nil
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

	cursor, err = r.orderCollection.Find(ctx, mongoFilter, findOptions)
	if err != nil {
		return nil, total, err
	}
	defer cursor.Close(ctx)

	var orders []*model.Order
	if err := cursor.All(ctx, &orders); err != nil {
		return nil, 0, err
	}

	return orders, total, nil
}

// ==================== Status Update Methods ====================

func (r *orderRepo) UpdateStatus(ctx context.Context, id string, status model.OrderStatus, history model.StatusHistory) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return apperror.ErrInvalidID
	}

	filter := bson.M{"_id": objectID, "deleted_at": bson.M{"$exists": false}}

	setFields := bson.M{
		"status":     status,
		"updated_at": time.Now(),
	}

	// Add specific timestamp based on status
	now := time.Now()
	switch status {
	case model.OrderStatusConfirmed:
		setFields["confirmed_at"] = now
	case model.OrderStatusShipped:
		setFields["shipped_at"] = now
	case model.OrderStatusDelivered:
		setFields["delivered_at"] = now
	case model.OrderStatusCanceled:
		setFields["canceled_at"] = now
	}

	update := bson.M{
		"$set": setFields,
		"$push": bson.M{
			"status_history": history,
		},
	}

	result, err := r.orderCollection.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}

	if result.MatchedCount == 0 {
		return mongo.ErrNoDocuments
	}

	return nil
}

func (r *orderRepo) UpdatePaymentStatus(ctx context.Context, id string, status model.PaymentStatus) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return apperror.ErrInvalidID
	}

	filter := bson.M{"_id": objectID, "deleted_at": bson.M{"$exists": false}}

	setFields := bson.M{
		"payment_status": status,
		"updated_at":     time.Now(),
	}

	if status == model.PaymentStatusPaid {
		setFields["paid_at"] = time.Now()
	}

	update := bson.M{"$set": setFields}

	result, err := r.orderCollection.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}

	if result.MatchedCount == 0 {
		return mongo.ErrNoDocuments
	}

	return nil
}

func (r *orderRepo) UpdateTracking(ctx context.Context, id string, trackingNumber, carrier string, estimatedDelivery *time.Time) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return apperror.ErrInvalidID
	}

	filter := bson.M{"_id": objectID, "deleted_at": bson.M{"$exists": false}}

	update := bson.M{
		"$set": bson.M{
			"tracking_number":    trackingNumber,
			"shipping_carrier":   carrier,
			"estimated_delivery": estimatedDelivery,
			"updated_at":         time.Now(),
		},
	}

	result, err := r.orderCollection.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}

	if result.MatchedCount == 0 {
		return mongo.ErrNoDocuments
	}

	return nil
}

func (r *orderRepo) UpdateCancelReason(ctx context.Context, id string, reason string) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return apperror.ErrInvalidID
	}

	filter := bson.M{"_id": objectID, "deleted_at": bson.M{"$exists": false}}

	update := bson.M{
		"$set": bson.M{
			"cancel_reason": reason,
			"updated_at":    time.Now(),
		},
	}

	result, err := r.orderCollection.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}

	if result.MatchedCount == 0 {
		return mongo.ErrNoDocuments
	}

	return nil
}

// ==================== Stats Methods ====================

func (r *orderRepo) GetStats(ctx context.Context, sellerID *string) (*OrderStats, error) {
	matchStage := bson.M{
		"deleted_at": bson.M{"$exists": false},
	}

	if sellerID != nil {
		objectID, err := primitive.ObjectIDFromHex(*sellerID)
		if err != nil {
			return nil, apperror.ErrInvalidID
		}
		matchStage["seller_id"] = objectID
	}

	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: matchStage}},
		{{Key: "$group", Value: bson.M{
			"_id":          nil,
			"total_orders": bson.M{"$sum": 1},
			"pending_orders": bson.M{
				"$sum": bson.M{"$cond": bson.A{bson.M{"$eq": bson.A{"$status", model.OrderStatusPending}}, 1, 0}},
			},
			"confirmed_orders": bson.M{
				"$sum": bson.M{"$cond": bson.A{bson.M{"$eq": bson.A{"$status", model.OrderStatusConfirmed}}, 1, 0}},
			},
			"packed_orders": bson.M{
				"$sum": bson.M{"$cond": bson.A{bson.M{"$eq": bson.A{"$status", model.OrderStatusPacked}}, 1, 0}},
			},
			"shipped_orders": bson.M{
				"$sum": bson.M{"$cond": bson.A{bson.M{"$eq": bson.A{"$status", model.OrderStatusShipped}}, 1, 0}},
			},
			"delivered_orders": bson.M{
				"$sum": bson.M{"$cond": bson.A{bson.M{"$eq": bson.A{"$status", model.OrderStatusDelivered}}, 1, 0}},
			},
			"canceled_orders": bson.M{
				"$sum": bson.M{"$cond": bson.A{bson.M{"$eq": bson.A{"$status", model.OrderStatusCanceled}}, 1, 0}},
			},
			"returned_orders": bson.M{
				"$sum": bson.M{"$cond": bson.A{bson.M{"$eq": bson.A{"$status", model.OrderStatusReturned}}, 1, 0}},
			},
			"refunded_orders": bson.M{
				"$sum": bson.M{"$cond": bson.A{bson.M{"$eq": bson.A{"$status", model.OrderStatusRefunded}}, 1, 0}},
			},
			"total_revenue": bson.M{
				"$sum": bson.M{"$cond": bson.A{bson.M{"$eq": bson.A{"$status", model.OrderStatusDelivered}}, "$total", 0}},
			},
		}}},
	}

	cursor, err := r.orderCollection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var results []OrderStats
	if err := cursor.All(ctx, &results); err != nil {
		return nil, err
	}

	if len(results) == 0 {
		return &OrderStats{}, nil
	}

	return &results[0], nil
}

func (r *orderRepo) CountTotal(ctx context.Context) (int64, error) {
	filter := bson.M{"deleted_at": bson.M{"$exists": false}}
	return r.orderCollection.CountDocuments(ctx, filter)
}

func (r *orderRepo) CountByStatus(ctx context.Context, status model.OrderStatus) (int64, error) {
	filter := bson.M{
		"status":     status,
		"deleted_at": bson.M{"$exists": false},
	}
	return r.orderCollection.CountDocuments(ctx, filter)
}

func (r *orderRepo) CountByCustomerID(ctx context.Context, customerID string) (int64, error) {
	objectID, err := primitive.ObjectIDFromHex(customerID)
	if err != nil {
		return 0, apperror.ErrInvalidID
	}

	filter := bson.M{
		"customer_id": objectID,
		"deleted_at":  bson.M{"$exists": false},
	}
	return r.orderCollection.CountDocuments(ctx, filter)
}

func (r *orderRepo) CountBySellerID(ctx context.Context, sellerID string) (int64, error) {
	objectID, err := primitive.ObjectIDFromHex(sellerID)
	if err != nil {
		return 0, apperror.ErrInvalidID
	}

	filter := bson.M{
		"seller_id":  objectID,
		"deleted_at": bson.M{"$exists": false},
	}
	return r.orderCollection.CountDocuments(ctx, filter)
}

func (r *orderRepo) CountCreatedAfter(ctx context.Context, since time.Time) (int64, error) {
	filter := bson.M{
		"created_at": bson.M{"$gte": since},
		"deleted_at": bson.M{"$exists": false},
	}
	return r.orderCollection.CountDocuments(ctx, filter)
}

func (r *orderRepo) GetRevenueBySellerID(ctx context.Context, sellerID string) (float64, error) {
	objectID, err := primitive.ObjectIDFromHex(sellerID)
	if err != nil {
		return 0, apperror.ErrInvalidID
	}

	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{
			"seller_id":  objectID,
			"status":     model.OrderStatusDelivered,
			"deleted_at": bson.M{"$exists": false},
		}}},
		{{Key: "$group", Value: bson.M{
			"_id":           nil,
			"total_revenue": bson.M{"$sum": "$total"},
		}}},
	}

	cursor, err := r.orderCollection.Aggregate(ctx, pipeline)
	if err != nil {
		return 0, err
	}
	defer cursor.Close(ctx)

	var results []struct {
		TotalRevenue float64 `bson:"total_revenue"`
	}
	if err := cursor.All(ctx, &results); err != nil {
		return 0, err
	}

	if len(results) == 0 {
		return 0, nil
	}

	return results[0].TotalRevenue, nil
}

func (r *orderRepo) GetTotalRevenue(ctx context.Context) (float64, error) {
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{
			"status":     model.OrderStatusDelivered,
			"deleted_at": bson.M{"$exists": false},
		}}},
		{{Key: "$group", Value: bson.M{
			"_id":           nil,
			"total_revenue": bson.M{"$sum": "$total"},
		}}},
	}

	cursor, err := r.orderCollection.Aggregate(ctx, pipeline)
	if err != nil {
		return 0, err
	}
	defer cursor.Close(ctx)

	var results []struct {
		TotalRevenue float64 `bson:"total_revenue"`
	}
	if err := cursor.All(ctx, &results); err != nil {
		return 0, err
	}

	if len(results) == 0 {
		return 0, nil
	}

	return results[0].TotalRevenue, nil
}

// ==================== Order Number Generator ====================

func (r *orderRepo) GenerateOrderNumber(ctx context.Context) (string, error) {
	today := time.Now().Format("20060102")
	prefix := "ORD-" + today + "-"

	startOfDay := time.Now().Truncate(24 * time.Hour)
	filter := bson.M{
		"created_at": bson.M{"$gte": startOfDay},
	}

	count, err := r.orderCollection.CountDocuments(ctx, filter)
	if err != nil {
		return "", err
	}

	orderNumber := fmt.Sprintf("%s%05d", prefix, count+1)
	return orderNumber, nil
}
