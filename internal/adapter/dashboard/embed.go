package dashboard

import (
	"embed"
	"io/fs"
)

//go:embed static/*
var embeddedFiles embed.FS

var staticFS, _ = fs.Sub(embeddedFiles, "static")
