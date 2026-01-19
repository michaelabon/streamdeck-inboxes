package marvin

import (
	"context"
	"encoding/json"
	"time"

	"ca.michaelabon.inboxes/internal/inbox"
	"github.com/samwho/streamdeck"
)

// Service implements inbox.Service for Amazing Marvin.
type Service struct{}

// Compile-time check that Service implements the interface.
var _ inbox.Service[*Settings, uint] = Service{}

func (s Service) ActionUUID() string {
	return "ca.michaelabon.streamdeck-inboxes.marvin.action"
}

func (s Service) RefreshInterval() time.Duration {
	return RefreshInterval
}

func (s Service) LogPrefix() string {
	return "[marvin]"
}

func (s Service) ParseSettings(raw json.RawMessage) (*Settings, error) {
	var settings Settings
	if err := json.Unmarshal(raw, &settings); err != nil {
		return nil, err
	}

	return &settings, nil
}

func (s Service) FetchResult(ctx context.Context, settings *Settings) (uint, error) {
	return FetchUnseenCount(settings)
}

func (s Service) Render(
	ctx context.Context,
	client *streamdeck.Client,
	result uint,
	err error,
) error {
	return inbox.RenderCount(ctx, client, result, err)
}

func (s Service) OpenURL(settings *Settings, result uint) string {
	return "https://app.amazingmarvin.com"
}
