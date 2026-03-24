package chatui

import "context"

// Provider is the interface for AI chat backends.
// Send starts a request and returns a channel of StreamEvents.
type Provider interface {
	// Name returns the display name of the provider.
	Name() string
	// Send starts a request with the given prompt and conversation history.
	// It returns a channel on which StreamEvents will be sent until the stream ends.
	Send(ctx context.Context, history []message, text string) <-chan StreamEvent
	// Close shuts down any subprocess or SDK client held by the provider.
	Close()
}
