package main

import (
	"context"
	"encoding/json"
	"net/url"
	"time"

	"ca.michaelabon.inboxes/internal/ynab"
	"github.com/samwho/streamdeck"
	sdcontext "github.com/samwho/streamdeck/context"
)

func setupYnab(client *streamdeck.Client) {
	const uuid = "ca.michaelabon.streamdeck-inboxes.ynab.action"

	storage := map[string]*ynab.Settings{}

	action := client.Action(uuid)

	var quit chan struct{}

	action.RegisterHandler(
		streamdeck.WillAppear,
		func(ctx context.Context, client *streamdeck.Client, event streamdeck.Event) error {
			p := streamdeck.WillAppearPayload{}
			if err := json.Unmarshal(event.Payload, &p); err != nil {
				return err
			}

			settings := &ynab.Settings{}
			if err := json.Unmarshal(p.Settings, &settings); err != nil {
				return err
			}

			storage[event.Context] = settings

			err := setTitle(ctx, client)(ynab.FetchUnseenCountAndNextAccountId(settings))
			if err != nil {
				return logEventError(event, err)
			}

			ticker := time.NewTicker(ynab.RefreshInterval)
			quit = make(chan struct{})
			go func() {
				for {
					select {
					case <-ticker.C:
						for ctxStr, settings := range storage {
							ctx := context.Background()
							ctx = sdcontext.WithContext(ctx, ctxStr)

							err := setTitle(
								ctx,
								client,
							)(
								ynab.FetchUnseenCountAndNextAccountId(settings),
							)
							if err != nil {
								fakeEventForLogging := streamdeck.Event{
									Action: uuid,
									Event:  "tick",
								}
								_ = logEventError(fakeEventForLogging, err)
							}
						}
					case <-quit:
						ticker.Stop()
						return
					}
				}
			}()

			return nil
		},
	)

	action.RegisterHandler(
		streamdeck.WillDisappear,
		func(ctx context.Context, client *streamdeck.Client, event streamdeck.Event) error {
			delete(storage, event.Context)

			return nil
		},
	)

	action.RegisterHandler(
		streamdeck.DidReceiveSettings,
		func(ctx context.Context, client *streamdeck.Client, event streamdeck.Event) error {
			p := streamdeck.DidReceiveSettingsPayload{}
			if err := json.Unmarshal(event.Payload, &p); err != nil {
				return err
			}

			settings := &ynab.Settings{}
			if err := json.Unmarshal(p.Settings, &settings); err != nil {
				return err
			}

			err := setTitle(ctx, client)(ynab.FetchUnseenCountAndNextAccountId(settings))
			if err != nil {
				return logEventError(event, err)
			}
			return nil
		},
	)

	action.RegisterHandler(
		streamdeck.KeyUp,
		func(ctx context.Context, client *streamdeck.Client, event streamdeck.Event) error {
			p := streamdeck.DidReceiveSettingsPayload{}
			if err := json.Unmarshal(event.Payload, &p); err != nil {
				return logEventError(event, err)
			}
			settings := &ynab.Settings{}
			if err := json.Unmarshal(p.Settings, &settings); err != nil {
				return logEventError(event, err)
			}

			ynabUrl, err := url.Parse("https://app.ynab.com/")
			if err != nil {
				return logEventError(event, err)
			}
			if settings.BudgetUuid != "" {
				ynabUrl = ynabUrl.JoinPath(settings.BudgetUuid, "accounts")
			}
			if settings.NextAccountId != "" {
				ynabUrl = ynabUrl.JoinPath(settings.NextAccountId)
			}

			err = client.OpenURL(ctx, *ynabUrl)
			if err != nil {
				return logEventError(event, err)
			}

			err = setTitle(ctx, client)(ynab.FetchUnseenCountAndNextAccountId(settings))
			if err != nil {
				return logEventError(event, err)
			}

			return nil
		},
	)
}
