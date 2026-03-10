package analytics

// DefaultPostHogAPIKey is the public write-only PostHog project key.
// This key can only ingest events — it cannot read data or modify project settings.
// Injected at build time via ldflags; falls back to placeholder (analytics disabled).
var DefaultPostHogAPIKey = "phc_kiloforge_placeholder"
