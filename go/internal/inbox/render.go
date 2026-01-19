package inbox

import (
	"context"
	"fmt"
	"log"
	"strconv"

	"ca.michaelabon.inboxes/internal/display"
	"github.com/samwho/streamdeck"
)

const (
	DefaultState = 0
	GoldState    = 1
)

// SetLoading displays a loading indicator on the button.
func SetLoading(ctx context.Context, client *streamdeck.Client) error {
	err := client.SetTitle(ctx, display.PadRight("..."), streamdeck.HardwareAndSoftware)
	if err != nil {
		return fmt.Errorf("could not set title: %w", err)
	}
	err = client.SetState(ctx, DefaultState)
	if err != nil {
		return fmt.Errorf("could not set state: %w", err)
	}

	return nil
}

// RenderCount is the standard renderer for single-count services.
// Use this in your Service.Render implementation for simple inbox types.
func RenderCount(ctx context.Context, client *streamdeck.Client, count uint, err error) error {
	if err != nil {
		newErr := client.SetTitle(ctx, display.PadRight("!"), streamdeck.HardwareAndSoftware)
		if newErr != nil {
			return fmt.Errorf("error setting title: %w  -- %w", newErr, err)
		}

		newErr = client.SetState(ctx, DefaultState)
		if newErr != nil {
			return fmt.Errorf("error setting state: %w  -- %w", newErr, err)
		}

		return err
	}

	if count == 0 {
		setErr := client.SetState(ctx, GoldState)
		if setErr != nil {
			log.Println("error while setting state", setErr)

			return setErr
		}
		setErr = client.SetTitle(ctx, "", streamdeck.HardwareAndSoftware)
		if setErr != nil {
			log.Println("error while setting icon title with unseen count", setErr)

			return setErr
		}
	} else {
		setErr := client.SetState(ctx, DefaultState)
		if setErr != nil {
			log.Println("error while setting state", setErr)

			return setErr
		}
		setErr = client.SetTitle(ctx, display.PadRight(strconv.FormatUint(uint64(count), 10)), streamdeck.HardwareAndSoftware)
		if setErr != nil {
			log.Println("error while setting icon title with unseen count", setErr)

			return setErr
		}
	}

	return nil
}
