package skills

import (
	"crypto/sha256"
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

//go:embed embedded/*
var embeddedSkills embed.FS

// RequiredSkill describes a skill needed for a specific operation.
type RequiredSkill struct {
	Name   string // e.g., "kf-developer"
	Reason string // e.g., "required for agent developer spawning"
}

// SkillStatus describes the installation state of a required skill.
type SkillStatus struct {
	RequiredSkill
	Installed bool   // true if SKILL.md exists in global or local dir
	Current   bool   // true if installed hash matches embedded hash
	Location  string // "global", "local", or "" if not installed
}

// CheckRequired verifies that all required skills are installed.
// Checks all provided directories (global, local, project, etc.) for each skill.
// Returns list of missing skills (not installed in any directory).
func CheckRequired(required []RequiredSkill, dirs ...string) []RequiredSkill {
	var missing []RequiredSkill
	for _, r := range required {
		found := false
		for _, dir := range dirs {
			if skillExists(r.Name, dir) {
				found = true
				break
			}
		}
		if !found {
			missing = append(missing, r)
		}
	}
	return missing
}

// CheckStatus returns detailed status for each required skill, including
// whether it's installed and whether it matches the embedded version.
// Directories are checked in order; the first match determines the location label.
// The label for each directory is its position: "global" for index 0, "local" for index 1,
// "project" for index 2, and "dir-N" for subsequent indices.
func CheckStatus(required []RequiredSkill, dirs ...string) []SkillStatus {
	labels := []string{"global", "local", "project"}
	statuses := make([]SkillStatus, len(required))
	for i, r := range required {
		statuses[i] = SkillStatus{RequiredSkill: r}

		for j, dir := range dirs {
			if skillExists(r.Name, dir) {
				statuses[i].Installed = true
				if j < len(labels) {
					statuses[i].Location = labels[j]
				} else {
					statuses[i].Location = fmt.Sprintf("dir-%d", j)
				}
				statuses[i].Current = hashMatches(r.Name, dir)
				break
			}
		}
	}
	return statuses
}

// RequiredSkillsForRole returns the skills needed for a given agent role.
func RequiredSkillsForRole(role string) []RequiredSkill {
	switch role {
	case "developer":
		return []RequiredSkill{
			{Name: "kf-developer", Reason: "required for developer agent spawning"},
		}
	case "reviewer":
		return []RequiredSkill{
			{Name: "kf-reviewer", Reason: "required for reviewer agent spawning"},
		}
	case "interactive":
		return []RequiredSkill{
			{Name: "kf-interactive", Reason: "required for interactive agent spawning"},
		}
	case "architect":
		return []RequiredSkill{
			{Name: "kf-architect", Reason: "required for architect agent spawning"},
		}
	case "advisor-product":
		return []RequiredSkill{
			{Name: "kf-advisor-product", Reason: "required for product advisor agent spawning"},
		}
	case "advisor-reliability":
		return []RequiredSkill{
			{Name: "kf-advisor-reliability", Reason: "required for reliability advisor agent spawning"},
		}
	case "setup":
		return []RequiredSkill{
			{Name: "kf-setup", Reason: "required for project setup"},
		}
	case "conflict-resolver":
		return []RequiredSkill{
			{Name: "kf-conflict-resolver", Reason: "required for conflict resolver agent spawning"},
		}
	default:
		return nil
	}
}

// SkillCommandForRole returns the slash command prefix for a role.
// Returns empty string if the role has no skill command (e.g., "interactive").
func SkillCommandForRole(role string) string {
	switch role {
	case "architect":
		return "/kf-architect"
	case "advisor-product":
		return "/kf-advisor-product"
	case "advisor-reliability":
		return "/kf-advisor-reliability"
	case "interactive":
		return "/kf-interactive"
	case "conflict-resolver":
		return "/kf-conflict-resolver"
	default:
		return ""
	}
}

// InstallEmbedded extracts an embedded skill to the given destination directory.
// Returns the path where the skill was installed.
func InstallEmbedded(skillName, destDir string) (string, error) {
	srcDir := filepath.Join("embedded", skillName)
	if _, err := embeddedSkills.ReadDir(srcDir); err != nil {
		return "", fmt.Errorf("skill %q not found in embedded assets: %w", skillName, err)
	}

	destPath := filepath.Join(destDir, skillName)

	// Remove existing to ensure clean install.
	_ = os.RemoveAll(destPath)

	// Walk the embedded tree and extract all files.
	err := fs.WalkDir(embeddedSkills, srcDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		// Compute relative path from srcDir.
		rel, _ := filepath.Rel(srcDir, path)
		target := filepath.Join(destPath, rel)

		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, readErr := embeddedSkills.ReadFile(path)
		if readErr != nil {
			return fmt.Errorf("read %s: %w", path, readErr)
		}
		return os.WriteFile(target, data, 0o644)
	})
	if err != nil {
		return "", fmt.Errorf("extract skill %q: %w", skillName, err)
	}

	return destPath, nil
}

// ListEmbedded returns the names of all skills available in the embedded assets.
func ListEmbedded() []string {
	entries, err := embeddedSkills.ReadDir("embedded")
	if err != nil {
		return nil
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() {
			names = append(names, e.Name())
		}
	}
	return names
}

// AllRequiredSkills returns required skills for all known roles.
func AllRequiredSkills() []RequiredSkill {
	var all []RequiredSkill
	seen := map[string]bool{}
	for _, role := range []string{"developer", "reviewer", "interactive", "architect", "advisor-product", "advisor-reliability", "setup", "conflict-resolver"} {
		for _, r := range RequiredSkillsForRole(role) {
			if !seen[r.Name] {
				seen[r.Name] = true
				all = append(all, r)
			}
		}
	}
	return all
}

// InstallAllEmbedded installs all embedded skills to destDir, skipping those
// that are already installed with matching hashes. Returns the names of
// skills that were installed or updated.
func InstallAllEmbedded(destDir string) ([]string, error) {
	names := ListEmbedded()
	var installed []string
	for _, name := range names {
		if hashMatches(name, destDir) {
			continue
		}
		if _, err := InstallEmbedded(name, destDir); err != nil {
			return installed, fmt.Errorf("install %s: %w", name, err)
		}
		installed = append(installed, name)
	}
	return installed, nil
}

// skillExists checks whether a skill directory contains SKILL.md.
func skillExists(name, dir string) bool {
	if dir == "" {
		return false
	}
	skillFile := filepath.Join(dir, name, "SKILL.md")
	_, err := os.Stat(skillFile)
	return err == nil
}

// hashMatches compares the SHA-256 of the installed SKILL.md against the embedded one.
func hashMatches(name, dir string) bool {
	installedPath := filepath.Join(dir, name, "SKILL.md")
	installedData, err := os.ReadFile(installedPath)
	if err != nil {
		return false
	}

	embeddedPath := filepath.Join("embedded", name, "SKILL.md")
	embeddedData, err := fs.ReadFile(embeddedSkills, embeddedPath)
	if err != nil {
		return false
	}

	return sha256.Sum256(installedData) == sha256.Sum256(embeddedData)
}
