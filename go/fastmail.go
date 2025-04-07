package main

import (
	"context"
	"encoding/json"
	"net/url"
	"time"

	"ca.michaelabon.inboxes/internal/fastmail"
	"github.com/samwho/streamdeck"
	sdcontext "github.com/samwho/streamdeck/context"
)

func setupFastmail(client *streamdeck.Client) {
	const uuid = "ca.michaelabon.streamdeck-inboxes.fastmail.action"

	storage := map[string]fastmail.Settings{}

	action := client.Action(uuid)

	var quit chan struct{}

	action.RegisterHandler(
		streamdeck.WillAppear,
		func(ctx context.Context, client *streamdeck.Client, event streamdeck.Event) error {
			p := streamdeck.WillAppearPayload{}
			if err := json.Unmarshal(event.Payload, &p); err != nil {
				return err
			}

			settings := fastmail.Settings{}
			if err := json.Unmarshal(p.Settings, &settings); err != nil {
				return err
			}

			storage[event.Context] = settings

			// Show a loading indicator immediately
			err := setLoading(ctx, client)
			if err != nil {
				return logEventError(event, err)
			}

			ticker := time.NewTicker(fastmail.RefreshInterval)
			quit = make(chan struct{})

			go func() {
				// Perform first update asynchronously
				ctxStr := event.Context
				localCtx := context.Background()
				localCtx = sdcontext.WithContext(localCtx, ctxStr)
				localSettings := settings

				err := setTitle(localCtx, client)(fastmail.FetchUnseenCount(localSettings))
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
						for ctxStr, settings := range storage {
							ctx := context.Background()
							ctx = sdcontext.WithContext(ctx, ctxStr)

							err := setTitle(ctx, client)(fastmail.FetchUnseenCount(settings))
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
			close(quit)

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

			settings := fastmail.Settings{}
			if err := json.Unmarshal(p.Settings, &settings); err != nil {
				return err
			}

			err := setTitle(ctx, client)(fastmail.FetchUnseenCount(settings))
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
				return err
			}
			settings := fastmail.Settings{}
			if err := json.Unmarshal(p.Settings, &settings); err != nil {
				return err
			}

			fastmailUrl, err := url.Parse("https://app.fastmail.com/mail/Inbox") // ?u=a56140cf
			if err != nil {
				return err
			}

			err = client.OpenURL(ctx, *fastmailUrl)
			if err != nil {
				return logEventError(event, err)
			}

			err = setTitle(ctx, client)(fastmail.FetchUnseenCount(settings))
			if err != nil {
				return logEventError(event, err)
			}

			return nil
		},
	)
}
