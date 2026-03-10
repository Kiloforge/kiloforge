package service

import (
	"fmt"
	"sort"
	"strings"

	"kiloforge/internal/core/domain"
	"kiloforge/internal/core/port"
	"kiloforge/pkg/kf"
)

// Compile-time check that TrackReaderImpl satisfies port.TrackReader.
var _ port.TrackReader = (*TrackReaderImpl)(nil)

// TrackReaderImpl implements port.TrackReader using the kf SDK.
type TrackReaderImpl struct{}

// NewTrackReader creates a new TrackReaderImpl.
func NewTrackReader() *TrackReaderImpl {
	return &TrackReaderImpl{}
}

func (r *TrackReaderImpl) DiscoverTracks(projectDir string) ([]port.TrackEntry, error) {
	client := kf.NewClientFromProject(projectDir)
	entries, err := client.ListTracks()
	if err != nil {
		return nil, fmt.Errorf("list tracks: %w", err)
	}

	// Load deps graph and conflicts (best-effort — files may not exist).
	depsGraph, _ := client.GetDepsGraph()
	allConflicts, _ := client.GetConflicts()

	// Build completed set for deps_met calculation.
	completed := make(map[string]bool, len(entries))
	for _, e := range entries {
		if e.Status == kf.StatusCompleted {
			completed[e.ID] = true
		}
	}

	result := make([]port.TrackEntry, len(entries))
	for i, e := range entries {
		entry := port.TrackEntry{
			ID:     e.ID,
			Title:  e.Title,
			Status: mapKFStatusToPort(e.Status),
		}
		if deps := depsGraph.GetDeps(e.ID); len(deps) > 0 {
			entry.DepsCount = len(deps)
			for _, d := range deps {
				if completed[d] {
					entry.DepsMet++
				}
			}
		}
		entry.ConflictCount = len(kf.FindConflicts(allConflicts, e.ID))
		result[i] = entry
	}
	return result, nil
}

// DiscoverTracksPaginated returns a paginated, optionally status-filtered list of tracks.
func (r *TrackReaderImpl) DiscoverTracksPaginated(projectDir string, opts domain.PageOpts, statuses ...string) (domain.Page[port.TrackEntry], error) {
	opts.Normalize()

	all, err := r.DiscoverTracks(projectDir)
	if err != nil {
		return domain.Page[port.TrackEntry]{}, err
	}

	if len(statuses) > 0 {
		set := make(map[string]bool, len(statuses))
		for _, s := range statuses {
			set[s] = true
		}
		var filtered []port.TrackEntry
		for _, e := range all {
			if set[e.Status] {
				filtered = append(filtered, e)
			}
		}
		all = filtered
	}

	sort.Slice(all, func(i, j int) bool { return all[i].ID < all[j].ID })
	total := len(all)

	if opts.Cursor != "" {
		cur := domain.DecodeCursor(opts.Cursor)
		if cur.SortVal != "" {
			idx := sort.Search(len(all), func(i int) bool { return all[i].ID > cur.SortVal })
			all = all[idx:]
		}
	}

	var nextCursor string
	if len(all) > opts.Limit {
		last := all[opts.Limit-1]
		nextCursor = domain.EncodeCursor(last.ID, last.ID)
		all = all[:opts.Limit]
	}

	return domain.Page[port.TrackEntry]{
		Items:      all,
		NextCursor: nextCursor,
		TotalCount: total,
	}, nil
}

func (r *TrackReaderImpl) GetTrackDetail(projectDir, trackID string) (*port.TrackDetail, error) {
	client := kf.NewClientFromProject(projectDir)
	track, err := client.GetTrack(trackID)
	if err != nil {
		return nil, fmt.Errorf("track %q not found", trackID)
	}

	progress := track.Progress()

	detail := &port.TrackDetail{
		ID:        track.ID,
		Title:     track.Title,
		Status:    mapKFStatusToPort(track.Status),
		Type:      track.Type,
		Spec:      renderSpec(track.Spec),
		Plan:      renderPlan(track.Plan),
		Phases:    port.ProgressCount{Total: progress.TotalPhases, Completed: progress.CompletedPhases},
		Tasks:     port.ProgressCount{Total: progress.TotalTasks, Completed: progress.CompletedTasks},
		CreatedAt: track.Created,
		UpdatedAt: track.Updated,
	}

	// Resolve dependencies with metadata from registry.
	depsGraph, _ := client.GetDepsGraph()
	if depIDs := depsGraph.GetDeps(trackID); len(depIDs) > 0 {
		registry := buildRegistryMap(client)
		detail.Dependencies = make([]port.TrackDependency, len(depIDs))
		for i, depID := range depIDs {
			dep := port.TrackDependency{ID: depID}
			if entry, ok := registry[depID]; ok {
				dep.Title = entry.Title
				dep.Status = mapKFStatusToPort(entry.Status)
			}
			detail.Dependencies[i] = dep
		}
	}

	// Resolve conflicts with metadata from registry.
	conflictPairs, _ := client.GetConflictsForTrack(trackID)
	if len(conflictPairs) > 0 {
		registry := buildRegistryMap(client)
		detail.Conflicts = make([]port.TrackConflict, len(conflictPairs))
		for i, cp := range conflictPairs {
			otherID := cp.TrackB
			if otherID == trackID {
				otherID = cp.TrackA
			}
			tc := port.TrackConflict{
				TrackID: otherID,
				Risk:    cp.Risk,
				Note:    cp.Note,
			}
			if entry, ok := registry[otherID]; ok {
				tc.TrackTitle = entry.Title
			}
			detail.Conflicts[i] = tc
		}
	}

	// Parse agent register from extra fields.
	detail.AgentRegister = parseAgentRegister(track.Extra)

	return detail, nil
}

