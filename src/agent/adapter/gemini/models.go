package gemini
 
import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"google.golang.org/genai"

	"smithai/src/agent/availability"
)

// ModelTier holds information about a Gemini model tier.
type ModelTier struct {
	Name         string
	Stable       string
	Experimental string
}

// ModelRegistry holds the available model list and currently active model.
type ModelRegistry struct {
	mu              sync.RWMutex
	Models          []ModelTier
	Active          string
	RefreshInterval time.Duration
	LastRefresh     time.Time
}

// NewModelRegistry creates a new ModelRegistry.
func NewModelRegistry(refreshInterval time.Duration) *ModelRegistry {
	return &ModelRegistry{
		Models:          []ModelTier{},
		Active:          "gemini-3-flash", // default fallback
		RefreshInterval: refreshInterval,
	}
}

// Refresh fetches the list of models from the Gemini API and updates the registry.
func (r *ModelRegistry) Refresh(ctx context.Context, client *genai.Client) error {
	models, err := client.Models.List(ctx, &genai.ListModelsConfig{})
	if err != nil {
		return fmt.Errorf("failed to list models: %w", err)
	}

	var newModels []ModelTier
	seen := make(map[string]bool)

	for _, m := range models.Items {
		// Filter out unavailable models
		if !availability.IsAvailable(m.Name) {
			continue
		}

		supportsGenerate := false
		supportsEmbed := false
		for _, action := range m.SupportedActions {
			if action == "generateContent" {
				supportsGenerate = true
			}
			if action == "embedContent" {
				supportsEmbed = true
			}
		}

		if supportsGenerate || supportsEmbed {
			// Basic mapping: using m.Name as both stable and experimental for now
			// since the API list doesn't explicitly separate them like the hardcoded list did.
			// But we can try to be smart if needed.
			if seen[m.Name] {
				continue
			}
			newModels = append(newModels, ModelTier{
				Name:         m.DisplayName,
				Stable:       m.Name,
				Experimental: m.Name,
			})
			seen[m.Name] = true
		}
	}

	r.mu.Lock()
	r.Models = newModels
	r.LastRefresh = time.Now()
	r.mu.Unlock()

	log.Printf("Fetched %d models from Gemini API\n", len(newModels))
	return nil
}

// GetModels returns a copy of the available models.
func (r *ModelRegistry) GetModels() []ModelTier {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return append([]ModelTier(nil), r.Models...)
}


// SetActive sets the active model.
func (r *ModelRegistry) SetActive(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.Active = id
}

// GetActive gets the active model.
func (r *ModelRegistry) GetActive() string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.Active
}
