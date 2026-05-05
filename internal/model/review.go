// model/review.go
package model

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Review struct {
	ID         primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	ProductID  primitive.ObjectID `bson:"product_id" json:"product_id"`
	UserID     primitive.ObjectID `bson:"user_id" json:"user_id"`
	UserName   string             `bson:"user_name" json:"user_name"` // Tên người đánh giá
	UserAvatar string             `bson:"user_avatar,omitempty" json:"user_avatar,omitempty"`

	// Review content
	Rating  int    `bson:"rating" json:"rating"` // 1-5 stars
	Title   string `bson:"title" json:"title"`
	Content string `bson:"content" json:"content"`

	// Media
	Images []Image `bson:"images,omitempty" json:"images,omitempty"`
	Videos []Video `bson:"videos,omitempty" json:"videos,omitempty"`

	// Additional info
	VerifiedPurchase bool         `bson:"verified_purchase" json:"verified_purchase"` // Mua hàng thực tế
	Status           ReviewStatus `bson:"status" json:"status"`
	ReplyCount       int          `bson:"reply_count" json:"reply_count"`

	// Timestamps
	CreatedAt time.Time  `bson:"created_at" json:"created_at"`
	UpdatedAt time.Time  `bson:"updated_at" json:"updated_at"`
	DeletedAt *time.Time `bson:"deleted_at,omitempty" json:"deleted_at,omitempty"`
}

// ReviewStatus - Trạng thái đánh giá
type ReviewStatus string

const (
	ReviewStatusPending  ReviewStatus = "pending"  // Chờ duyệt
	ReviewStatusApproved ReviewStatus = "approved" // Đã duyệt
	ReviewStatusRejected ReviewStatus = "rejected" // Bị từ chối
	ReviewStatusHidden   ReviewStatus = "hidden"   // Ẩn
)

// ReviewReply - Phản hồi/comment trên review (hỗ trợ nested replies)
type ReviewReply struct {
	ID            primitive.ObjectID  `bson:"_id,omitempty" json:"id"`
	ReviewID      primitive.ObjectID  `bson:"review_id" json:"review_id"`
	UserID        primitive.ObjectID  `bson:"user_id" json:"user_id"`
	UserName      string              `bson:"user_name" json:"user_name"`
	UserAvatar    string              `bson:"user_avatar,omitempty" json:"user_avatar,omitempty"`
	Content       string              `bson:"content" json:"content"`
	ParentReplyID *primitive.ObjectID `bson:"parent_reply_id,omitempty" json:"parent_reply_id,omitempty"` // Null = reply to review, có giá trị = reply to comment
	Replies       []ReviewReply       `bson:"replies,omitempty" json:"replies,omitempty"`                 // Nested replies
	CreatedAt     time.Time           `bson:"created_at" json:"created_at"`
	UpdatedAt     time.Time           `bson:"updated_at" json:"updated_at"`
}

// ReviewSummary - Tóm tắt đánh giá sản phẩm
type ReviewSummary struct {
	ProductID       primitive.ObjectID `bson:"_id" json:"product_id"`
	AverageRating   float64            `bson:"average_rating" json:"average_rating"`
	TotalReviews    int64              `bson:"total_reviews" json:"total_reviews"`
	RatingBreakdown map[string]int64   `bson:"rating_breakdown" json:"rating_breakdown"` // {"5": 100, "4": 50, ...}
}
