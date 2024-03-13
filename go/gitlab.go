package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"time"

	"ca.michaelabon.inboxes/internal/display"
	"ca.michaelabon.inboxes/internal/gitlab"
	"github.com/samwho/streamdeck"
	sdcontext "github.com/samwho/streamdeck/context"
)

func setupGitLab(client *streamdeck.Client) {
	const uuid = "ca.michaelabon.streamdeck-inboxes.gitlab.action"

	storage := map[string]*gitlab.Settings{}

	action := client.Action(uuid)

	var quit chan struct{}

	results := gitlab.Result{}

	action.RegisterHandler(
		streamdeck.WillAppear,
		func(ctx context.Context, client *streamdeck.Client, event streamdeck.Event) error {
			var err error
			p := streamdeck.WillAppearPayload{}
			if err := json.Unmarshal(event.Payload, &p); err != nil {
				return err
			}

			settings := &gitlab.Settings{}
			if err := json.Unmarshal(p.Settings, &settings); err != nil {
				return err
			}

			storage[event.Context] = settings

			results, err = gitlab.FetchUnseenCount(settings)
			err = setGitLabImage(ctx, client)(results, err)
			if err != nil {
				return logEventError(event, err)
			}

			ticker := time.NewTicker(gitlab.RefreshInterval)
			quit = make(chan struct{})

			go func() {
				for {
					select {
					case <-ticker.C:
						for ctxStr, settings := range storage {
							ctx := context.Background()
							ctx = sdcontext.WithContext(ctx, ctxStr)

							err := setGitLabImage(ctx, client)(gitlab.FetchUnseenCount(settings))
							if err != nil {
								fakeEventForLogging := streamdeck.Event{
									Action: uuid,
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
			var err error
			p := streamdeck.DidReceiveSettingsPayload{}
			if err = json.Unmarshal(event.Payload, &p); err != nil {
				return err
			}

			settings := &gitlab.Settings{}
			if err = json.Unmarshal(p.Settings, &settings); err != nil {
				return err
			}

			storage[event.Context] = settings

			results, err = gitlab.FetchUnseenCount(settings)
			err = setGitLabImage(ctx, client)(results, err)
			if err != nil {
				return logEventError(event, err)
			}
			return nil
		},
	)

	action.RegisterHandler(
		streamdeck.KeyUp,
		func(ctx context.Context, client *streamdeck.Client, event streamdeck.Event) error {
			var err error

			settings := storage[event.Context]

			gitlabUrl, err := url.Parse(settings.Server)
			if err != nil {
				return err
			}

			if results.ToDos > 0 {
				gitlabUrl = gitlabUrl.JoinPath("/dashboard/todos")
			} else if results.ReviewMRs > 0 {
				gitlabUrl = gitlabUrl.JoinPath("/dashboard/merge_requests")
				query := gitlabUrl.Query()
				query.Set("reviewer_username", settings.Username)
				gitlabUrl.RawQuery = query.Encode()
			} else if results.AssignedMRs > 0 {
				gitlabUrl = gitlabUrl.JoinPath("/dashboard/merge_requests")
				query := gitlabUrl.Query()
				query.Set("assignee_username", settings.Username)
				gitlabUrl.RawQuery = query.Encode()
			} else if results.AssignedIssues > 0 {
				gitlabUrl = gitlabUrl.JoinPath("/dashboard/issues")
				query := gitlabUrl.Query()
				query.Set("state", "opened")
				query.Set("assignee_username[]", settings.Username)
			} else {
				gitlabUrl = gitlabUrl.JoinPath("/dashboard/projects/starred")
			}

			log.Printf("[gitlab] Generated URL: %s\n", gitlabUrl.String())

			err = client.OpenURL(ctx, *gitlabUrl)
			if err != nil {
				return logEventError(event, err)
			}

			results, err = gitlab.FetchUnseenCount(settings)
			err = setGitLabImage(ctx, client)(results, err)
			if err != nil {
				return logEventError(event, err)
			}

			return nil
		},
	)
}

//go:embed gitlab_button_default.svg
var svgTemplate string

func setGitLabImage(
	ctx context.Context,
	client *streamdeck.Client,
) func(gitlab.Result, error) error {
	return func(unseenCount gitlab.Result, origErr error) error {
		if origErr != nil {
			newErr := client.SetTitle(ctx, display.PadRight("!"), streamdeck.HardwareAndSoftware)
			if newErr != nil {
				return fmt.Errorf("error setting title: %w  -- %w", newErr, origErr)
			}

			newErr = client.SetState(ctx, defaultState)
			if newErr != nil {
				return fmt.Errorf("error setting state: %w  -- %w", newErr, origErr)
			}

			newErr = client.SetImage(ctx, "", streamdeck.HardwareAndSoftware)
			if newErr != nil {
				return fmt.Errorf("error setting blank image: %w  -- %w", newErr, origErr)
			}
			return origErr
		}

		newErr := client.SetTitle(ctx, "", streamdeck.HardwareAndSoftware)
		if newErr != nil {
			return fmt.Errorf("error setting title: %w  -- %w", newErr, origErr)
		}

		var err error

		filledSvg := fmt.Sprintf(
			svgTemplate,
			unseenCount.AssignedIssues,
			unseenCount.AssignedIssues,
			unseenCount.AssignedMRs+unseenCount.ReviewMRs,
			unseenCount.AssignedMRs+unseenCount.ReviewMRs,
			unseenCount.ToDos,
			unseenCount.ToDos,
		)

		err = client.SetImage(ctx, display.EncodeSVG(filledSvg), streamdeck.HardwareAndSoftware)
		if err != nil {
			log.Println("[gitlab] error while setting image", err)
			return err
		}

		return nil
	}
}
