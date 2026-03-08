package main

// Injected at build time via -ldflags -X.
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)
