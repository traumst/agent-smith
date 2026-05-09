package gemini

// ModelTier holds information about a Gemini model tier.
type ModelTier struct {
	Name         string
	Stable       string
	Experimental string
}

// ModelRegistry holds the available model list and currently active model.
type ModelRegistry struct {
	Models []ModelTier
	Active string
}

// NewModelRegistry creates a new ModelRegistry with supported Gemini models.
func NewModelRegistry() *ModelRegistry {
	return &ModelRegistry{
		Models: []ModelTier{
			{Name: "Gemini 2.5 Flash", Stable: "gemini-2.5-flash", Experimental: "gemini-2.5-flash-preview"},
			{Name: "Gemini 2.5 Flash Lite", Stable: "gemini-2.5-flash-lite", Experimental: "gemini-2.5-flash-lite-preview"},
			{Name: "Gemini 3 Flash", Stable: "gemini-3-flash", Experimental: "gemini-3-flash-preview"},
			{Name: "Gemini 3.1 Flash Lite", Stable: "gemini-3.1-flash-lite", Experimental: "gemini-3.1-flash-lite-preview"},
			{Name: "Gemma 4 26B", Stable: "gemma-4-26b-it", Experimental: "gemma-4-26b-it-preview"},
			{Name: "Gemma 4 31B", Stable: "gemma-4-31b-it", Experimental: "gemma-4-31b-it-preview"},
			{Name: "Gemma 3 27B", Stable: "gemma-3-27b-it", Experimental: "gemma-3-27b-it-preview"},
			// the models below here cannot be selected by the user
			{Name: "Gemini Embedding 1", Stable: "gemini-embedding-001", Experimental: "gemini-embedding-001-preview"},
			{Name: "Gemini Embedding 2", Stable: "gemini-embedding-2", Experimental: "gemini-embedding-2-preview"},
		},
		Active: "gemini-3-flash", // default
	}
}

// SetActive sets the active model.
func (r *ModelRegistry) SetActive(id string) {
	r.Active = id
}

// GetActive gets the active model.
func (r *ModelRegistry) GetActive() string {
	return r.Active
}
