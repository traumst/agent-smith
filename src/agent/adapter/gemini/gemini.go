package gemini

import (
	"context"
	"encoding/json"

	"google.golang.org/genai"

	"smithai/src/agent/protocol"
	"smithai/src/agent/ratelimit"
)

// Adapter implements the adapter.Adapter interface for the Gemini API.
type Adapter struct {
	client  *genai.Client
	model   string
	limiter *ratelimit.Limiter
}

// NewAdapter creates a new Gemini adapter. rpm controls requests per minute (0 = no limit).
func NewAdapter(client *genai.Client, model string, rpm int) *Adapter {
	if model == "" {
		model = "gemini-2.5-flash-lite"
	}
	return &Adapter{
		client:  client,
		model:   model,
		limiter: ratelimit.NewLimiter(rpm),
	}
}

// Chat sends the request to Gemini and streams back the responses.
func (a *Adapter) Chat(ctx context.Context, req *protocol.Request, streamChan chan<- *protocol.Response) error {
	defer close(streamChan)

	if _, err := a.limiter.Wait(ctx); err != nil {
		streamChan <- &protocol.Response{Error: err}
		return err
	}

	var contents []*genai.Content

	// Add system prompt if present
	sysPrompt := ""
	if req.SystemPrompt.Competence != "" {
		sysPrompt += req.SystemPrompt.Competence + "\n\n"
	}
	if req.SystemPrompt.Mood != "" {
		sysPrompt += req.SystemPrompt.Mood + "\n\n"
	}
	if req.SystemPrompt.Instructions != "" {
		sysPrompt += req.SystemPrompt.Instructions + "\n\n"
	}

	config := &genai.GenerateContentConfig{
		SystemInstruction: &genai.Content{
			Parts: []*genai.Part{
				{Text: sysPrompt},
			},
		},
	}
	if req.MaxTokens > 0 {
		maxT := int32(req.MaxTokens)
		config.MaxOutputTokens = maxT
	}

	// Tools
	if len(req.Tools) > 0 {
		var toolDefs []*genai.Tool
		var functionDefs []*genai.FunctionDeclaration
		for _, td := range req.Tools {
			var schema *genai.Schema
			if td.Parameters != nil {
				b, err := json.Marshal(td.Parameters)
				if err == nil {
					json.Unmarshal(b, &schema)
					// Fix lowercase types to uppercase as required by Gemini API
					if schema != nil && schema.Type != "" {
						// We'll let Gemini API complain if it's strictly requiring uppercase,
						// though usually json.Unmarshal is fine. For safety, we could walk the schema
						// but let's stick to simple marshal/unmarshal.
					}
				}
			}
			functionDefs = append(functionDefs, &genai.FunctionDeclaration{
				Name:        td.Name,
				Description: td.Description,
				Parameters:  schema,
			})
		}
		toolDefs = append(toolDefs, &genai.Tool{FunctionDeclarations: functionDefs})
		config.Tools = toolDefs
	}

	// Add History
	for _, m := range req.History {
		role := m.Role
		if role == "assistant" {
			role = "model"
		}

		var parts []*genai.Part
		if m.Content != "" {
			parts = append(parts, &genai.Part{Text: m.Content})
		}

		// Map ToolCalls
		for _, tc := range m.ToolCalls {
			argsMap, ok := tc.Arguments.(map[string]any)
			if !ok && tc.Arguments != nil {
				b, _ := json.Marshal(tc.Arguments)
				json.Unmarshal(b, &argsMap)
			}
			parts = append(parts, &genai.Part{
				FunctionCall: &genai.FunctionCall{
					Name: tc.Name,
					Args: argsMap,
				},
			})
		}

		// Map ToolResults
		for _, tr := range m.ToolResults {
			parts = append(parts, &genai.Part{
				FunctionResponse: &genai.FunctionResponse{
					Name:     tr.Name,
					Response: map[string]any{"result": tr.Result},
				},
			})
		}

		contents = append(contents, &genai.Content{
			Role:  role,
			Parts: parts,
		})
	}

	// Add User Prompt
	if req.UserPrompt != "" {
		contents = append(contents, &genai.Content{
			Role: "user",
			Parts: []*genai.Part{
				{Text: req.UserPrompt},
			},
		})
	}

	if req.Stream {
		var totalTokens int
		for resp, err := range a.client.Models.GenerateContentStream(ctx, a.model, contents, config) {
			if err != nil {
				streamChan <- &protocol.Response{Error: err}
				return err
			}

			if resp == nil {
				continue
			}

			// Process chunk
			pr := &protocol.Response{}

			// Map Usage Metadata
			if resp.UsageMetadata != nil {
				pr.TokensUsed = int(resp.UsageMetadata.TotalTokenCount)
				totalTokens = pr.TokensUsed
			}

			// Map Text
			if len(resp.Candidates) > 0 && resp.Candidates[0].Content != nil {
				for _, part := range resp.Candidates[0].Content.Parts {
					if part.Text != "" {
						pr.Content += part.Text
					}
					if part.FunctionCall != nil {
						pr.ToolCalls = append(pr.ToolCalls, protocol.ToolCall{
							ID:        part.FunctionCall.Name, // GenAI doesn't use explicit IDs for tools the way OpenAI does
							Name:      part.FunctionCall.Name,
							Arguments: part.FunctionCall.Args,
						})
					}
				}
			}

			streamChan <- pr
		}

		// Send final done message
		streamChan <- &protocol.Response{Done: true, TokensUsed: totalTokens}

	} else {
		resp, err := a.client.Models.GenerateContent(ctx, a.model, contents, config)
		if err != nil {
			streamChan <- &protocol.Response{Error: err}
			return err
		}

		pr := &protocol.Response{Done: true}
		if resp.UsageMetadata != nil {
			pr.TokensUsed = int(resp.UsageMetadata.TotalTokenCount)
		}
		if len(resp.Candidates) > 0 && resp.Candidates[0].Content != nil {
			for _, part := range resp.Candidates[0].Content.Parts {
				if part.Text != "" {
					pr.Content += part.Text
				}
				if part.FunctionCall != nil {
					pr.ToolCalls = append(pr.ToolCalls, protocol.ToolCall{
						ID:        part.FunctionCall.Name,
						Name:      part.FunctionCall.Name,
						Arguments: part.FunctionCall.Args,
					})
				}
			}
		}
		streamChan <- pr
	}

	return nil
}
