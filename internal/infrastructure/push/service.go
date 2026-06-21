package push

import (
	"context"
	"encoding/json"
	"log/slog"

	webpush "github.com/SherClockHolmes/webpush-go"
	"github.com/friend-diaries/backend/internal/config"
	"github.com/friend-diaries/backend/internal/infrastructure/repository"
	"github.com/google/uuid"
)

type Service struct {
	repo      *repository.PushSubscriptionRepository
	vapidPub  string
	vapidPriv string
	subject   string
	enabled   bool
}

type Payload struct {
	Title string `json:"title"`
	Body  string `json:"body"`
	URL   string `json:"url,omitempty"`
}

func NewService(repo *repository.PushSubscriptionRepository, cfg config.PushConfig) *Service {
	enabled := cfg.VAPIDPublicKey != "" && cfg.VAPIDPrivateKey != ""
	if !enabled {
		slog.Info("push notifications disabled: set VAPID_PUBLIC_KEY and VAPID_PRIVATE_KEY to enable")
	}
	return &Service{
		repo:      repo,
		vapidPub:  cfg.VAPIDPublicKey,
		vapidPriv: cfg.VAPIDPrivateKey,
		subject:   cfg.VAPIDSubject,
		enabled:   enabled,
	}
}

func (s *Service) PublicKey() string { return s.vapidPub }

func (s *Service) SendToUsers(ctx context.Context, userIDs []uuid.UUID, payload Payload) {
	if !s.enabled || len(userIDs) == 0 {
		return
	}
	subs, err := s.repo.ListByUserIDs(ctx, userIDs)
	if err != nil || len(subs) == 0 {
		return
	}

	data, _ := json.Marshal(payload)

	for _, sub := range subs {
		ws := &webpush.Subscription{
			Endpoint: sub.Endpoint,
			Keys: webpush.Keys{
				P256dh: sub.P256dh,
				Auth:   sub.Auth,
			},
		}
		resp, err := webpush.SendNotification(data, ws, &webpush.Options{
			VAPIDPublicKey:  s.vapidPub,
			VAPIDPrivateKey: s.vapidPriv,
			Subscriber:      s.subject,
			TTL:             86400,
		})
		if err != nil {
			slog.Warn("push send failed", "user_id", sub.UserID, "error", err)
			continue
		}
		resp.Body.Close()
		if resp.StatusCode == 410 || resp.StatusCode == 404 {
			_ = s.repo.Delete(ctx, sub.Endpoint, sub.UserID)
		}
	}
}
