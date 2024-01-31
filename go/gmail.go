package main

import (
	"ca.michaelabon.inboxes/internal/gmail"
	"context"
	"encoding/json"
	"net/url"
	"time"

	"github.com/samwho/streamdeck"
	sdcontext "github.com/samwho/streamdeck/context"
)

func setupGmail(client *streamdeck.Client) {
	storage := map[string]*gmail.Settings{}

	action := client.Action("ca.michaelabon.streamdeck-inboxes.gmail.action")

	action.RegisterHandler(streamdeck.WillAppear, func(ctx context.Context, client *streamdeck.Client, event streamdeck.Event) error {
		p := streamdeck.WillAppearPayload{}
		if err := json.Unmarshal(event.Payload, &p); err != nil {
			return err
		}

		settings := &gmail.Settings{}
		if err := json.Unmarshal(p.Settings, &settings); err != nil {
			return err
		}

		storage[event.Context] = settings

		err := setTitle(ctx, client)(gmail.FetchUnseenCount(settings))
		if err != nil {
			return logEventError(event, err)
		}
		return nil
	})

	action.RegisterHandler(streamdeck.WillDisappear, func(ctx context.Context, client *streamdeck.Client, event streamdeck.Event) error {
		delete(storage, event.Context)

		return nil
	})

	action.RegisterHandler(streamdeck.DidReceiveSettings, func(ctx context.Context, client *streamdeck.Client, event streamdeck.Event) error {
		p := streamdeck.DidReceiveSettingsPayload{}
		if err := json.Unmarshal(event.Payload, &p); err != nil {
			return err
		}

		settings := &gmail.Settings{}
		if err := json.Unmarshal(p.Settings, &settings); err != nil {
			return err
		}

		storage[event.Context] = settings

		err := setTitle(ctx, client)(gmail.FetchUnseenCount(settings))
		if err != nil {
			return logEventError(event, err)
		}
		return nil
	})

	action.RegisterHandler(streamdeck.KeyUp, func(ctx context.Context, client *streamdeck.Client, event streamdeck.Event) error {
		p := streamdeck.KeyUpPayload{}
		if err := json.Unmarshal(event.Payload, &p); err != nil {
			return err
		}
		settings := &gmail.Settings{}
		if err := json.Unmarshal(p.Settings, &settings); err != nil {
			return err
		}

		gmailUrl, err := url.Parse("https://mail.google.com/mail/u/")
		if err != nil {
			return err
		}

		gmailUrl.Query().Set("authuser", settings.Username)
		err = client.OpenURL(ctx, *gmailUrl)
		if err != nil {
			return logEventError(event, err)
		}

		err = setTitle(ctx, client)(gmail.FetchUnseenCount(settings))
		if err != nil {
			return logEventError(event, err)
		}
		return nil
	})

	go func() {
		for range time.Tick(gmail.RefreshInterval) {
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

		}
	}()

}
