package gemini

import (
	"context"

	"google.golang.org/genai"

	"smithai/src/agent/protocol"
)

// Adapter implements the adapter.Adapter interface for the Gemini API.
type Adapter struct {
	client *genai.Client
	model  string
}

// NewAdapter creates a new Gemini adapter.
func NewAdapter(client *genai.Client, model string) *Adapter {
	if model == "" {
		model = "gemini-2.5-flash"
	}
	return &Adapter{
		client: client,
		model:  model,
	}
}

// Chat sends the request to Gemini and streams back the responses.
func (a *Adapter) Chat(ctx context.Context, req *protocol.Request, streamChan chan<- *protocol.Response) error {
	defer close(streamChan)

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
			// Basic mapping. Assuming td.Parameters is a valid map that can be cast or marshalled
			// into genai.Schema if needed, or we just pass it as generic JSON schema.
			// The GenAI SDK uses genai.Schema struct. We will need to map it.
			// For simplicity in Phase 2, we might not map full schemas deeply unless needed.
			functionDefs = append(functionDefs, &genai.FunctionDeclaration{
				Name:        td.Name,
				Description: td.Description,
				// We need to convert td.Parameters (which is `any`) to *genai.Schema
				// We will handle tool schemas properly in the dispatcher implementation.
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
		contents = append(contents, &genai.Content{
			Role: role,
			Parts: []*genai.Part{
				{Text: m.Content},
			},
		})
	}

	// Add User Prompt
	contents = append(contents, &genai.Content{
		Role: "user",
		Parts: []*genai.Part{
			{Text: req.UserPrompt},
		},
	})

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
							ID:   part.FunctionCall.Name, // GenAI doesn't use explicit IDs for tools the way OpenAI does
							Name: part.FunctionCall.Name,
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
						ID:   part.FunctionCall.Name,
						Name: part.FunctionCall.Name,
						Arguments: part.FunctionCall.Args,
					})
				}
			}
		}
		streamChan <- pr
	}

	return nil
}
