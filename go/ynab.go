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

const uuid = "ca.michaelabon.streamdeck-inboxes.ynab.action"

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

			// Show a loading indicator immediately
			err := setLoading(ctx, client)
			if err != nil {
				return logEventError(event, err)
			}

			ticker := time.NewTicker(ynab.FastRefreshInterval)
			quit = make(chan struct{})

			go func() {
				// Perform first update asynchronously
				localCtx := sdcontext.WithContext(context.Background(), event.Context)
				localSettings := settings

				count, err := ynab.FetchUnseenCountAndNextAccountId(localSettings)
				err = setTitle(localCtx, client)(count, err)
				if err != nil {
					fakeEventForLogging := streamdeck.Event{
						Action: uuid,
						Event:  "async_init",
					}
					_ = logEventError(fakeEventForLogging, err)
				}

				// Then start the ticker loop for periodic updates
				for {
					select {
					case <-ticker.C:
						doUpdate(storage, client, "tick")
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
			close(quit)
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
			settings := storage[event.Context]

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

			doUpdate(storage, client, event.Event)

			return nil
		},
	)
}

func doUpdate(storage map[string]*ynab.Settings, client *streamdeck.Client, event string) {
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
				Event:  event,
			}
			_ = logEventError(fakeEventForLogging, err)
		}
	}
}
