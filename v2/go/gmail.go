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
	gmailStorage := map[string]*gmail.Settings{}

	gmailAction := client.Action("ca.michaelabon.streamdeck-inboxes.gmail.action")

	gmailAction.RegisterHandler(streamdeck.WillAppear, func(ctx context.Context, client *streamdeck.Client, event streamdeck.Event) error {
		p := streamdeck.WillAppearPayload{}
		if err := json.Unmarshal(event.Payload, &p); err != nil {
			return err
		}

		settings := &gmail.Settings{}
		if err := json.Unmarshal(p.Settings, &settings); err != nil {
			return err
		}

		gmailStorage[event.Context] = settings

		err := setTitle(ctx, client)(gmail.FetchUnseenCount(settings))
		if err != nil {
			return logEventError(event, err)
		}
		return nil
	})

	gmailAction.RegisterHandler(streamdeck.WillDisappear, func(ctx context.Context, client *streamdeck.Client, event streamdeck.Event) error {
		delete(gmailStorage, event.Context)

		return nil
	})

	gmailAction.RegisterHandler(streamdeck.DidReceiveSettings, func(ctx context.Context, client *streamdeck.Client, event streamdeck.Event) error {
		p := streamdeck.DidReceiveSettingsPayload{}
		if err := json.Unmarshal(event.Payload, &p); err != nil {
			return err
		}

		settings := &gmail.Settings{}
		if err := json.Unmarshal(p.Settings, &settings); err != nil {
			return err
		}

		gmailStorage[event.Context] = settings

		err := setTitle(ctx, client)(gmail.FetchUnseenCount(settings))
		if err != nil {
			return logEventError(event, err)
		}
		return nil
	})

	gmailAction.RegisterHandler(streamdeck.KeyUp, func(ctx context.Context, client *streamdeck.Client, event streamdeck.Event) error {
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
			for ctxStr, settings := range gmailStorage {
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
