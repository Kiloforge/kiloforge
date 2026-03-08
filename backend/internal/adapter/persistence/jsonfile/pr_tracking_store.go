package jsonfile

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"kiloforge/internal/core/domain"
	"kiloforge/internal/core/port"
)

var _ port.PRTrackingStore = (*PRTrackingStoreAdapter)(nil)

const prTrackingFile = "pr-tracking.json"

// PRTrackingStoreAdapter wraps the file-based PR tracking functions into a port.PRTrackingStore.
type PRTrackingStoreAdapter struct {
	dataDir string
}

// NewPRTrackingStoreAdapter creates a PRTrackingStoreAdapter.
func NewPRTrackingStoreAdapter(dataDir string) *PRTrackingStoreAdapter {
	return &PRTrackingStoreAdapter{dataDir: dataDir}
}

func (a *PRTrackingStoreAdapter) LoadPRTracking(slug string) (*domain.PRTracking, error) {
	dir := filepath.Join(a.dataDir, "projects", slug)
	return LoadPRTracking(dir)
}

func (a *PRTrackingStoreAdapter) SavePRTracking(slug string, t *domain.PRTracking) error {
	dir := filepath.Join(a.dataDir, "projects", slug)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create project dir: %w", err)
	}
	return SavePRTracking(t, dir)
}

// PRTrackingPath returns the path for a project's PR tracking file.
func PRTrackingPath(dataDir, slug string) string {
	return filepath.Join(dataDir, "projects", slug, prTrackingFile)
}

// SavePRTracking writes a tracking record to the given directory.
func SavePRTracking(t *domain.PRTracking, dir string) error {
	data, err := json.MarshalIndent(t, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal pr tracking: %w", err)
	}
	return os.WriteFile(filepath.Join(dir, prTrackingFile), append(data, '\n'), 0o644)
}

// LoadPRTracking reads a tracking record from the given directory.
func LoadPRTracking(dir string) (*domain.PRTracking, error) {
	data, err := os.ReadFile(filepath.Join(dir, prTrackingFile))
	if err != nil {
		return nil, fmt.Errorf("read pr tracking: %w", err)
	}
	var t domain.PRTracking
	if err := json.Unmarshal(data, &t); err != nil {
		return nil, fmt.Errorf("parse pr tracking: %w", err)
	}
	return &t, nil
}
