package loop

import (
	"context"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"time"

	"agentsmith/src/agent/adapter"
	"agentsmith/src/agent/protocol"
	"agentsmith/src/agent/tools"
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

	if a.Dispatcher != nil {
		req.Tools = a.Dispatcher.Definitions()
	}

	go func() {
		defer close(outChan)
		currentReq := prepareRequest(req)

		for {
			select {
			case <-ctx.Done():
				outChan <- &protocol.Response{Error: ctx.Err(), Done: true}
				return
			default:
			}

			fullResponse, finalDoneResp, err := a.callLLM(ctx, &currentReq, outChan)
			if err != nil {
				fmt.Printf("[LLM Error] %s\n", err)
				outChan <- &protocol.Response{Error: err, Done: true}
				return
			}

			if len(fullResponse.ToolCalls) == 0 {
				if finalDoneResp != nil {
					outChan <- finalDoneResp
				}
				return
			}

			a.executeTools(ctx, &currentReq, fullResponse, outChan)
		}
	}()

	return outChan, nil
}

// prepareRequest makes a safe copy of the request with user prompt folded into history.
func prepareRequest(req *protocol.Request) protocol.Request {
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
	return currentReq
}

// callLLM sends request to the adapter with retry logic. Streams partial responses to outChan.
// Returns the accumulated response, final done response, and any error after retries exhausted.
func (a *Agent) callLLM(ctx context.Context, req *protocol.Request, outChan chan<- *protocol.Response) (protocol.Message, *protocol.Response, error) {
	var streamErr error
	var fullResponse protocol.Message
	var finalDoneResp *protocol.Response

	for attempt := 0; attempt <= 3; attempt++ {
		adapterChan := make(chan *protocol.Response)

		fmt.Printf("[LLM Request] attempt=%d history_len=%d\n", attempt+1, len(req.History))
		go func() {
			a.Adapter.Chat(ctx, req, adapterChan)
		}()

		streamErr = nil
		fullResponse = protocol.Message{Role: "assistant"}

		for resp := range adapterChan {
			if resp.Error != nil {
				streamErr = resp.Error
				break
			}

			fmt.Printf("[LLM response chunk] %s\n", resp.Content)
			fullResponse.Content += resp.Content
			if resp.Model != "" {
				fullResponse.Model = resp.Model
			}
			if len(resp.ToolCalls) > 0 {
				fmt.Printf("[LLM response toolcalls] %s\n", resp.ToolCalls)
				fullResponse.ToolCalls = append(fullResponse.ToolCalls, resp.ToolCalls...)
			}

			if resp.Done {
				finalDoneResp = resp
			} else {
				outChan <- resp
			}
		}

		if streamErr == nil {
			break
		}

		fmt.Printf("[LLM STREAM ERROR] %s\n", streamErr)
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

	return fullResponse, finalDoneResp, streamErr
}
func (a *Agent) executeTools(ctx context.Context, req *protocol.Request, response protocol.Message, outChan chan<- *protocol.Response) {
	req.History = append(req.History, response)
	toolResultsMsg := protocol.Message{Role: "user"}

	for _, call := range response.ToolCalls {
		result, err := a.Dispatcher.Dispatch(ctx, call)
		if err != nil {
			result = "Error: " + err.Error()
		}

		outChan <- &protocol.Response{
			Content: fmt.Sprintf("[\nTool:output\n%s:%s\n]\n", call.Name, result),
		}

		toolResultsMsg.ToolResults = append(toolResultsMsg.ToolResults, protocol.ToolResult{
			ID:     call.ID,
			Name:   call.Name,
			Result: result,
		})
	}

	req.History = append(req.History, toolResultsMsg)
}
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
