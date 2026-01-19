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

func (s Service) ActionUUID() string {
	return "ca.michaelabon.streamdeck-inboxes.ynab.action"
}

func (s Service) RefreshInterval() time.Duration {
	return FastRefreshInterval
}

func (s Service) LogPrefix() string {
	return "[ynab]"
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
		return Result{Count: 0}, nil
	}

	// FetchUnseenCountAndNextAccountId modifies settings.NextAccountId as a side effect
	count, err := FetchUnseenCountAndNextAccountId(set)
	if err != nil {
		return Result{Count: 0}, err
	}

	return Result{
		Count:         count,
		NextAccountId: set.NextAccountId,
	}, nil
}

func (s Service) Render(
	ctx context.Context,
	client *streamdeck.Client,
	result any,
	err error,
) error {
	var count uint
	if result != nil {
		if r, ok := result.(Result); ok {
			count = r.Count
		}
	}

	return inbox.RenderCount(ctx, client, count, err)
}

func (s Service) OpenURL(settings any, result any) string {
	set, ok := settings.(*Settings)
	if !ok {
		return "https://app.ynab.com/"
	}

	baseURL := "https://app.ynab.com/"
	if set.BudgetUuid == "" {
		return baseURL
	}

	url := baseURL + set.BudgetUuid + "/accounts"

	// Use NextAccountId from result if available
	if result != nil {
		if r, ok := result.(Result); ok && r.NextAccountId != "" {
			url += "/" + r.NextAccountId
		}
	}

	return url
}
