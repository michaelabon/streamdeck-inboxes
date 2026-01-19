package fastmail

import (
	"context"
	"encoding/json"
	"time"

	"ca.michaelabon.inboxes/internal/inbox"
	"github.com/samwho/streamdeck"
)

// Service implements inbox.Service for Fastmail.
type Service struct{}

func (s Service) ActionUUID() string {
	return "ca.michaelabon.streamdeck-inboxes.fastmail.action"
}

func (s Service) RefreshInterval() time.Duration {
	return RefreshInterval
}

func (s Service) LogPrefix() string {
	return "[fastmail]"
}

func (s Service) ParseSettings(raw json.RawMessage) (any, error) {
	var settings Settings
	if err := json.Unmarshal(raw, &settings); err != nil {
		return nil, err
	}

	return &settings, nil
}

func (s Service) FetchResult(ctx context.Context, settings any) (any, error) {
	set, ok := settings.(*Settings)
	if !ok {
		return uint(0), nil
	}
	// FetchUnseenCount takes value type
	return FetchUnseenCount(*set)
}

func (s Service) Render(
	ctx context.Context,
	client *streamdeck.Client,
	result any,
	err error,
) error {
	var count uint
	if result != nil {
		if c, ok := result.(uint); ok {
			count = c
		}
	}

	return inbox.RenderCount(ctx, client, count, err)
}

func (s Service) OpenURL(settings any, result any) string {
	return "https://app.fastmail.com/mail/Inbox"
}
