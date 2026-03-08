package skills

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// Manifest tracks installed skill versions and file checksums.
type Manifest struct {
	Version   string            `json:"version"`
	Checksums map[string]string `json:"checksums"` // "skill-name/file" → sha256
}

// ManifestPath returns the path to the skills manifest file.
func ManifestPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".kiloforge", "skills-manifest.json")
}

// LoadManifest reads the manifest from disk, returning an empty manifest if not found.
func LoadManifest() (*Manifest, error) {
	data, err := os.ReadFile(ManifestPath())
	if err != nil {
		if os.IsNotExist(err) {
			return &Manifest{Checksums: map[string]string{}}, nil
		}
		return nil, fmt.Errorf("read manifest: %w", err)
	}
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parse manifest: %w", err)
	}
	if m.Checksums == nil {
		m.Checksums = map[string]string{}
	}
	return &m, nil
}

// Save writes the manifest to disk.
func (m *Manifest) Save() error {
	dir := filepath.Dir(ManifestPath())
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create manifest dir: %w", err)
	}
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal manifest: %w", err)
	}
	return os.WriteFile(ManifestPath(), data, 0o644)
}

// ModifiedSkill describes a skill with locally modified files.
type ModifiedSkill struct {
	Name  string
	Files []string
}

// DetectModified compares current files against stored checksums.
func DetectModified(skillsDir string, manifest *Manifest) []ModifiedSkill {
	// Group checksums by skill name.
	skillFiles := map[string][]string{}
	for key := range manifest.Checksums {
		parts := filepath.SplitList(key)
		if len(parts) == 0 {
			// Use first path component as skill name.
			name := key
			if idx := findSep(key); idx >= 0 {
				name = key[:idx]
			}
			skillFiles[name] = append(skillFiles[name], key)
		}
	}

	// Simpler approach: iterate all checksummed files.
	var modified []ModifiedSkill
	bySkill := map[string][]string{}

	for relPath, expectedHash := range manifest.Checksums {
		absPath := filepath.Join(skillsDir, relPath)
		currentHash, err := hashFile(absPath)
		if err != nil || currentHash != expectedHash {
			skillName := relPath
			if idx := findSep(relPath); idx >= 0 {
				skillName = relPath[:idx]
			}
			bySkill[skillName] = append(bySkill[skillName], relPath)
		}
	}

	for name, files := range bySkill {
		modified = append(modified, ModifiedSkill{Name: name, Files: files})
	}
	return modified
}

// ComputeChecksums walks skillsDir and computes SHA-256 for each file.
func ComputeChecksums(skillsDir string) (map[string]string, error) {
	checksums := map[string]string{}
	err := filepath.WalkDir(skillsDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		relPath, _ := filepath.Rel(skillsDir, path)
		hash, hashErr := hashFile(path)
		if hashErr != nil {
			return hashErr
		}
		checksums[relPath] = hash
		return nil
	})
	return checksums, err
}

func hashFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

func findSep(s string) int {
	for i, c := range s {
		if c == '/' || c == filepath.Separator {
			return i
		}
	}
	return -1
}

// SkillsDir returns the default skills directory.
func SkillsDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".claude", "skills")
}
