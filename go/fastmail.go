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

			err := setTitle(ctx, client)(fastmail.FetchUnseenCount(settings))
			if err != nil {
				return logEventError(event, err)
			}
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

			fastmailUrl, err := url.Parse("https://app.fastmail.com/mail/Inbox") //?u=a56140cf
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

	go func() {
		for range time.Tick(fastmail.RefreshInterval) {
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
		}
	}()
}
