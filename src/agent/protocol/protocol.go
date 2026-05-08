package protocol

// SystemPrompt defines the agent's core instructions and persona.
type SystemPrompt struct {
	Competence   string `json:"competence"`
	Mood         string `json:"mood"`
	Instructions string `json:"instructions"`
}

// Message represents a single chat message in the conversation history.
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ToolDef represents the JSON schema definition for a tool available to the agent.
type ToolDef struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Parameters  any    `json:"parameters"` // e.g., JSON schema map
}

// ToolCall represents a specific tool invocation requested by the agent.
type ToolCall struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Arguments any    `json:"arguments"`
}

// FileDelta represents a request to change file content.
type FileDelta struct {
	Path        string `json:"path"`
	StartLine   int    `json:"startLine"`
	StartCol    int    `json:"startCol"`
	EndLine     int    `json:"endLine"`
	EndCol      int    `json:"endCol"`
	Content     string `json:"content"`
}

// Request is the input given to the agent loop to produce a response.
type Request struct {
	SystemPrompt SystemPrompt `json:"systemPrompt"`
	UserPrompt   string       `json:"userPrompt"`
	History      []Message    `json:"history"`
	Tools        []ToolDef    `json:"tools"`
	MaxTokens    int          `json:"maxTokens"`
	Stream       bool         `json:"stream"`
}

// Response is the output produced by the agent loop.
type Response struct {
	Done        bool       `json:"done"`
	Content     string     `json:"content"`
	ToolCalls   []ToolCall `json:"toolCalls"`
	FileDelta   *FileDelta `json:"fileDelta,omitempty"`
	LoadContext []string   `json:"loadContext,omitempty"`
	TokensUsed  int        `json:"tokensUsed"`
	Error       error      `json:"error,omitempty"`
}
