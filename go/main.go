package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"ca.michaelabon.inboxes/internal/fastmail"
	"ca.michaelabon.inboxes/internal/gitlab"
	"ca.michaelabon.inboxes/internal/gmail"
	"ca.michaelabon.inboxes/internal/inbox"
	"ca.michaelabon.inboxes/internal/marvin"
	"ca.michaelabon.inboxes/internal/todoist"
	"ca.michaelabon.inboxes/internal/ynab"
	"github.com/samwho/streamdeck"
)

func main() {
	now := time.Now()
	fileName := fmt.Sprintf("streamdeck-inboxes-%s-*.log", now.Format("2006-01-02t15h04m05s"))
	f, err := os.CreateTemp("logs", fileName)
	if err != nil {
		log.Fatalf("error creating temp file: %v", err)
	}
	defer func(f *os.File) {
		err := f.Close()
		if err != nil {
			log.Printf("unable to close file \"%s\": %v\n", fileName, err)
		}
	}(f)
	log.SetOutput(f)

	ctx := context.Background()
	if err := run(ctx); err != nil {
		log.Printf("%v\n", err)

		return
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

func setup(client *streamdeck.Client) {
	inbox.Register(client, fastmail.Service{})
	inbox.Register(client, gitlab.Service{})
	inbox.Register(client, gmail.Service{})
	inbox.Register(client, marvin.Service{})
	inbox.Register(client, todoist.Service{})
	inbox.Register(client, ynab.Service{})
}
