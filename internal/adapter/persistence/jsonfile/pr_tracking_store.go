package jsonfile

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"crelay/internal/core/domain"
)

const prTrackingFile = "pr-tracking.json"

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