// buildRegistryMap returns a map of track ID → registry entry for fast lookup.
func buildRegistryMap(client *kf.Client) map[string]kf.TrackEntry {
	entries, err := client.ListTracks()
	if err != nil {
		return nil
	}
	m := make(map[string]kf.TrackEntry, len(entries))
	for _, e := range entries {
		m[e.ID] = e
	}
	return m
}

// parseAgentRegister extracts agent identity data from the track's extra map.
// Returns nil if no register data is present.
func parseAgentRegister(extra map[string]interface{}) *port.AgentRegister {
	if len(extra) == 0 {
		return nil
	}

	var reg port.AgentRegister
	hasData := false

	if raw, ok := extra["created_by"]; ok {
		if identity := parseAgentIdentity(raw, "created_at"); identity != nil {
			reg.CreatedBy = identity
			hasData = true
		}
	}
	if raw, ok := extra["claim"]; ok {
		if identity := parseAgentIdentity(raw, "claimed_at"); identity != nil {
			reg.ClaimedBy = identity
			hasData = true
		}
	}

	if !hasData {
		return nil
	}
	return &reg
}

// parseAgentIdentity extracts an AgentIdentity from a YAML-unmarshaled value.
// The timestampKey parameter specifies which key holds the timestamp (e.g., "created_at" or "claimed_at").
func parseAgentIdentity(raw interface{}, timestampKey string) *port.AgentIdentity {
	m, ok := raw.(map[string]interface{})
	if !ok {
		// Try map[interface{}]interface{} which some YAML parsers produce.
		if mi, ok2 := raw.(map[interface{}]interface{}); ok2 {
			m = make(map[string]interface{}, len(mi))
			for k, v := range mi {
				if ks, ok3 := k.(string); ok3 {
					m[ks] = v
				}
			}
		} else {
			return nil
		}
	}

	identity := &port.AgentIdentity{}
	hasData := false

	if v, ok := m["agent_id"]; ok {
		if s, ok := v.(string); ok && s != "" {
			identity.AgentID = s
			hasData = true
		}
	}
	if v, ok := m["role"]; ok {
		if s, ok := v.(string); ok && s != "" {
			identity.Role = s
			hasData = true
		}
	}
	if v, ok := m["session_id"]; ok {
		if s, ok := v.(string); ok && s != "" {
			identity.SessionID = s
			hasData = true
		}
	}
	if v, ok := m["worktree"]; ok {
		if s, ok := v.(string); ok && s != "" {
			identity.Worktree = s
			hasData = true
		}
	}
	if v, ok := m["branch"]; ok {
		if s, ok := v.(string); ok && s != "" {
			identity.Branch = s
			hasData = true
		}
	}
	if v, ok := m["model"]; ok {
		if s, ok := v.(string); ok && s != "" {
			identity.Model = s
			hasData = true
		}
	}
	if v, ok := m[timestampKey]; ok {
		if s, ok := v.(string); ok && s != "" {
			identity.Timestamp = s
			hasData = true
		}
	}

	if !hasData {
		return nil
	}
	return identity
}

func (r *TrackReaderImpl) RemoveTrack(projectDir, trackID string) error {
	client := kf.NewClientFromProject(projectDir)
	return client.RemoveTrack(trackID)
}

func (r *TrackReaderImpl) IsInitialized(projectDir string) bool {
	client := kf.NewClientFromProject(projectDir)
	return client.IsInitialized()
}

// mapKFStatusToPort maps kf SDK statuses to the port/gen status strings.
// The gen layer uses "complete" while kf uses "completed".
func mapKFStatusToPort(status string) string {
	if status == kf.StatusCompleted {
		return StatusComplete
	}
	return status
}

// renderSpec converts structured Spec to a readable markdown string.
func renderSpec(s kf.Spec) string {
	var b strings.Builder
	if s.Summary != "" {
		b.WriteString("## Summary\n\n")
		b.WriteString(strings.TrimSpace(s.Summary))
		b.WriteString("\n")
	}
	if s.Context != "" {
		b.WriteString("\n## Context\n\n")
		b.WriteString(strings.TrimSpace(s.Context))
		b.WriteString("\n")
	}
	if s.CodebaseAnalysis != "" {
		b.WriteString("\n## Codebase Analysis\n\n")
		b.WriteString(strings.TrimSpace(s.CodebaseAnalysis))
		b.WriteString("\n")
	}
	if len(s.AcceptanceCriteria) > 0 {
		b.WriteString("\n## Acceptance Criteria\n\n")
		for _, c := range s.AcceptanceCriteria {
			b.WriteString("- ")
			b.WriteString(c)
			b.WriteString("\n")
		}
	}
	if s.OutOfScope != "" {
		b.WriteString("\n## Out of Scope\n\n")
		b.WriteString(strings.TrimSpace(s.OutOfScope))
		b.WriteString("\n")
	}
	if s.TechnicalNotes != "" {
		b.WriteString("\n## Technical Notes\n\n")
		b.WriteString(strings.TrimSpace(s.TechnicalNotes))
		b.WriteString("\n")
	}
	return b.String()
}

// renderPlan converts structured Plan phases to a readable markdown string.
func renderPlan(phases []kf.Phase) string {
	var b strings.Builder
	for i, p := range phases {
		if i > 0 {
			b.WriteString("\n")
		}
		fmt.Fprintf(&b, "## Phase %d: %s\n\n", i+1, p.Name)
		for _, t := range p.Tasks {
			if t.Done {
				b.WriteString("- [x] ")
			} else {
				b.WriteString("- [ ] ")
			}
			b.WriteString(t.Text)
			b.WriteString("\n")
		}
	}
	return b.String()
}
