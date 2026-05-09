package loop

import (
	"context"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"time"

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

		// Make a shallow copy of req to modify History safely during recursion
		currentReq := *req
		currentReq.History = make([]protocol.Message, len(req.History))
		copy(currentReq.History, req.History)

		if currentReq.UserPrompt != "" {
			currentReq.History = append(currentReq.History, protocol.Message{
				Role:    "user",
				Content: currentReq.UserPrompt,
			})
			currentReq.UserPrompt = ""
		}

		for {
			// Check if context is cancelled
			select {
			case <-ctx.Done():
				outChan <- &protocol.Response{Error: ctx.Err(), Done: true}
				return
			default:
			}

			var streamErr error
			var fullResponse protocol.Message
			fullResponse.Role = "assistant"
			var finalDoneResp *protocol.Response

			// Retry loop with exponential backoff
			for attempt := 0; attempt <= 3; attempt++ {
				adapterChan := make(chan *protocol.Response)

				fmt.Printf("[LLM Request] attempt=%d history_len=%d\n", attempt+1, len(currentReq.History))
				go func() {
					a.Adapter.Chat(ctx, &currentReq, adapterChan)
				}()

				streamErr = nil
				fullResponse = protocol.Message{Role: "assistant"}

				for resp := range adapterChan {
					if resp.Error != nil {
						streamErr = resp.Error
						break
					}

					// Accumulate full response
					fullResponse.Content += resp.Content
					if len(resp.ToolCalls) > 0 {
						fullResponse.ToolCalls = append(fullResponse.ToolCalls, resp.ToolCalls...)
					}

					if resp.Done {
						finalDoneResp = resp
					} else {
						// Only stream content out if we aren't going to loop immediately
						// without showing what we are doing. Actually, we should stream everything.
						outChan <- resp
					}
				}

				if streamErr == nil {
					break // success
				}
				// wait and retry
				delay := parseRetryDelay(streamErr)
				if delay == 0 {
					delay = time.Duration(1<<attempt) * time.Second
				}
				secs := int(math.Ceil(delay.Seconds()))
				outChan <- &protocol.Response{
					Content: fmt.Sprintf("[Rate limited. Retrying in %ds...]\n", secs),
				}
				time.Sleep(delay)
			}

			if streamErr != nil {
				outChan <- &protocol.Response{Error: streamErr, Done: true}
				return
			}

			// If no tools were called, the reasoning loop is complete!
			if len(fullResponse.ToolCalls) == 0 {
				if finalDoneResp != nil {
					outChan <- finalDoneResp
				}
				return
			}

			// We have tool calls. Let's execute them.
			currentReq.History = append(currentReq.History, fullResponse)
			toolResultsMsg := protocol.Message{Role: "user"}

			for _, call := range fullResponse.ToolCalls {
				result, err := a.Dispatcher.Dispatch(ctx, call)
				if err != nil {
					result = "Error: " + err.Error()
				}

				// Inform the user stream that a tool is being run/has run
				outChan <- &protocol.Response{
					Content: fmt.Sprintf("[\nTool:output\n%s:%s\n]\n", call.Name, result),
				}

				toolResultsMsg.ToolResults = append(toolResultsMsg.ToolResults, protocol.ToolResult{
					ID:     call.ID,
					Name:   call.Name,
					Result: result,
				})
			}

			currentReq.History = append(currentReq.History, toolResultsMsg)
			// Loop will now continue and send the updated history to the LLM
		}
	}()

	return outChan, nil
}

// parseRetryDelay extracts "retry in Xs" from error message, rounds up to whole seconds.
// Returns 0 if no retry delay found.
func parseRetryDelay(err error) time.Duration {
	re := regexp.MustCompile(`(?i)retry in ([0-9.]+)s`)
	matches := re.FindStringSubmatch(err.Error())
	if len(matches) < 2 {
		return 0
	}
	secs, parseErr := strconv.ParseFloat(matches[1], 64)
	if parseErr != nil {
		return 0
	}
	return time.Duration(math.Ceil(secs)) * time.Second
}
