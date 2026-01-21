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

// Register sets up all Stream Deck event handlers for a service.
// The type parameters match the service's settings and result types,
// providing compile-time type safety throughout the event handlers.
func Register[S any, R any](client *streamdeck.Client, svc Service[S, R]) {
	action := client.Action(svc.ActionUUID())

	// buttonState holds per-button state with the service's concrete types
	type buttonState struct {
		settings S
		result   R
	}
	storage := map[string]*buttonState{}
	var quit chan struct{}

	logPrefix := svc.LogPrefix()

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
				return logError(logPrefix, event, err)
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
					log.Printf("%s render error: %v", logPrefix, renderErr)
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
								log.Printf("%s render error: %v", logPrefix, renderErr)
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
				return logError(logPrefix, event, renderErr)
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

			var result R
			if state, ok := storage[event.Context]; ok {
				result = state.result
			}

			urlStr := svc.OpenURL(settings, result)
			if urlStr != "" {
				parsedURL, err := url.Parse(urlStr)
				if err != nil {
					return logError(logPrefix, event, err)
				}
				if err := client.OpenURL(ctx, *parsedURL); err != nil {
					return logError(logPrefix, event, err)
				}
			}

			// Refresh after click
			result, fetchErr := svc.FetchResult(ctx, settings)
			if state, ok := storage[event.Context]; ok {
				state.result = result
			}
			if renderErr := svc.Render(ctx, client, result, fetchErr); renderErr != nil {
				return logError(logPrefix, event, renderErr)
			}

			return nil
		},
	)

	// Check if service supports SendToPlugin handling for property inspector communication
	if handler, ok := any(svc).(SendToPluginHandler[S]); ok {
		action.RegisterHandler(
			streamdeck.SendToPlugin,
			func(ctx context.Context, client *streamdeck.Client, event streamdeck.Event) error {
				// Parse the payload to extract settings
				var payload struct {
					Settings json.RawMessage `json:"settings"`
				}
				if err := json.Unmarshal(event.Payload, &payload); err != nil {
					return err
				}

				settings, err := svc.ParseSettings(payload.Settings)
				if err != nil {
					// Send error response back to PI
					return client.SendToPropertyInspector(ctx, map[string]interface{}{
						"error": err.Error(),
					})
				}

				response, err := handler.HandleSendToPlugin(ctx, client, event.Payload, settings)
				if err != nil {
					log.Printf("%s SendToPlugin error: %v", logPrefix, err)

					return client.SendToPropertyInspector(ctx, map[string]interface{}{
						"error": err.Error(),
					})
				}

				if response != nil {
					return client.SendToPropertyInspector(ctx, response)
				}

				return nil
			},
		)
	}
}

func logError(logPrefix string, event streamdeck.Event, err error) error {
	log.Printf("%s[%s] %v", logPrefix, event.Event, err)

	return err
}
