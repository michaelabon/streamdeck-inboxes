package inbox

import (
	"context"
	"encoding/json"
	"time"

	"github.com/samwho/streamdeck"
)

// Service defines the contract for any inbox service.
// The type parameters provide compile-time type safety:
//   - S: the settings type for this service
//   - R: the result type returned by FetchResult
type Service[S any, R any] interface {
	// ActionUUID returns the Stream Deck action identifier
	ActionUUID() string

	// RefreshInterval returns how often to poll for updates
	RefreshInterval() time.Duration

	// ParseSettings unmarshals JSON settings into the service's settings type
	ParseSettings(raw json.RawMessage) (S, error)

	// FetchResult fetches the current inbox state
	FetchResult(ctx context.Context, settings S) (R, error)

	// Render updates the Stream Deck button display
	Render(ctx context.Context, client *streamdeck.Client, result R, err error) error

	// OpenURL returns the URL to open when the button is pressed
	OpenURL(settings S, result R) string

	// LogPrefix returns the logging prefix, e.g., "[marvin]"
	LogPrefix() string
}

// SendToPluginHandler is an optional interface for services that need
// to handle property inspector communication (e.g., fetching dynamic options).
type SendToPluginHandler[S any] interface {
	// HandleSendToPlugin processes messages from the property inspector.
	// Returns a response payload to send back, or nil if no response needed.
	HandleSendToPlugin(ctx context.Context, client *streamdeck.Client,
		payload json.RawMessage, settings S) (interface{}, error)
}
