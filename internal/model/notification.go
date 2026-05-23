package model

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Notification struct {
	ID          primitive.ObjectID     `bson:"_id,omitempty" json:"id"`
	RecipientID primitive.ObjectID     `bson:"recipient_id" json:"recipient_id"`
	ActorID     primitive.ObjectID     `bson:"actor_id,omitempty" json:"actor_id,omitempty"`
	Type        NotificationType       `bson:"type,omitempty" json:"type,omitempty"`
	Message     string                 `bson:"message,omitempty" json:"message,omitempty"`
	Link        string                 `bson:"link,omitempty" json:"link,omitempty"`
	IsRead      bool                   `bson:"is_read,omitempty" json:"is_read,omitempty"`
	Metadata    map[string]interface{} `bson:"metadata,omitempty" json:"metadata,omitempty"`
	CreatedAt   time.Time              `bson:"created_at,omitempty" json:"created_at,omitempty"`
}

type NotificationType string

const (
	NotificationTypeOrderStatus   NotificationType = "order_status"
	NotificationTypePaymentStatus NotificationType = "payment_status"
	NotificationTypeOrderCanceled NotificationType = "order_canceled"
	NotificationTypeRefund        NotificationType = "refund"
	NotificationTypeSystem        NotificationType = "system"
)
