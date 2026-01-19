package inbox

import (
	"context"
	"encoding/json"
	"log"
	"net/url"
	"time"

	"github.com/samwho/streamdeck"
	sdcontext "github.com/samwho/streamdeck/context"
)

// buttonState holds per-button state including cached result for URL routing
type buttonState struct {
	settings any
	result   any
}

// Register sets up all Stream Deck event handlers for a service.
// This replaces the 150-line setup{Service} functions.
func Register(client *streamdeck.Client, svc Service) {
	action := client.Action(svc.ActionUUID())
	storage := map[string]*buttonState{}
	var quit chan struct{}

	action.RegisterHandler(
		streamdeck.WillAppear,
		func(ctx context.Context, client *streamdeck.Client, event streamdeck.Event) error {
			p := streamdeck.WillAppearPayload{}
			if err := json.Unmarshal(event.Payload, &p); err != nil {
				return err
			}

			settings, err := svc.ParseSettings(p.Settings)
			if err != nil {
				return err
			}
			storage[event.Context] = &buttonState{settings: settings}

			// Show loading state
			if err := SetLoading(ctx, client); err != nil {
				return logError(svc, event, err)
			}

			// Start polling
			ticker := time.NewTicker(svc.RefreshInterval())
			quit = make(chan struct{})

			go func() {
				// Initial fetch
				ctxStr := event.Context
				localCtx := sdcontext.WithContext(context.Background(), ctxStr)
				localSettings := settings

				result, fetchErr := svc.FetchResult(localCtx, localSettings)
				if state, ok := storage[ctxStr]; ok {
					state.result = result
				}
				if renderErr := svc.Render(localCtx, client, result, fetchErr); renderErr != nil {
					log.Printf("%s render error: %v", svc.LogPrefix(), renderErr)
				}

				// Periodic updates
				for {
					select {
					case <-ticker.C:
						for ctxStr, state := range storage {
							ctx := sdcontext.WithContext(context.Background(), ctxStr)
							result, fetchErr := svc.FetchResult(ctx, state.settings)
							state.result = result
							if renderErr := svc.Render(ctx, client, result, fetchErr); renderErr != nil {
								log.Printf("%s render error: %v", svc.LogPrefix(), renderErr)
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
			if quit != nil {
				close(quit)
			}

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

			settings, err := svc.ParseSettings(p.Settings)
			if err != nil {
				return err
			}

			if state, ok := storage[event.Context]; ok {
				state.settings = settings
			} else {
				storage[event.Context] = &buttonState{settings: settings}
			}

			result, fetchErr := svc.FetchResult(ctx, settings)
			if state, ok := storage[event.Context]; ok {
				state.result = result
			}
			if renderErr := svc.Render(ctx, client, result, fetchErr); renderErr != nil {
				return logError(svc, event, renderErr)
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

			settings, err := svc.ParseSettings(p.Settings)
			if err != nil {
				return err
			}

			var result any
			if state, ok := storage[event.Context]; ok {
				result = state.result
			}

			urlStr := svc.OpenURL(settings, result)
			if urlStr != "" {
				parsedURL, err := url.Parse(urlStr)
				if err != nil {
					return logError(svc, event, err)
				}
				if err := client.OpenURL(ctx, *parsedURL); err != nil {
					return logError(svc, event, err)
				}
			}

			// Refresh after click
			result, fetchErr := svc.FetchResult(ctx, settings)
			if state, ok := storage[event.Context]; ok {
				state.result = result
			}
			if renderErr := svc.Render(ctx, client, result, fetchErr); renderErr != nil {
				return logError(svc, event, renderErr)
			}

			return nil
		},
	)
}

func logError(svc Service, event streamdeck.Event, err error) error {
	log.Printf("%s[%s] %v", svc.LogPrefix(), event.Event, err)

	return err
}
