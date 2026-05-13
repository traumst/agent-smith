package gemini

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	serviceusage "google.golang.org/api/serviceusage/v1beta1"
	"google.golang.org/genai"

	"agentsmith/src/agent/availability"
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
func NewModelRegistry(activeModel string, refreshInterval time.Duration) *ModelRegistry {
	r := &ModelRegistry{
		Models:          []ModelTier{},
		Active:          activeModel,
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
	projectID := os.Getenv("PROJECT_ID")
	quotaModels := getQuotaModels(ctx, projectID)

	models, err := client.Models.List(ctx, &genai.ListModelsConfig{})
	if err != nil {
		return fmt.Errorf("failed to list models: %w", err)
	}

	r.mu.RLock()
	seen := make(map[string]bool)
	for _, m := range r.Models {
		seen[m.Stable] = true
	}
	r.mu.RUnlock()

	var newModels []ModelTier
	for _, m := range models.Items {
		if !availability.IsAvailable(m.Name) {
			continue
		}

		if !isTextModel(m, projectID, quotaModels) {
			continue
		}

		if !isGenerateModel(m) {
			continue
		}

		newModels = append(newModels, ModelTier{
			Name:         m.DisplayName,
			Stable:       m.Name,
			Experimental: m.Name,
		})

		if !seen[m.Name] {
			availability.MarkAvailable(m.Name, "model", availability.ReasonDiscovered)
			seen[m.Name] = true
		}
	}

	r.mu.Lock()
	r.Models = newModels
	r.LastRefresh = time.Now()
	r.mu.Unlock()

	log.Printf("Fetched %d models from Gemini API (filtered by category and quota)\n", len(newModels))
	return nil
}

func getQuotaModels(ctx context.Context, projectID string) map[string]bool {
	quotaModels := make(map[string]bool)
	if projectID == "" {
		return quotaModels
	}

	svc, err := serviceusage.NewService(ctx)
	if err != nil {
		log.Printf("Warning: failed to create service usage client: %v\n", err)
		return quotaModels
	}

	parent := fmt.Sprintf("projects/%s/services/aiplatform.googleapis.com", projectID)
	resp, err := svc.Services.ConsumerQuotaMetrics.List(parent).Do()
	if err != nil {
		log.Printf("Warning: failed to list quota metrics: %v\n", err)
		return quotaModels
	}

	for _, metric := range resp.Metrics {
		if !strings.Contains(metric.DisplayName, "Text-out models") {
			continue
		}

		for _, limit := range metric.ConsumerQuotaLimits {
			for _, bucket := range limit.QuotaBuckets {
				if bucket.EffectiveLimit > 0 {
					parts := strings.Split(metric.DisplayName, " - ")
					if len(parts) >= 3 {
						quotaModels[parts[2]] = true
					}
				}
			}
		}
	}
	return quotaModels
}

func isTextModel(m *genai.Model, projectID string, quotaModels map[string]bool) bool {
	if projectID != "" {
		for qm := range quotaModels {
			if strings.Contains(m.DisplayName, qm) || strings.Contains(qm, m.DisplayName) {
				return true
			}
		}
		return false
	}

	// heuristic fallback
	return !strings.Contains(m.Name, "-tts") &&
		!strings.Contains(m.Name, "-image") &&
		!strings.Contains(m.Name, "-clip") &&
		!strings.Contains(m.Name, "-embedding") &&
		!strings.Contains(m.Name, "robotics") &&
		!strings.Contains(m.Name, "deep-research")
}

func isGenerateModel(m *genai.Model) bool {
	for _, action := range m.SupportedActions {
		if action == "generateContent" {
			return true
		}
	}
	return false
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
