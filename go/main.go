package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"

	"ca.michaelabon.inboxes/internal/display"

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

func setup(client *streamdeck.Client) {
	setupFastmail(client)
	setupGmail(client)
	setupMarvin(client)
	setupTodoist(client)
	setupYnab(client)
}

func logEventError(event streamdeck.Event, err error) error {
	log.Printf("[%s][%s] %v\n", event.Action, event.Event, err)
	return err
}

const (
	defaultState = 0
	goldState    = 1
)

func setTitle(ctx context.Context, client *streamdeck.Client) func(uint, error) error {
	return func(unseenCount uint, origErr error) error {
		if origErr != nil {
			newErr := client.SetTitle(ctx, display.PadRight("!"), streamdeck.HardwareAndSoftware)
			if newErr != nil {
				return fmt.Errorf("error setting title: %w  -- %w", newErr, origErr)
			}

			newErr = client.SetState(ctx, defaultState)
			if newErr != nil {
				return fmt.Errorf("error setting state: %w  -- %w", newErr, origErr)
			}
			return origErr
		}

		var err error
		if unseenCount == 0 {
			err = client.SetState(ctx, goldState)
			if err != nil {
				log.Println("error while setting state", err)
				return err
			}
			err = client.SetTitle(ctx, "", streamdeck.HardwareAndSoftware)
			if err != nil {
				log.Println("error while setting icon title with unseen count", err)
				return err
			}
		} else {
			err = client.SetState(ctx, defaultState)
			if err != nil {
				log.Println("error while setting state", err)
				return err
			}
			err = client.SetTitle(ctx, display.PadRight(strconv.Itoa(int(unseenCount))), streamdeck.HardwareAndSoftware)
			if err != nil {
				log.Println("error while setting icon title with unseen count", err)
				return err
			}
		}

		return nil
	}
}
