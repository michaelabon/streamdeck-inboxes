package main

import (
	"context"
	"encoding/json"
	"net/url"
	"time"

	"ca.michaelabon.inboxes/internal/gmail"
	"github.com/samwho/streamdeck"
	sdcontext "github.com/samwho/streamdeck/context"
)

func setupGmail(client *streamdeck.Client) {
	storage := map[string]gmail.Settings{}
	action := client.Action("ca.michaelabon.streamdeck-inboxes.gmail.action")
	var quit chan struct{}

	action.RegisterHandler(
		streamdeck.WillAppear,
		func(ctx context.Context, client *streamdeck.Client, event streamdeck.Event) error {
			p := streamdeck.WillAppearPayload{}
			if err := json.Unmarshal(event.Payload, &p); err != nil {
				return err
			}

			settings := gmail.Settings{}
			if err := json.Unmarshal(p.Settings, &settings); err != nil {
				return err
			}

			storage[event.Context] = settings

			// Show a loading indicator immediately
			err := setLoading(ctx, client)
			if err != nil {
				return logEventError(event, err)
			}

			ticker := time.NewTicker(gmail.RefreshInterval)
			quit = make(chan struct{})

			go func() {
				// Perform first update asynchronously
				localCtx := sdcontext.WithContext(context.Background(), event.Context)
				localSettings := settings

				err := setTitle(localCtx, client)(gmail.FetchUnseenCount(localSettings))
				if err != nil {
					fakeEventForLogging := streamdeck.Event{
						Action: "ca.michaelabon.streamdeck-inboxes.gmail.action",
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

							err := setTitle(ctx, client)(gmail.FetchUnseenCount(settings))
							if err != nil {
								fakeEventForLogging := streamdeck.Event{
									Action: "ca.michaelabon.streamdeck-inboxes.gmail.action",
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

			settings := gmail.Settings{}
			if err := json.Unmarshal(p.Settings, &settings); err != nil {
				return err
			}

			storage[event.Context] = settings

			err := setTitle(ctx, client)(gmail.FetchUnseenCount(settings))
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

			gmailUrl, err := url.Parse(
				"https://mail.google.com/mail/u/?authuser=" + settings.Username,
			)
			if err != nil {
				return err
			}

			err = client.OpenURL(ctx, *gmailUrl)
			if err != nil {
				return logEventError(event, err)
			}

			for i := 10; i < 180; i += 10 {
				go func() {
					time.Sleep(time.Duration(i) * time.Second)
					_ = setTitle(ctx, client)(gmail.FetchUnseenCount(settings))
				}()
			}

			return nil
		},
	)
}
