package main

import (
	"context"
	"encoding/json"
	"net/url"
	"time"

	"ca.michaelabon.inboxes/internal/marvin"
	"github.com/samwho/streamdeck"
	sdcontext "github.com/samwho/streamdeck/context"
)

func setupMarvin(client *streamdeck.Client) {
	storage := map[string]*marvin.Settings{}

	action := client.Action("ca.michaelabon.streamdeck-inboxes.marvin.action")

	var quit chan struct{}

	action.RegisterHandler(
		streamdeck.WillAppear,
		func(ctx context.Context, client *streamdeck.Client, event streamdeck.Event) error {
			p := streamdeck.WillAppearPayload{}
			if err := json.Unmarshal(event.Payload, &p); err != nil {
				return err
			}

			settings := &marvin.Settings{}
			if err := json.Unmarshal(p.Settings, &settings); err != nil {
				return err
			}

			storage[event.Context] = settings

			err := setTitle(ctx, client)(marvin.FetchUnseenCount(settings))
			if err != nil {
				return logEventError(event, err)
			}

			ticker := time.NewTicker(marvin.RefreshInterval)
			quit = make(chan struct{})
			go func() {
				for {
					select {
					case <-ticker.C:
						for ctxStr, settings := range storage {
							ctx := context.Background()
							ctx = sdcontext.WithContext(ctx, ctxStr)

							err := setTitle(ctx, client)(marvin.FetchUnseenCount(settings))
							if err != nil {
								fakeEventForLogging := streamdeck.Event{
									Action: "ca.michaelabon.streamdeck-inboxes.marvin.action",
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

			settings := &marvin.Settings{}
			if err := json.Unmarshal(p.Settings, &settings); err != nil {
				return err
			}

			storage[event.Context] = settings

			err := setTitle(ctx, client)(marvin.FetchUnseenCount(settings))
			if err != nil {
				return logEventError(event, err)
			}

			return nil
		},
	)

	action.RegisterHandler(
		streamdeck.KeyUp,
		func(ctx context.Context, client *streamdeck.Client, event streamdeck.Event) error {
			p := streamdeck.KeyUpPayload{}
			if err := json.Unmarshal(event.Payload, &p); err != nil {
				return err
			}
			settings := &marvin.Settings{}
			if err := json.Unmarshal(p.Settings, &settings); err != nil {
				return err
			}

			marvinUrl, err := url.Parse("https://app.amazingmarvin.com")
			if err != nil {
				return err
			}

			err = client.OpenURL(ctx, *marvinUrl)
			if err != nil {
				return logEventError(event, err)
			}

			err = setTitle(ctx, client)(marvin.FetchUnseenCount(settings))
			if err != nil {
				return logEventError(event, err)
			}

			return nil
		},
	)
}
