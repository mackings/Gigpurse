package usecase

import (
	"context"
	"time"

	"gigpurse/internal/domain"
)

type notificationUsecase struct {
	notifRepo domain.NotificationRepository
}

func NewNotificationUsecase(repo domain.NotificationRepository) domain.NotificationUsecase {
	return &notificationUsecase{
		notifRepo: repo,
	}
}

func (u *notificationUsecase) Create(ctx context.Context, userID, title, message string) (*domain.Notification, error) {
	notif := &domain.Notification{
		UserID:    userID,
		Title:     title,
		Message:   message,
		IsRead:    false,
		CreatedAt: time.Now(),
	}

	if err := u.notifRepo.Create(ctx, notif); err != nil {
		return nil, err
	}
	return notif, nil
}

func (u *notificationUsecase) ListForUser(ctx context.Context, userID string) ([]*domain.Notification, error) {
	return u.notifRepo.ListForUser(ctx, userID)
}

func (u *notificationUsecase) MarkAsRead(ctx context.Context, notifID, userID string) error {
	return u.notifRepo.MarkAsRead(ctx, notifID, userID)
}
