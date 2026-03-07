package badge

import (
	"encoding/xml"
	"strings"
	"testing"
)

func TestRenderBadge_ValidSVG(t *testing.T) {
	t.Parallel()
	svg := RenderBadge("track", "running")
	if err := xml.Unmarshal(svg, new(any)); err != nil {
		t.Fatalf("invalid XML: %v\n%s", err, svg)
	}
}

func TestRenderBadge_StatusColors(t *testing.T) {
	t.Parallel()
	tests := []struct {
		status string
		color  string
	}{
		{"running", "#4c1"},
		{"failed", "#e05d44"},
		{"completed", "#007ec6"},
		{"waiting", "#dfb317"},
		{"halted", "#fe7d37"},
		{"pending", "#9f9f9f"},
		{"unknown-status", "#9f9f9f"},
	}
	for _, tt := range tests {
		svg := string(RenderBadge("test", tt.status))
		if !strings.Contains(svg, tt.color) {
			t.Errorf("status %q: expected color %s in SVG", tt.status, tt.color)
		}
	}
}

func TestRenderBadge_ContainsText(t *testing.T) {
	t.Parallel()
	svg := string(RenderBadge("my-track", "running"))
	if !strings.Contains(svg, "my-track") {
		t.Error("badge does not contain label text")
	}
	if !strings.Contains(svg, "running") {
		t.Error("badge does not contain status text")
	}
}

func TestRenderBadge_EscapesXML(t *testing.T) {
	t.Parallel()
	svg := RenderBadge("a<b", "s&t")
	if err := xml.Unmarshal(svg, new(any)); err != nil {
		t.Fatalf("invalid XML after escaping: %v\n%s", err, svg)
	}
	s := string(svg)
	if strings.Contains(s, "a<b") {
		t.Error("label not properly escaped")
	}
}

func TestRenderDualBadge_ValidSVG(t *testing.T) {
	t.Parallel()
	svg := RenderDualBadge("PR #42", "running", "waiting")
	if err := xml.Unmarshal(svg, new(any)); err != nil {
		t.Fatalf("invalid XML: %v\n%s", err, svg)
	}
}

func TestRenderDualBadge_ContainsBothStatuses(t *testing.T) {
	t.Parallel()
	svg := string(RenderDualBadge("PR #5", "running", "halted"))
	if !strings.Contains(svg, "dev: running") {
		t.Error("missing dev status")
	}
	if !strings.Contains(svg, "rev: halted") {
		t.Error("missing rev status")
	}
}

func TestTrackBadgeMarkdown(t *testing.T) {
	t.Parallel()
	md := TrackBadgeMarkdown("my-track_123", "http://localhost:3001")
	if !strings.Contains(md, "/-/api/badges/track/my-track_123") {
		t.Error("missing badge URL")
	}
	if !strings.Contains(md, "/-/tracks/my-track_123") {
		t.Error("missing link URL")
	}
}

func TestPRBadgeMarkdown(t *testing.T) {
	t.Parallel()
	md := PRBadgeMarkdown("myproject", 42, "http://localhost:3001")
	if !strings.Contains(md, "/-/api/badges/pr/myproject/42") {
		t.Error("missing badge URL")
	}
	if !strings.Contains(md, "/-/pr/myproject/42") {
		t.Error("missing link URL")
	}
}

func TestAgentBadgeMarkdown(t *testing.T) {
	t.Parallel()
	md := AgentBadgeMarkdown("agent-123", "http://localhost:3001", "http://localhost:3001/tracks/t1")
	if !strings.Contains(md, "/-/api/badges/agent/agent-123") {
		t.Error("missing badge URL")
	}
	if !strings.Contains(md, "http://localhost:3001/tracks/t1") {
		t.Error("missing link URL")
	}
}
