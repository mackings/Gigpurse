package http

import (
	"context"

	"gigpurse/internal/domain"
)

// broadcastingNotificationRepository wraps a NotificationRepository so every
// created notification is also pushed in real time over the Hub, if the
// recipient is online, and over Web Push to any devices they've registered
// (online or not). This is the single seam all existing
// notification-creation call sites (job, contract, dispute, review, and
// notification usecases) get realtime + push delivery through, with zero
// changes to any of them.
type broadcastingNotificationRepository struct {
	domain.NotificationRepository
	hub  *Hub
	push *pushSender
}

func NewBroadcastingNotificationRepository(repo domain.NotificationRepository, hub *Hub, push *pushSender) domain.NotificationRepository {
	return &broadcastingNotificationRepository{NotificationRepository: repo, hub: hub, push: push}
}

func (r *broadcastingNotificationRepository) Create(ctx context.Context, notif *domain.Notification) error {
	if err := r.NotificationRepository.Create(ctx, notif); err != nil {
		return err
	}
	r.hub.Send(notif.UserID, "notification", notif)
	if r.push.enabled() {
		// Fire-and-forget on a background context: the request that
		// triggered this notification shouldn't wait on (or fail because
		// of) a push provider being slow or unreachable.
		go r.push.SendToUser(context.Background(), notif.UserID, notif)
	}
	return nil
}
