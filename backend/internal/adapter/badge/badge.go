package badge

import (
	"fmt"
	"strings"
)

// StatusColors maps agent status strings to hex colors.
var StatusColors = map[string]string{
	"running":       "#4c1",
	"waiting":       "#dfb317",
	"completed":     "#007ec6",
	"failed":        "#e05d44",
	"halted":        "#fe7d37",
	"stopped":       "#9f9f9f",
	"suspended":     "#9f9f9f",
	"suspending":    "#dfb317",
	"force-killed":  "#e05d44",
	"resume-failed": "#e05d44",
	"pending":       "#9f9f9f",
	"unknown":       "#9f9f9f",
}

// ColorForStatus returns the hex color for a given status.
func ColorForStatus(status string) string {
	if c, ok := StatusColors[status]; ok {
		return c
	}
	return "#9f9f9f"
}

const charWidth = 7 // approximate px per character for Verdana 11px
const padding = 10  // px padding per side

// RenderBadge generates a shields.io-style flat SVG badge.
func RenderBadge(label, status string) []byte {
	color := ColorForStatus(status)
	labelWidth := len(label)*charWidth + padding*2
	statusWidth := len(status)*charWidth + padding*2
	totalWidth := labelWidth + statusWidth
	labelCenter := labelWidth / 2
	statusCenter := labelWidth + statusWidth/2

	svg := fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="20">
  <rect width="%d" height="20" fill="#555"/>
  <rect x="%d" width="%d" height="20" fill="%s"/>
  <text x="%d" y="14" fill="#fff" font-family="Verdana,Geneva,sans-serif" font-size="11" text-anchor="middle">%s</text>
  <text x="%d" y="14" fill="#fff" font-family="Verdana,Geneva,sans-serif" font-size="11" text-anchor="middle">%s</text>
</svg>`,
		totalWidth,
		labelWidth,
		labelWidth, statusWidth, color,
		labelCenter, escapeXML(label),
		statusCenter, escapeXML(status),
	)
	return []byte(svg)
}

// RenderDualBadge generates a badge showing two statuses (for PR with dev + rev).
func RenderDualBadge(label, status1, status2 string) []byte {
	combined := fmt.Sprintf("dev: %s · rev: %s", status1, status2)
	// Use the "worst" color.
	color := worstColor(status1, status2)
	labelWidth := len(label)*charWidth + padding*2
	statusWidth := len(combined)*charWidth + padding*2
	totalWidth := labelWidth + statusWidth
	labelCenter := labelWidth / 2
	statusCenter := labelWidth + statusWidth/2

	svg := fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="20">
  <rect width="%d" height="20" fill="#555"/>
  <rect x="%d" width="%d" height="20" fill="%s"/>
  <text x="%d" y="14" fill="#fff" font-family="Verdana,Geneva,sans-serif" font-size="11" text-anchor="middle">%s</text>
  <text x="%d" y="14" fill="#fff" font-family="Verdana,Geneva,sans-serif" font-size="11" text-anchor="middle">%s</text>
</svg>`,
		totalWidth,
		labelWidth,
		labelWidth, statusWidth, color,
		labelCenter, escapeXML(label),
		statusCenter, escapeXML(combined),
	)
	return []byte(svg)
}

// statusPriority returns a priority number (higher = worse).
var statusPriority = map[string]int{
	"running":   1,
	"completed": 2,
	"waiting":   3,
	"suspended": 4,
	"halted":    5,
	"stopped":   5,
	"failed":    6,
	"pending":   0,
	"unknown":   0,
}

func worstColor(s1, s2 string) string {
	p1 := statusPriority[s1]
	p2 := statusPriority[s2]
	if p1 >= p2 {
		return ColorForStatus(s1)
	}
	return ColorForStatus(s2)
}

func escapeXML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}
