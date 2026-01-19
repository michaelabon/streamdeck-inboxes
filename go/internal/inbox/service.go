package inbox

import (
	"context"
	"encoding/json"
	"time"

	"github.com/samwho/streamdeck"
)

// Service defines the contract for any inbox service.
// Implement this interface to add a new inbox type.
type Service interface {
	// ActionUUID returns the Stream Deck action identifier
	// e.g., "ca.michaelabon.streamdeck-inboxes.marvin.action"
	ActionUUID() string

	// RefreshInterval returns how often to poll for updates
	RefreshInterval() time.Duration

	// ParseSettings unmarshals JSON settings into the service's settings type
	ParseSettings(raw json.RawMessage) (any, error)

	// FetchResult fetches the current inbox state (count or multi-count struct)
	FetchResult(ctx context.Context, settings any) (any, error)

	// Render updates the Stream Deck button display
	Render(ctx context.Context, client *streamdeck.Client, result any, err error) error

	// OpenURL returns the URL to open when the button is pressed.
	// Result is provided so services like GitLab can route to different pages.
	OpenURL(settings any, result any) string

	// LogPrefix returns the logging prefix, e.g., "[marvin]"
	LogPrefix() string
}
