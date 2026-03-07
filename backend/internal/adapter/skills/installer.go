package skills

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// InstalledSkill describes an installed skill.
type InstalledSkill struct {
	Name     string `json:"name"`
	Path     string `json:"path"`
	Modified bool   `json:"modified"`
}

// Installer manages skill installation from GitHub tarballs.
type Installer struct {
	httpClient *http.Client
}

// NewInstaller creates a new skill installer.
func NewInstaller() *Installer {
	return &Installer{
		httpClient: &http.Client{Timeout: 60 * time.Second},
	}
}

// NewInstallerWith creates an installer with a custom http.Client (for testing).
func NewInstallerWith(c *http.Client) *Installer {
	return &Installer{httpClient: c}
}

// Install downloads a tarball, extracts skills, and installs them to destDir.
// Returns the list of installed skills.
func (inst *Installer) Install(tarballURL, destDir string) ([]InstalledSkill, error) {
	// Download to temp file.
	tmpFile, err := os.CreateTemp("", "crelay-skills-*.tar.gz")
	if err != nil {
		return nil, fmt.Errorf("create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	resp, err := inst.httpClient.Get(tarballURL)
	if err != nil {
		tmpFile.Close()
		return nil, fmt.Errorf("download tarball: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		tmpFile.Close()
		return nil, fmt.Errorf("download failed: status %d", resp.StatusCode)
	}

	if _, err := io.Copy(tmpFile, resp.Body); err != nil {
		tmpFile.Close()
		return nil, fmt.Errorf("write tarball: %w", err)
	}
	tmpFile.Close()

	// Extract to temp directory.
	extractDir, err := os.MkdirTemp("", "crelay-skills-extract-*")
	if err != nil {
		return nil, fmt.Errorf("create extract dir: %w", err)
	}
	defer os.RemoveAll(extractDir)

	if err := extractTarGz(tmpPath, extractDir); err != nil {
		return nil, fmt.Errorf("extract tarball: %w", err)
	}

	// Find skill directories (those containing SKILL.md).
	skills, err := findSkills(extractDir)
	if err != nil {
		return nil, fmt.Errorf("find skills: %w", err)
	}

	// Atomically install each skill.
	var installed []InstalledSkill
	for _, skillSrc := range skills {
		name := filepath.Base(skillSrc)
		destPath := filepath.Join(destDir, name)

		// Remove existing and rename in.
		if err := os.RemoveAll(destPath); err != nil {
			return nil, fmt.Errorf("remove existing skill %s: %w", name, err)
		}
		if err := copyDir(skillSrc, destPath); err != nil {
			return nil, fmt.Errorf("install skill %s: %w", name, err)
		}
		installed = append(installed, InstalledSkill{Name: name, Path: destPath})
	}

	return installed, nil
}

// ListInstalled returns skills found in the given directory with modification status.
func ListInstalled(skillsDir string, manifest *Manifest) []InstalledSkill {
	entries, err := os.ReadDir(skillsDir)
	if err != nil {
		return nil
	}

	modified := DetectModified(skillsDir, manifest)
	modifiedSet := map[string]bool{}
	for _, m := range modified {
		modifiedSet[m.Name] = true
	}

	var skills []InstalledSkill
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		// Only count directories that contain SKILL.md.
		skillFile := filepath.Join(skillsDir, e.Name(), "SKILL.md")
		if _, err := os.Stat(skillFile); err != nil {
			continue
		}
		skills = append(skills, InstalledSkill{
			Name:     e.Name(),
			Path:     filepath.Join(skillsDir, e.Name()),
			Modified: modifiedSet[e.Name()],
		})
	}
	return skills
}

func extractTarGz(tarPath, destDir string) error {
	f, err := os.Open(tarPath)
	if err != nil {
		return err
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		// Strip the top-level directory (GitHub tarballs have owner-repo-hash/).
		parts := strings.SplitN(hdr.Name, "/", 2)
		if len(parts) < 2 || parts[1] == "" {
			continue
		}
		relPath := parts[1]

		target := filepath.Join(destDir, relPath)
		// Prevent path traversal.
		if !strings.HasPrefix(filepath.Clean(target), filepath.Clean(destDir)) {
			continue
		}

		switch hdr.Typeflag {
		case tar.TypeDir:
			os.MkdirAll(target, 0o755)
		case tar.TypeReg:
			os.MkdirAll(filepath.Dir(target), 0o755)
			out, err := os.Create(target)
			if err != nil {
				return err
			}
			if _, err := io.Copy(out, tr); err != nil {
				out.Close()
				return err
			}
			out.Close()
		}
	}
	return nil
}

// findSkills walks the extract directory and finds directories containing SKILL.md.
func findSkills(root string) ([]string, error) {
	var skills []string
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		skillFile := filepath.Join(root, e.Name(), "SKILL.md")
		if _, err := os.Stat(skillFile); err == nil {
			skills = append(skills, filepath.Join(root, e.Name()))
		}
	}
	return skills, nil
}

func copyDir(src, dst string) error {
	return filepath.WalkDir(src, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		relPath, _ := filepath.Rel(src, path)
		target := filepath.Join(dst, relPath)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
	})
}
