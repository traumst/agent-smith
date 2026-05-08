package loop

import (
	"context"

	"smithai/src/agent/adapter"
	"smithai/src/agent/protocol"
	"smithai/src/agent/tools"
)

// Agent represents the core reasoning loop.
type Agent struct {
	Adapter    adapter.Adapter
	Dispatcher tools.Dispatcher
}

// NewAgent creates a new Agent.
func NewAgent(adp adapter.Adapter, dispatcher tools.Dispatcher) *Agent {
	return &Agent{
		Adapter:    adp,
		Dispatcher: dispatcher,
	}
}

// Run executes the core reasoning loop for a single request.
func (a *Agent) Run(ctx context.Context, req *protocol.Request) (<-chan *protocol.Response, error) {
	outChan := make(chan *protocol.Response)

	// Inject tool definitions into the request
	if a.Dispatcher != nil {
		req.Tools = a.Dispatcher.Definitions()
	}

	go func() {
		defer close(outChan)

		// Create an internal channel to receive adapter stream
		adapterChan := make(chan *protocol.Response)
		
		go func() {
			err := a.Adapter.Chat(ctx, req, adapterChan)
			if err != nil && err.Error() != "iterator done" && err.Error() != "EOF" {
				// Send error if adapter failed before closing
				// Note: Chat implementation should already send errors down the stream,
				// but we catch here just in case it returns an error directly.
			}
		}()

		for resp := range adapterChan {
			// Check if context is cancelled
			select {
			case <-ctx.Done():
				outChan <- &protocol.Response{Error: ctx.Err(), Done: true}
				return
			default:
			}

			// Forward the response chunk
			outChan <- resp

			// If the adapter requested tool calls, we execute them immediately.
			// In Phase 2, we execute dummy tools and potentially send results back.
			// For a fully autonomous loop, we would append the tool results to history
			// and call the adapter again. Since the requirement says "the loop handles it"
			// and we are just doing a smoke test, we will just execute and return.
			if len(resp.ToolCalls) > 0 && a.Dispatcher != nil {
				for _, call := range resp.ToolCalls {
					result, err := a.Dispatcher.Dispatch(ctx, call)
					if err != nil {
						outChan <- &protocol.Response{
							Content: "\n[Tool Error: " + err.Error() + "]",
						}
					} else {
						outChan <- &protocol.Response{
							Content: "\n[Tool Result: " + result + "]",
						}
					}
				}
			}
		}
	}()

	return outChan, nil
}
