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

// NewModelRegistry creates a new ModelRegistry and loads available models.
func NewModelRegistry(refreshInterval time.Duration) *ModelRegistry {
	r := &ModelRegistry{
		Models:          []ModelTier{},
		Active:          "gemini-2.5-flash-lite", // default fallback
		RefreshInterval: refreshInterval,
	}
	r.Load()
	return r
}

// Load reads models from the .available file.
func (r *ModelRegistry) Load() {
	entries, err := availability.GetAvailable()
	if err != nil {
		log.Printf("Warning: failed to load available models: %v\n", err)
		return
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	for _, entry := range entries {
		if entry.ItemType == "model" {
			r.Models = append(r.Models, ModelTier{
				Name:         entry.ItemName, // We use displayName as Name in MarkAvailable for simplicity or just name?
				Stable:       entry.ItemName,
				Experimental: entry.ItemName,
			})
		}
	}
	if len(r.Models) > 0 {
		log.Printf("Loaded %d models from .available\n", len(r.Models))
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

	r.mu.RLock()
	for _, m := range r.Models {
		seen[m.Stable] = true
	}
	r.mu.RUnlock()

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
			if seen[m.Name] {
				// Already in registry, but let's make sure it's in newModels for the final state
				newModels = append(newModels, ModelTier{
					Name:         m.DisplayName,
					Stable:       m.Name,
					Experimental: m.Name,
				})
				continue
			}

			// New model found!
			newTier := ModelTier{
				Name:         m.DisplayName,
				Stable:       m.Name,
				Experimental: m.Name,
			}
			newModels = append(newModels, newTier)
			seen[m.Name] = true

			// Save to .available
			availability.MarkAvailable(m.Name, "model", "discovered")
		}
	}

	r.mu.Lock()
	r.Models = newModels
	r.LastRefresh = time.Now()
	r.mu.Unlock()

	log.Printf("Fetched %d models from Gemini API (updated .available if needed)\n", len(newModels))
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

