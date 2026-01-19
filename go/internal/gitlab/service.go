package gitlab

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"time"

	"ca.michaelabon.inboxes/internal/display"
	"ca.michaelabon.inboxes/internal/inbox"
	"github.com/samwho/streamdeck"
)

//go:embed gitlab_button_default.svg
var svgTemplate string

// Service implements inbox.Service for GitLab.
type Service struct{}

// Compile-time check that Service implements the interface.
var _ inbox.Service[*Settings, Result] = Service{}

func (s Service) ActionUUID() string {
	return "ca.michaelabon.streamdeck-inboxes.gitlab.action"
}

func (s Service) RefreshInterval() time.Duration {
	return RefreshInterval
}

func (s Service) LogPrefix() string {
	return "[gitlab]"
}

func (s Service) ParseSettings(raw json.RawMessage) (*Settings, error) {
	var settings Settings
	if err := json.Unmarshal(raw, &settings); err != nil {
		return nil, err
	}

	return &settings, nil
}

func (s Service) FetchResult(ctx context.Context, settings *Settings) (Result, error) {
	return FetchUnseenCount(settings)
}

func (s Service) Render(
	ctx context.Context,
	client *streamdeck.Client,
	result Result,
	err error,
) error {
	if err != nil {
		newErr := client.SetTitle(ctx, display.PadRight("!"), streamdeck.HardwareAndSoftware)
		if newErr != nil {
			return fmt.Errorf("error setting title: %w  -- %w", newErr, err)
		}

		newErr = client.SetState(ctx, inbox.DefaultState)
		if newErr != nil {
			return fmt.Errorf("error setting state: %w  -- %w", newErr, err)
		}

		newErr = client.SetImage(ctx, "", streamdeck.HardwareAndSoftware)
		if newErr != nil {
			return fmt.Errorf("error setting blank image: %w  -- %w", newErr, err)
		}

		return err
	}

	total := result.ToDos + result.AssignedMRs + result.ReviewMRs + result.AssignedIssues
	if total == 0 {
		_ = client.SetState(ctx, inbox.GoldState)
	} else {
		_ = client.SetState(ctx, inbox.DefaultState)
	}

	newErr := client.SetTitle(ctx, "", streamdeck.HardwareAndSoftware)
	if newErr != nil {
		return fmt.Errorf("error setting title: %w", newErr)
	}

	filledSvg := fmt.Sprintf(
		svgTemplate,
		result.AssignedIssues,
		result.AssignedIssues,
		result.AssignedMRs+result.ReviewMRs,
		result.AssignedMRs+result.ReviewMRs,
		result.ToDos,
		result.ToDos,
	)

	setErr := client.SetImage(ctx, display.EncodeSVG(filledSvg), streamdeck.HardwareAndSoftware)
	if setErr != nil {
		log.Println("[gitlab] error while setting image", setErr)

		return setErr
	}

	return nil
}

func (s Service) OpenURL(settings *Settings, result Result) string {
	if settings.Server == "" {
		return ""
	}

	gitlabURL, err := url.Parse(settings.Server)
	if err != nil {
		return settings.Server
	}

	switch {
	case result.ToDos > 0:
		gitlabURL = gitlabURL.JoinPath("/dashboard/todos")
	case result.ReviewMRs > 0:
		gitlabURL = gitlabURL.JoinPath("/dashboard/merge_requests")
		query := gitlabURL.Query()
		query.Set("reviewer_username", settings.Username)
		gitlabURL.RawQuery = query.Encode()
	case result.AssignedMRs > 0:
		gitlabURL = gitlabURL.JoinPath("/dashboard/merge_requests")
		query := gitlabURL.Query()
		query.Set("assignee_username", settings.Username)
		gitlabURL.RawQuery = query.Encode()
	case result.AssignedIssues > 0:
		gitlabURL = gitlabURL.JoinPath("/dashboard/issues")
		query := gitlabURL.Query()
		query.Set("state", "opened")
		query.Set("assignee_username[]", settings.Username)
		gitlabURL.RawQuery = query.Encode()
	default:
		gitlabURL = gitlabURL.JoinPath("/dashboard/projects/starred")
	}

	log.Printf("[gitlab] Generated URL: %s\n", gitlabURL.String())

	return gitlabURL.String()
}
