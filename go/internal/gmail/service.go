package gmail

import (
	"context"
	"encoding/json"
	"net/url"
	"time"

	"ca.michaelabon.inboxes/internal/inbox"
	"github.com/samwho/streamdeck"
)

// Service implements inbox.Service for Gmail via IMAP.
type Service struct{}

// Compile-time check that Service implements the interfaces.
var (
	_ inbox.Service[*Settings, uint]       = Service{}
	_ inbox.SendToPluginHandler[*Settings] = Service{}
)

func (s Service) ActionUUID() string {
	return "ca.michaelabon.streamdeck-inboxes.gmail.action"
}

func (s Service) RefreshInterval() time.Duration {
	return RefreshInterval
}

func (s Service) LogPrefix() string {
	return "[gmail]"
}

func (s Service) ParseSettings(raw json.RawMessage) (*Settings, error) {
	var settings Settings
	if err := json.Unmarshal(raw, &settings); err != nil {
		return nil, err
	}

	return &settings, nil
}

func (s Service) FetchResult(ctx context.Context, settings *Settings) (uint, error) {
	return FetchUnseenCount(*settings)
}

func (s Service) Render(
	ctx context.Context,
	client *streamdeck.Client,
	result uint,
	err error,
) error {
	return inbox.RenderCount(ctx, client, result, err)
}

func (s Service) OpenURL(settings *Settings, result uint) string {
	base := "https://mail.google.com/mail/u/0/?authuser=" + settings.Username

	label := settings.Label
	if label == "" || label == DefaultMailbox {
		return base + "#inbox"
	}

	// Map Gmail system labels to URL fragments
	systemLabels := map[string]string{
		"[Gmail]/Starred":   "#starred",
		"[Gmail]/Sent Mail": "#sent",
		"[Gmail]/Drafts":    "#drafts",
		"[Gmail]/All Mail":  "#all",
		"[Gmail]/Spam":      "#spam",
		"[Gmail]/Trash":     "#trash",
		"[Gmail]/Important": "#imp",
	}

	if fragment, ok := systemLabels[label]; ok {
		return base + fragment
	}

	// Custom labels use #label/<name>
	return base + "#label/" + url.PathEscape(label)
}

// HandleSendToPlugin processes messages from the property inspector.
func (s Service) HandleSendToPlugin(
	ctx context.Context,
	client *streamdeck.Client,
	payload json.RawMessage,
	settings *Settings,
) (interface{}, error) {
	var request struct {
		Action string `json:"action"`
	}
	if err := json.Unmarshal(payload, &request); err != nil {
		return nil, err
	}

	switch request.Action {
	case "fetchLabels":
		labels, err := FetchLabels(*settings)
		if err != nil {
			// Return error as payload to PI, not as Go error
			//nolint:nilerr // intentionally returning nil error with error payload
			return map[string]interface{}{
				"action": "fetchLabels",
				"error":  err.Error(),
			}, nil
		}

		return map[string]interface{}{
			"action": "fetchLabels",
			"labels": labels,
		}, nil
	default:
		//nolint:nilnil // unknown actions are intentionally ignored
		return nil, nil
	}
}
