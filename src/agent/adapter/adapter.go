package adapter

import (
	"context"
	"smithai/src/agent/protocol"
)

// Adapter defines the interface for communicating with an LLM provider.
type Adapter interface {
	// Chat sends a request to the LLM and streams responses back through streamChan.
	// The implementation must close streamChan when finished or upon error.
	Chat(ctx context.Context, req *protocol.Request, streamChan chan<- *protocol.Response) error
}
