package main

import (
	"context"
	"encoding/json"
	"log"
	"net/url"
	"os"
	"time"

	"ca.michaelabon.inboxes/internal/fastmail"
	"ca.michaelabon.inboxes/internal/gmail"

	"github.com/samwho/streamdeck"
)

func main() {
	fileName := "streamdeck-inboxes.log"
	f, err := os.CreateTemp("logs", fileName)
	if err != nil {
		log.Fatalf("error creating temp file: %v", err)
	}
	defer func(f *os.File) {
		err := f.Close()
		if err != nil {
			log.Printf("unable to close file “%s”: %v\n", fileName, err)
		}
	}(f)
	log.SetOutput(f)

	ctx := context.Background()
	if err := run(ctx); err != nil {
		log.Fatalf("%v\n", err)
	}
}

func run(ctx context.Context) error {
	params, err := streamdeck.ParseRegistrationParams(os.Args)
	if err != nil {
		return err
	}

	client := streamdeck.NewClient(ctx, params)
	setup(client)

	return client.Run()
}

type GmailStorage struct {
	Ctx      context.Context
	Client   *streamdeck.Client
	Settings *gmail.Settings
}

type FastmailStorage struct {
	Ctx      context.Context
	Client   *streamdeck.Client
	Settings *fastmail.Settings
}

func setup(client *streamdeck.Client) {
	gmailStorage := GmailStorage{}
	fastmailStorage := FastmailStorage{}

	gmailAction := client.Action("ca.michaelabon.streamdeck-inboxes.gmail.action")

	gmailAction.RegisterHandler(streamdeck.WillAppear, func(ctx context.Context, client *streamdeck.Client, event streamdeck.Event) error {
		p := streamdeck.WillAppearPayload{}
		if err := json.Unmarshal(event.Payload, &p); err != nil {
			return err
		}
		if err := json.Unmarshal(p.Settings, &gmailStorage.Settings); err != nil {
			return err
		}

		gmailStorage.Ctx = ctx
		gmailStorage.Client = client

		return gmail.FetchAndUpdate(client, ctx, gmailStorage.Settings)
	})

	gmailAction.RegisterHandler(streamdeck.DidReceiveSettings, func(ctx context.Context, client *streamdeck.Client, event streamdeck.Event) error {
		p := streamdeck.DidReceiveSettingsPayload{}
		if err := json.Unmarshal(event.Payload, &p); err != nil {
			return err
		}
		if err := json.Unmarshal(p.Settings, &gmailStorage.Settings); err != nil {
			return err
		}

		gmailStorage.Ctx = ctx
		gmailStorage.Client = client

		return gmail.FetchAndUpdate(client, ctx, gmailStorage.Settings)
	})

	gmailAction.RegisterHandler(streamdeck.KeyUp, func(ctx context.Context, client *streamdeck.Client, event streamdeck.Event) error {
		p := streamdeck.KeyUpPayload{}
		if err := json.Unmarshal(event.Payload, &p); err != nil {
			return err
		}
		if err := json.Unmarshal(p.Settings, &gmailStorage.Settings); err != nil {
			return err
		}

		gmailStorage.Ctx = ctx
		gmailStorage.Client = client

		gmailUrl, err := url.Parse("https://mail.google.com/mail/u/")
		if err != nil {
			return err
		}

		gmailUrl.Query().Set("authuser", gmailStorage.Settings.Username)
		err = client.OpenURL(ctx, *gmailUrl)
		if err != nil {
			return err
		}

		return nil
	})

	fastmailAction := client.Action("ca.michaelabon.streamdeck-inboxes.fastmail.action")

	fastmailAction.RegisterHandler(streamdeck.WillAppear, func(ctx context.Context, client *streamdeck.Client, event streamdeck.Event) error {
		p := streamdeck.WillAppearPayload{}
		if err := json.Unmarshal(event.Payload, &p); err != nil {
			return err
		}
		if err := json.Unmarshal(p.Settings, &fastmailStorage.Settings); err != nil {
			return err
		}
		fastmailStorage.Ctx = ctx
		fastmailStorage.Client = client

		return fastmail.FetchAndUpdate(client, ctx, fastmailStorage.Settings)
	})

	fastmailAction.RegisterHandler(streamdeck.DidReceiveSettings, func(ctx context.Context, client *streamdeck.Client, event streamdeck.Event) error {
		p := streamdeck.DidReceiveSettingsPayload{}
		if err := json.Unmarshal(event.Payload, &p); err != nil {
			return err
		}
		if err := json.Unmarshal(p.Settings, &fastmailStorage.Settings); err != nil {
			return err
		}

		log.Println("[fastmail] New api token received", fastmailStorage.Settings.ApiToken)

		fastmailStorage.Ctx = ctx
		fastmailStorage.Client = client

		return fastmail.FetchAndUpdate(client, ctx, fastmailStorage.Settings)
	})

	go func() {
		for range time.Tick(gmail.RefreshInterval) {
			if gmailStorage.Ctx == nil || gmailStorage.Client == nil {
				return
			}

			_ = gmail.FetchAndUpdate(gmailStorage.Client, gmailStorage.Ctx, gmailStorage.Settings)
		}
	}()

	go func() {
		for range time.Tick(fastmail.RefreshInterval) {
			if fastmailStorage.Ctx == nil || fastmailStorage.Client == nil {
				return
			}

			_ = fastmail.FetchAndUpdate(fastmailStorage.Client, fastmailStorage.Ctx, fastmailStorage.Settings)
		}
	}()
}
