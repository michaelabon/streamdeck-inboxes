package ynab

import (
	"context"
	"encoding/json"
	"time"

	"ca.michaelabon.inboxes/internal/inbox"
	"github.com/samwho/streamdeck"
)

// Result holds both the count and the next account ID for URL routing.
type Result struct {
	Count         uint
	NextAccountId string
}

// Service implements inbox.Service for YNAB (You Need A Budget).
type Service struct{}

// Compile-time check that Service implements the interface.
var _ inbox.Service[*Settings, Result] = Service{}

func (s Service) ActionUUID() string {
	return "ca.michaelabon.streamdeck-inboxes.ynab.action"
}

func (s Service) RefreshInterval() time.Duration {
	return FastRefreshInterval
}

func (s Service) LogPrefix() string {
	return "[ynab]"
}

func (s Service) ParseSettings(raw json.RawMessage) (*Settings, error) {
	var settings Settings
	if err := json.Unmarshal(raw, &settings); err != nil {
		return nil, err
	}

	return &settings, nil
}

func (s Service) FetchResult(ctx context.Context, settings *Settings) (Result, error) {
	// FetchUnseenCountAndNextAccountId modifies settings.NextAccountId as a side effect
	count, err := FetchUnseenCountAndNextAccountId(settings)
	if err != nil {
		return Result{Count: 0}, err
	}

	return Result{
		Count:         count,
		NextAccountId: settings.NextAccountId,
	}, nil
}

func (s Service) Render(
	ctx context.Context,
	client *streamdeck.Client,
	result Result,
	err error,
) error {
	return inbox.RenderCount(ctx, client, result.Count, err)
}

func (s Service) OpenURL(settings *Settings, result Result) string {
	baseURL := "https://app.ynab.com/"
	if settings.BudgetUuid == "" {
		return baseURL
	}

	url := baseURL + settings.BudgetUuid + "/accounts"

	if result.NextAccountId != "" {
		url += "/" + result.NextAccountId
	}

	return url
}
