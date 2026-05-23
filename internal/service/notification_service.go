package service

import (
	"context"

	"github.com/luong-vh/Digimart_Backend/internal/apperror"
	"github.com/luong-vh/Digimart_Backend/internal/dto"
	"github.com/luong-vh/Digimart_Backend/internal/model"
	"github.com/luong-vh/Digimart_Backend/internal/platform/bus"
	"github.com/luong-vh/Digimart_Backend/internal/repo"
	"github.com/luong-vh/Digimart_Backend/internal/util"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type CreateNotificationInput struct {
	RecipientID string
	ActorID     string
	Type        model.NotificationType
	Message     string
	Link        string
	Metadata    map[string]interface{}
}

type NotificationService interface {
	Create(ctx context.Context, input CreateNotificationInput) (*dto.NotificationResponse, error)
	List(recipientID string, page, pageSize int64) ([]dto.NotificationResponse, int64, error)
	CountUnread(recipientID string) (int64, error)
	MarkAsRead(recipientID, notificationID string) error
	MarkAllAsRead(recipientID string) error
}

type notificationService struct {
	notificationRepo repo.NotificationRepo
	eventBus         bus.EventBus
}

func NewNotificationService(notificationRepo repo.NotificationRepo, eventBus bus.EventBus) NotificationService {
	return &notificationService{notificationRepo: notificationRepo, eventBus: eventBus}
}

func (s *notificationService) Create(ctx context.Context, input CreateNotificationInput) (*dto.NotificationResponse, error) {
	recipientObjID, err := primitive.ObjectIDFromHex(input.RecipientID)
	if err != nil {
		return nil, apperror.ErrInvalidID
	}

	notification := &model.Notification{
		RecipientID: recipientObjID,
		Type:        input.Type,
		Message:     input.Message,
		Link:        input.Link,
		IsRead:      false,
		Metadata:    input.Metadata,
	}

	if input.ActorID != "" {
		actorObjID, err := primitive.ObjectIDFromHex(input.ActorID)
		if err != nil {
			return nil, apperror.ErrInvalidID
		}
		notification.ActorID = actorObjID
	}

	created, err := s.notificationRepo.Create(ctx, notification)
	if err != nil {
		return nil, err
	}

	response := dto.FromNotification(created)
	if s.eventBus != nil {
		s.eventBus.Publish(bus.NotificationCreatedEvent{
			RecipientID:  input.RecipientID,
			Notification: response,
		})
	}

	return &response, nil
}

func (s *notificationService) List(recipientID string, page, pageSize int64) ([]dto.NotificationResponse, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	ctx, cancel := util.NewDefaultDBContext()
	defer cancel()

	notifications, total, err := s.notificationRepo.FindByRecipient(ctx, recipientID, &repo.FindOptions{
		Skip:  (page - 1) * pageSize,
		Limit: pageSize,
		Sort:  map[string]int{"created_at": -1},
	})
	if err != nil {
		return nil, 0, err
	}

	return dto.FromNotifications(notifications), total, nil
}

func (s *notificationService) CountUnread(recipientID string) (int64, error) {
	ctx, cancel := util.NewDefaultDBContext()
	defer cancel()
	return s.notificationRepo.CountUnread(ctx, recipientID)
}

func (s *notificationService) MarkAsRead(recipientID, notificationID string) error {
	ctx, cancel := util.NewDefaultDBContext()
	defer cancel()
	if err := s.notificationRepo.MarkAsRead(ctx, recipientID, notificationID); err != nil {
		if err == mongo.ErrNoDocuments {
			return apperror.ErrNotificationNotFound
		}
		return err
	}
	return nil
}

func (s *notificationService) MarkAllAsRead(recipientID string) error {
	ctx, cancel := util.NewDefaultDBContext()
	defer cancel()
	return s.notificationRepo.MarkAllAsRead(ctx, recipientID)
}
