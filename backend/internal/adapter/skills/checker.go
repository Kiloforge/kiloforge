package skills

import (
	"os"
	"path/filepath"
)

// RequiredSkill describes a skill needed for a specific operation.
type RequiredSkill struct {
	Name   string // e.g., "conductor-developer"
	Reason string // e.g., "required for agent developer spawning"
}

// CheckRequired verifies that all required skills are installed.
// Checks both global (~/.claude/skills/) and local (.claude/skills/) directories.
// Returns list of missing skills.
func CheckRequired(required []RequiredSkill, globalDir, localDir string) []RequiredSkill {
	var missing []RequiredSkill
	for _, r := range required {
		if !skillExists(r.Name, globalDir) && !skillExists(r.Name, localDir) {
			missing = append(missing, r)
		}
	}
	return missing
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
