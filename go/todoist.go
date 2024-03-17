package main

import (
	"context"
	"encoding/json"
	"net/url"
	"time"

	"ca.michaelabon.inboxes/internal/todoist"
	"github.com/samwho/streamdeck"
	sdcontext "github.com/samwho/streamdeck/context"
)

func setupTodoist(client *streamdeck.Client) {
	storage := map[string]*todoist.Settings{}

	action := client.Action("ca.michaelabon.streamdeck-inboxes.todoist.action")

	var quit chan struct{}

	action.RegisterHandler(
		streamdeck.WillAppear,
		func(ctx context.Context, client *streamdeck.Client, event streamdeck.Event) error {
			p := streamdeck.WillAppearPayload{}
			if err := json.Unmarshal(event.Payload, &p); err != nil {
				return err
			}

			settings := &todoist.Settings{}
			if err := json.Unmarshal(p.Settings, &settings); err != nil {
				return err
			}

			storage[event.Context] = settings

			err := setTitle(ctx, client)(todoist.FetchUnseenCount(settings))
			if err != nil {
				return logEventError(event, err)
			}

			ticker := time.NewTicker(todoist.RefreshInterval)
			quit = make(chan struct{})
			go func() {
				for {
					select {
					case <-ticker.C:
						for ctxStr, settings := range storage {
							ctx := context.Background()
							ctx = sdcontext.WithContext(ctx, ctxStr)

							err := setTitle(ctx, client)(todoist.FetchUnseenCount(settings))
							if err != nil {
								fakeEventForLogging := streamdeck.Event{
									Action: "ca.michaelabon.streamdeck-inboxes.todoist.action",
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

			settings := &todoist.Settings{}
			if err := json.Unmarshal(p.Settings, &settings); err != nil {
				return err
			}

			storage[event.Context] = settings

			err := setTitle(ctx, client)(todoist.FetchUnseenCount(settings))
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
			settings := &todoist.Settings{}
			if err := json.Unmarshal(p.Settings, &settings); err != nil {
				return err
			}

			todoistUrl, err := url.Parse("https://app.todoist.com/")
			if err != nil {
				return err
			}

			err = client.OpenURL(ctx, *todoistUrl)
			if err != nil {
				return logEventError(event, err)
			}

			err = setTitle(ctx, client)(todoist.FetchUnseenCount(settings))
			if err != nil {
				return logEventError(event, err)
			}

			return nil
		},
	)
}
