package http

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	"gigpurse/internal/domain"

	webpush "github.com/SherClockHolmes/webpush-go"
)

// pushSender delivers real OS-level Web Push notifications (distinct from
// the Hub's in-app websocket push) to every subscription a user has
// registered — one per browser/device they've enabled push on.
type pushSender struct {
	subRepo    domain.PushSubscriptionRepository
	vapidPub   string
	vapidPriv  string
	vapidEmail string
}

func NewPushSender(subRepo domain.PushSubscriptionRepository, vapidPub, vapidPriv, vapidEmail string) *pushSender {
	return &pushSender{subRepo: subRepo, vapidPub: vapidPub, vapidPriv: vapidPriv, vapidEmail: vapidEmail}
}

// enabled reports whether VAPID keys are configured — if not, push is a
// no-op (e.g. in a dev environment that hasn't generated keys yet).
func (s *pushSender) enabled() bool {
	return s != nil && s.vapidPub != "" && s.vapidPriv != ""
}

// SendToUser pushes notif to every device the user has subscribed on. This
// is fire-and-forget: it's called from a goroutine, logs failures instead
// of returning them, and prunes subscriptions the push service reports as
// gone (410) or not-found (404) — the browser dropped them, so we should
// stop trying.
func (s *pushSender) SendToUser(ctx context.Context, userID string, notif *domain.Notification) {
	if !s.enabled() {
		return
	}
	subs, err := s.subRepo.ListByUser(ctx, userID)
	if err != nil || len(subs) == 0 {
		return
	}

	payload, err := json.Marshal(map[string]string{
		"title": notif.Title,
		"body":  notif.Message,
		"link":  notif.Link,
	})
	if err != nil {
		return
	}

	for _, sub := range subs {
		resp, err := webpush.SendNotification(payload, &webpush.Subscription{
			Endpoint: sub.Endpoint,
			Keys:     webpush.Keys{P256dh: sub.P256dh, Auth: sub.Auth},
		}, &webpush.Options{
			Subscriber:      s.vapidEmail,
			VAPIDPublicKey:  s.vapidPub,
			VAPIDPrivateKey: s.vapidPriv,
			TTL:             60,
		})
		if err != nil {
			log.Printf("push send failed for user %s: %v", userID, err)
			continue
		}
		resp.Body.Close()
		if resp.StatusCode == http.StatusGone || resp.StatusCode == http.StatusNotFound {
			_ = s.subRepo.DeleteByEndpoint(ctx, userID, sub.Endpoint)
		}
	}
}
