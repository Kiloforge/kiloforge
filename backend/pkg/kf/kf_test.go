package kf_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"kiloforge/pkg/kf"
)

// --- Registry tests ---

func TestReadRegistry(t *testing.T) {
	input := `# comment
track-a_20260101Z: {"title":"Track A","status":"pending","type":"feature","created":"2026-01-01","updated":"2026-01-01"}
track-b_20260102Z: {"title":"Track B","status":"completed","type":"bug","created":"2026-01-02","updated":"2026-01-02"}
`
	entries, err := kf.ReadRegistry(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if entries[0].ID != "track-a_20260101Z" {
		t.Errorf("expected ID track-a_20260101Z, got %s", entries[0].ID)
	}
	if entries[0].Status != "pending" {
		t.Errorf("expected status pending, got %s", entries[0].Status)
	}
	if entries[1].Status != "completed" {
		t.Errorf("expected status completed, got %s", entries[1].Status)
	}
}

func TestWriteRegistryAlphabeticalOrder(t *testing.T) {
	entries := []kf.TrackEntry{
		{ID: "zebra_20260101Z", Title: "Zebra", Status: "pending", Type: "feature", Created: "2026-01-01", Updated: "2026-01-01"},
		{ID: "alpha_20260102Z", Title: "Alpha", Status: "pending", Type: "feature", Created: "2026-01-02", Updated: "2026-01-02"},
	}
	var buf strings.Builder
	if err := kf.WriteRegistry(&buf, entries); err != nil {
		t.Fatal(err)
	}
	output := buf.String()
	alphaIdx := strings.Index(output, "alpha_20260102Z")
	zebraIdx := strings.Index(output, "zebra_20260101Z")
	if alphaIdx >= zebraIdx {
		t.Error("alpha should come before zebra in sorted output")
	}
}

func TestWriteRegistryCanonicalJSON(t *testing.T) {
	entries := []kf.TrackEntry{
		{ID: "test_20260101Z", Title: "Test", Status: "pending", Type: "feature", Created: "2026-01-01", Updated: "2026-01-01"},
	}
	var buf strings.Builder
	if err := kf.WriteRegistry(&buf, entries); err != nil {
		t.Fatal(err)
	}
	line := ""
	for _, l := range strings.Split(buf.String(), "\n") {
		if strings.HasPrefix(l, "test_") {
			line = l
			break
		}
	}
	// Verify field order: title before status before type
	titleIdx := strings.Index(line, `"title"`)
	statusIdx := strings.Index(line, `"status"`)
	typeIdx := strings.Index(line, `"type"`)
	if titleIdx > statusIdx || statusIdx > typeIdx {
		t.Errorf("canonical order violated: title=%d status=%d type=%d", titleIdx, statusIdx, typeIdx)
	}
}

func TestRegistryRoundtrip(t *testing.T) {
	entries := []kf.TrackEntry{
		{ID: "a_20260101Z", Title: "A", Status: "pending", Type: "feature", Created: "2026-01-01", Updated: "2026-01-01"},
		{ID: "b_20260102Z", Title: "B", Status: "archived", Type: "bug", Created: "2026-01-02", Updated: "2026-01-02", ArchivedAt: "2026-01-03", ArchiveReason: "done"},
	}
	var buf strings.Builder
	if err := kf.WriteRegistry(&buf, entries); err != nil {
		t.Fatal(err)
	}
	parsed, err := kf.ReadRegistry(strings.NewReader(buf.String()))
	if err != nil {
		t.Fatal(err)
	}
	if len(parsed) != 2 {
		t.Fatalf("expected 2, got %d", len(parsed))
	}
	if parsed[1].ArchiveReason != "done" {
		t.Errorf("expected archive_reason done, got %q", parsed[1].ArchiveReason)
	}
}

func TestFindEntry(t *testing.T) {
	entries := []kf.TrackEntry{
		{ID: "a", Title: "A"},
		{ID: "b", Title: "B"},
	}
	found := kf.FindEntry(entries, "b")
	if found == nil || found.Title != "B" {
		t.Error("expected to find entry B")
	}
	if kf.FindEntry(entries, "c") != nil {
		t.Error("expected nil for missing entry")
	}
}

func TestActiveEntries(t *testing.T) {
	entries := []kf.TrackEntry{
		{ID: "a", Status: kf.StatusPending},
		{ID: "b", Status: kf.StatusCompleted},
		{ID: "c", Status: kf.StatusInProgress},
		{ID: "d", Status: kf.StatusArchived},
	}
	active := kf.ActiveEntries(entries)
	if len(active) != 2 {
		t.Fatalf("expected 2 active, got %d", len(active))
	}
}

// --- Deps tests ---

func TestReadDeps(t *testing.T) {
	input := `# comment

a_20260101Z: []

b_20260102Z:
  - a_20260101Z
  - c_20260103Z
`
	graph, err := kf.ReadDeps(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	if deps := graph.GetDeps("a_20260101Z"); len(deps) != 0 {
		t.Errorf("expected 0 deps for a, got %d", len(deps))
	}
	if deps := graph.GetDeps("b_20260102Z"); len(deps) != 2 {
		t.Errorf("expected 2 deps for b, got %d", len(deps))
	}
}

func TestDepsAddRemove(t *testing.T) {
	graph := make(kf.DepsGraph)
	graph.AddDep("b", "a")
	graph.AddDep("b", "c")
	graph.AddDep("b", "a") // duplicate
	if len(graph.GetDeps("b")) != 2 {
		t.Errorf("expected 2 deps, got %d", len(graph.GetDeps("b")))
	}
	graph.RemoveDep("b", "a")
	if len(graph.GetDeps("b")) != 1 {
		t.Errorf("expected 1 dep, got %d", len(graph.GetDeps("b")))
	}
}

func TestDepsAllSatisfied(t *testing.T) {
	graph := make(kf.DepsGraph)
	graph.AddDep("feature", "infra")
	graph.AddDep("feature", "schema")

	completed := map[string]bool{"infra": true}
	if graph.AllDepsSatisfied("feature", completed) {
		t.Error("should not be satisfied (schema not complete)")
	}

	completed["schema"] = true
	if !graph.AllDepsSatisfied("feature", completed) {
		t.Error("should be satisfied")
	}
}

func TestDepsRemoveTrack(t *testing.T) {
	graph := make(kf.DepsGraph)
	graph["a"] = nil
	graph["b"] = []string{"a"}
	graph["c"] = []string{"a", "b"}

	graph.RemoveTrack("a")
	if _, ok := graph["a"]; ok {
		t.Error("a should be removed as key")
	}
	if deps := graph["b"]; len(deps) != 0 {
		t.Error("a should be removed from b's deps")
	}
	if deps := graph["c"]; len(deps) != 1 || deps[0] != "b" {
		t.Errorf("c should have only b, got %v", deps)
	}
}

func TestDepsRoundtrip(t *testing.T) {
	graph := kf.DepsGraph{
		"z_track": []string{"a_track", "b_track"},
		"a_track": nil,
	}
	var buf strings.Builder
	if err := kf.WriteDeps(&buf, graph); err != nil {
		t.Fatal(err)
	}
	parsed, err := kf.ReadDeps(strings.NewReader(buf.String()))
	if err != nil {
		t.Fatal(err)
	}
	if deps := parsed.GetDeps("z_track"); len(deps) != 2 {
		t.Errorf("expected 2 deps, got %d", len(deps))
	}
}

// --- Conflicts tests ---

func TestConflictPairOrdering(t *testing.T) {
	pair := kf.NewConflictPair("zebra", "alpha", "high", "test")
	if pair.TrackA != "alpha" || pair.TrackB != "zebra" {
		t.Errorf("pair should be ordered: got %s/%s", pair.TrackA, pair.TrackB)
	}
	if pair.PairKey() != "alpha/zebra" {
		t.Errorf("unexpected pair key: %s", pair.PairKey())
	}
}

func TestConflictInvolves(t *testing.T) {
	pair := kf.NewConflictPair("a", "b", "low", "")
	if !pair.Involves("a") || !pair.Involves("b") {
		t.Error("should involve both a and b")
	}
	if pair.Involves("c") {
		t.Error("should not involve c")
	}
}

func TestConflictsRoundtrip(t *testing.T) {
	pairs := []kf.ConflictPair{
		kf.NewConflictPair("z_track", "a_track", "high", "overlap"),
		kf.NewConflictPair("b_track", "c_track", "low", "minor"),
	}
	var buf strings.Builder
	if err := kf.WriteConflicts(&buf, pairs); err != nil {
		t.Fatal(err)
	}
	parsed, err := kf.ReadConflicts(strings.NewReader(buf.String()))
	if err != nil {
		t.Fatal(err)
	}
	if len(parsed) != 2 {
		t.Fatalf("expected 2 pairs, got %d", len(parsed))
	}
	// Should be sorted: a_track/z_track before b_track/c_track
	if parsed[0].PairKey() != "a_track/z_track" {
		t.Errorf("expected a_track/z_track first, got %s", parsed[0].PairKey())
	}
}

func TestAddOrUpdateConflict(t *testing.T) {
	pairs := []kf.ConflictPair{
		kf.NewConflictPair("a", "b", "high", "first"),
	}
	updated := kf.AddOrUpdateConflict(pairs, kf.NewConflictPair("a", "b", "low", "updated"))
	if len(updated) != 1 {
		t.Fatalf("expected 1 pair, got %d", len(updated))
	}
	if updated[0].Risk != "low" {
		t.Errorf("expected risk low, got %s", updated[0].Risk)
	}
}

func TestRemoveConflictsForTrack(t *testing.T) {
	pairs := []kf.ConflictPair{
		kf.NewConflictPair("a", "b", "high", ""),
		kf.NewConflictPair("a", "c", "low", ""),
		kf.NewConflictPair("d", "e", "medium", ""),
	}
	filtered := kf.RemoveConflictsForTrack(pairs, "a")
	if len(filtered) != 1 {
		t.Fatalf("expected 1 pair remaining, got %d", len(filtered))
	}
	if filtered[0].PairKey() != "d/e" {
		t.Errorf("expected d/e, got %s", filtered[0].PairKey())
	}
}

// --- Track content tests ---

func TestTrackProgress(t *testing.T) {
	track := &kf.Track{
		Plan: []kf.Phase{
			{Name: "Setup", Tasks: []kf.Task{{Text: "T1", Done: true}, {Text: "T2", Done: true}}},
			{Name: "Impl", Tasks: []kf.Task{{Text: "T3", Done: false}}},
		},
	}
	stats := track.Progress()
	if stats.TotalTasks != 3 {
		t.Errorf("expected 3 total, got %d", stats.TotalTasks)
	}
	if stats.CompletedTasks != 2 {
		t.Errorf("expected 2 completed, got %d", stats.CompletedTasks)
	}
	if stats.CompletedPhases != 1 {
		t.Errorf("expected 1 completed phase, got %d", stats.CompletedPhases)
	}
	if stats.Percent != 66 {
		t.Errorf("expected 66%%, got %d%%", stats.Percent)
	}
}

func TestSetTaskDone(t *testing.T) {
	track := &kf.Track{
		Plan: []kf.Phase{
			{Name: "P1", Tasks: []kf.Task{{Text: "T1", Done: false}}},
		},
	}
	if err := track.SetTaskDone(1, 1, true); err != nil {
		t.Fatal(err)
	}
	if !track.Plan[0].Tasks[0].Done {
		t.Error("task should be done")
	}
	if err := track.SetTaskDone(2, 1, true); err == nil {
		t.Error("expected error for out-of-range phase")
	}
}

func TestNewTrack(t *testing.T) {
	track := kf.NewTrack("test_20260101Z", "Test", "feature", "A summary")
	if track.ID != "test_20260101Z" {
		t.Errorf("unexpected ID: %s", track.ID)
	}
	if track.Status != kf.StatusPending {
		t.Errorf("expected pending, got %s", track.Status)
	}
	if track.Spec.Summary != "A summary" {
		t.Errorf("unexpected summary: %s", track.Spec.Summary)
	}
}

// --- Track YAML roundtrip (requires yaml.v3) ---

func TestTrackYAMLRoundtrip(t *testing.T) {
	dir := t.TempDir()
	trackDir := filepath.Join(dir, "test_20260101Z")
	if err := os.MkdirAll(trackDir, 0o755); err != nil {
		t.Fatal(err)
	}

	original := &kf.Track{
		ID:      "test_20260101Z",
		Title:   "Test Track",
		Type:    "feature",
		Status:  "pending",
		Created: "2026-01-01",
		Updated: "2026-01-01",
		Spec: kf.Spec{
			Summary:            "A test track",
			Context:            "Some context",
			AcceptanceCriteria: []string{"Criterion 1", "Criterion 2"},
		},
		Plan: []kf.Phase{
			{Name: "Setup", Tasks: []kf.Task{
				{Text: "Do thing 1", Done: false},
				{Text: "Do thing 2", Done: true},
			}},
		},
		Extra: map[string]interface{}{"key1": "val1"},
	}

	path := filepath.Join(trackDir, "track.yaml")
	if err := kf.WriteTrack(path, original); err != nil {
		t.Fatal(err)
	}

	loaded, err := kf.ReadTrack(path)
	if err != nil {
		t.Fatal(err)
	}

	if loaded.ID != original.ID {
		t.Errorf("ID mismatch: %s != %s", loaded.ID, original.ID)
	}
	if loaded.Spec.Summary != original.Spec.Summary {
		t.Errorf("summary mismatch")
	}
	if len(loaded.Spec.AcceptanceCriteria) != 2 {
		t.Errorf("expected 2 criteria, got %d", len(loaded.Spec.AcceptanceCriteria))
	}
	if len(loaded.Plan) != 1 || len(loaded.Plan[0].Tasks) != 2 {
		t.Error("plan structure mismatch")
	}
	if !loaded.Plan[0].Tasks[1].Done {
		t.Error("task 2 should be done")
	}
	if loaded.Extra["key1"] != "val1" {
		t.Errorf("extra key1 mismatch: %q", loaded.Extra["key1"])
	}
}

// --- Client integration test ---

func TestClientFullLifecycle(t *testing.T) {
	dir := t.TempDir()
	kfDir := filepath.Join(dir, ".agent", "kf")
	os.MkdirAll(filepath.Join(kfDir, "tracks"), 0o755)

	client := kf.NewClient(kfDir)

	// Add tracks
	if err := client.AddTrack(kf.TrackEntry{
		ID: "infra_20260101Z", Title: "Infra", Status: kf.StatusPending,
		Type: "chore", Created: "2026-01-01", Updated: "2026-01-01",
	}, nil); err != nil {
		t.Fatal(err)
	}

	if err := client.AddTrack(kf.TrackEntry{
		ID: "feature_20260102Z", Title: "Feature", Status: kf.StatusPending,
		Type: "feature", Created: "2026-01-02", Updated: "2026-01-02",
	}, []string{"infra_20260101Z"}); err != nil {
		t.Fatal(err)
	}

	// Add conflict
	if err := client.AddConflict("infra_20260101Z", "feature_20260102Z", "medium", "shared config"); err != nil {
		t.Fatal(err)
	}

	// Feature should be blocked
	satisfied, unmet, err := client.CheckDeps("feature_20260102Z")
	if err != nil {
		t.Fatal(err)
	}
	if satisfied {
		t.Error("feature should be blocked")
	}
	if len(unmet) != 1 || unmet[0] != "infra_20260101Z" {
		t.Errorf("unexpected unmet: %v", unmet)
	}

	// Complete infra
	if err := client.UpdateStatus("infra_20260101Z", kf.StatusCompleted); err != nil {
		t.Fatal(err)
	}

	// Feature should now be unblocked
	satisfied, _, err = client.CheckDeps("feature_20260102Z")
	if err != nil {
		t.Fatal(err)
	}
	if !satisfied {
		t.Error("feature should be unblocked")
	}

	// Conflicts should be cleaned
	conflicts, err := client.GetConflictsForTrack("feature_20260102Z")
	if err != nil {
		t.Fatal(err)
	}
	if len(conflicts) != 0 {
		t.Errorf("expected 0 conflicts, got %d", len(conflicts))
	}

	// Ready tracks should include feature
	ready, err := client.ListReadyTracks()
	if err != nil {
		t.Fatal(err)
	}
	if len(ready) != 1 || ready[0].ID != "feature_20260102Z" {
		t.Errorf("expected feature in ready, got %v", ready)
	}

	// Save and read track content
	track := kf.NewTrack("feature_20260102Z", "Feature", "feature", "Build feature")
	track.Plan = []kf.Phase{{Name: "Impl", Tasks: []kf.Task{{Text: "Do it", Done: false}}}}
	if err := client.SaveTrack(track); err != nil {
		t.Fatal(err)
	}
	loaded, err := client.GetTrack("feature_20260102Z")
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Spec.Summary != "Build feature" {
		t.Errorf("unexpected summary: %s", loaded.Spec.Summary)
	}

	// Progress
	stats, err := client.GetTrackProgress("feature_20260102Z")
	if err != nil {
		t.Fatal(err)
	}
	if stats.TotalTasks != 1 || stats.CompletedTasks != 0 {
		t.Errorf("unexpected progress: %+v", stats)
	}
}

// --- Project metadata tests ---

func TestParseQuickLinks(t *testing.T) {
	content := `# Quick Links
#
# Comments
- [Product Definition](./product.md)
- [Tech Stack](./tech-stack.md)
- [Go Style Guide](./code_styleguides/go.md)
`
	links := kf.ParseQuickLinks(content)
	if len(links) != 3 {
		t.Fatalf("expected 3 links, got %d", len(links))
	}
	if links[0].Label != "Product Definition" || links[0].Path != "./product.md" {
		t.Errorf("link 0: %+v", links[0])
	}
	if links[2].Label != "Go Style Guide" || links[2].Path != "./code_styleguides/go.md" {
		t.Errorf("link 2: %+v", links[2])
	}
}

func TestReadStyleGuides(t *testing.T) {
	dir := t.TempDir()
	sgDir := filepath.Join(dir, "code_styleguides")
	os.MkdirAll(sgDir, 0o755)
	os.WriteFile(filepath.Join(sgDir, "go.md"), []byte("# Go Style\nUse gofmt."), 0o644)
	os.WriteFile(filepath.Join(sgDir, "python.md"), []byte("# Python Style\nUse black."), 0o644)
	os.WriteFile(filepath.Join(sgDir, "not-md.txt"), []byte("skip me"), 0o644)

	guides, err := kf.ReadStyleGuides(sgDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(guides) != 2 {
		t.Fatalf("expected 2 guides, got %d", len(guides))
	}
	// Sorted by ReadDir (alphabetical)
	if guides[0].Name != "go" || !strings.Contains(guides[0].Content, "gofmt") {
		t.Errorf("guide 0: %+v", guides[0])
	}
	if guides[1].Name != "python" || !strings.Contains(guides[1].Content, "black") {
		t.Errorf("guide 1: %+v", guides[1])
	}
}

func TestReadStyleGuidesNotFound(t *testing.T) {
	guides, err := kf.ReadStyleGuides("/nonexistent/dir")
	if err != nil {
		t.Fatal(err)
	}
	if guides != nil {
		t.Errorf("expected nil, got %v", guides)
	}
}

func TestGetProjectInfo(t *testing.T) {
	dir := t.TempDir()
	kfDir := filepath.Join(dir, ".agent", "kf")
	os.MkdirAll(filepath.Join(kfDir, "code_styleguides"), 0o755)
	os.MkdirAll(filepath.Join(kfDir, "tracks"), 0o755)

	os.WriteFile(filepath.Join(kfDir, "product.md"), []byte("# Product\nMy app"), 0o644)
	os.WriteFile(filepath.Join(kfDir, "product-guidelines.md"), []byte("# Guidelines\nBe good"), 0o644)
	os.WriteFile(filepath.Join(kfDir, "tech-stack.md"), []byte("# Tech\nGo + React"), 0o644)
	os.WriteFile(filepath.Join(kfDir, "workflow.md"), []byte("# Workflow\nTrunk-based"), 0o644)
	os.WriteFile(filepath.Join(kfDir, "quick-links.md"), []byte("- [Product](./product.md)\n- [Tech](./tech-stack.md)\n"), 0o644)
	os.WriteFile(filepath.Join(kfDir, "code_styleguides", "go.md"), []byte("# Go\nUse gofmt"), 0o644)

	client := kf.NewClient(kfDir)
	info, err := client.GetProjectInfo()
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(info.Product, "My app") {
		t.Errorf("product: %q", info.Product)
	}
	if !strings.Contains(info.ProductGuidelines, "Be good") {
		t.Errorf("guidelines: %q", info.ProductGuidelines)
	}
	if !strings.Contains(info.TechStack, "Go + React") {
		t.Errorf("tech stack: %q", info.TechStack)
	}
	if !strings.Contains(info.Workflow, "Trunk-based") {
		t.Errorf("workflow: %q", info.Workflow)
	}
	if len(info.QuickLinks) != 2 {
		t.Errorf("expected 2 quick links, got %d", len(info.QuickLinks))
	}
	if len(info.StyleGuides) != 1 || info.StyleGuides[0].Name != "go" {
		t.Errorf("style guides: %+v", info.StyleGuides)
	}
}

func TestGetProjectInfoMissingProduct(t *testing.T) {
	dir := t.TempDir()
	client := kf.NewClient(dir)
	_, err := client.GetProjectInfo()
	if err == nil {
		t.Fatal("expected error for missing product.md")
	}
}

func TestRemoveTrack(t *testing.T) {
	dir := t.TempDir()
	kfDir := filepath.Join(dir, ".agent", "kf")
	os.MkdirAll(filepath.Join(kfDir, "tracks"), 0o755)

	client := kf.NewClient(kfDir)

	// Add a track with content
	entry := kf.TrackEntry{
		ID: "rm_20260101Z", Title: "Remove Me", Status: kf.StatusPending,
		Type: "feature", Created: "2026-01-01", Updated: "2026-01-01",
	}
	if err := client.AddTrack(entry, nil); err != nil {
		t.Fatal(err)
	}
	track := kf.NewTrack("rm_20260101Z", "Remove Me", "feature", "summary")
	if err := client.SaveTrack(track); err != nil {
		t.Fatal(err)
	}

	// Verify track exists
	if _, err := client.GetTrackEntry("rm_20260101Z"); err != nil {
		t.Fatalf("track should exist: %v", err)
	}
	if _, err := client.GetTrack("rm_20260101Z"); err != nil {
		t.Fatalf("track content should exist: %v", err)
	}

	// Remove
	if err := client.RemoveTrack("rm_20260101Z"); err != nil {
		t.Fatal(err)
	}

	// Should be gone from registry
	if _, err := client.GetTrackEntry("rm_20260101Z"); err == nil {
		t.Error("track should be removed from registry")
	}

	// Track dir should be gone
	if _, err := client.GetTrack("rm_20260101Z"); err == nil {
		t.Error("track dir should be removed")
	}
}

func TestRemoveTrackNotFound(t *testing.T) {
	dir := t.TempDir()
	kfDir := filepath.Join(dir, ".agent", "kf")
	os.MkdirAll(filepath.Join(kfDir, "tracks"), 0o755)
	// Write empty registry
	os.WriteFile(filepath.Join(kfDir, "tracks.yaml"), []byte(""), 0o644)

	client := kf.NewClient(kfDir)
	err := client.RemoveTrack("nonexistent_20260101Z")
	if err == nil {
		t.Error("expected error for nonexistent track")
	}
}

func TestIsInitialized(t *testing.T) {
	dir := t.TempDir()
	kfDir := filepath.Join(dir, ".agent", "kf")

	client := kf.NewClient(kfDir)

	// Not initialized — dir doesn't exist
	if client.IsInitialized() {
		t.Error("should not be initialized before kf dir exists")
	}

	// Create dir but no required files
	os.MkdirAll(kfDir, 0o755)
	if client.IsInitialized() {
		t.Error("should not be initialized without required files")
	}

	// Add product.md only
	os.WriteFile(filepath.Join(kfDir, "product.md"), []byte("# Product"), 0o644)
	if client.IsInitialized() {
		t.Error("should not be initialized without tracks.yaml")
	}

	// Add tracks.yaml
	os.WriteFile(filepath.Join(kfDir, "tracks.yaml"), []byte(""), 0o644)
	if !client.IsInitialized() {
		t.Error("should be initialized with product.md and tracks.yaml")
	}
}

func TestGetTrackSummary(t *testing.T) {
	dir := t.TempDir()
	kfDir := filepath.Join(dir, ".agent", "kf")
	os.MkdirAll(filepath.Join(kfDir, "tracks"), 0o755)

	registry := `track-a_20260101Z: {"title":"A","status":"pending","type":"feature","created":"2026-01-01","updated":"2026-01-01"}
track-b_20260102Z: {"title":"B","status":"completed","type":"bug","created":"2026-01-02","updated":"2026-01-02"}
track-c_20260103Z: {"title":"C","status":"in-progress","type":"feature","created":"2026-01-03","updated":"2026-01-03"}
track-d_20260104Z: {"title":"D","status":"archived","type":"chore","created":"2026-01-04","updated":"2026-01-04"}
track-e_20260105Z: {"title":"E","status":"pending","type":"feature","created":"2026-01-05","updated":"2026-01-05"}
`
	os.WriteFile(filepath.Join(kfDir, "tracks.yaml"), []byte(registry), 0o644)

	client := kf.NewClient(kfDir)
	summary, err := client.GetTrackSummary()
	if err != nil {
		t.Fatal(err)
	}
	if summary.Total != 5 {
		t.Errorf("total: %d", summary.Total)
	}
	if summary.Pending != 2 {
		t.Errorf("pending: %d", summary.Pending)
	}
	if summary.InProgress != 1 {
		t.Errorf("in-progress: %d", summary.InProgress)
	}
	if summary.Completed != 1 {
		t.Errorf("completed: %d", summary.Completed)
	}
	if summary.Archived != 1 {
		t.Errorf("archived: %d", summary.Archived)
	}
}
