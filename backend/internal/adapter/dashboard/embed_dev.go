// Stub for dev/test builds when frontend assets haven't been compiled.
// The production build uses embed.go with -tags=embed_frontend.

//go:build !embed_frontend

package dashboard

import (
	"io/fs"
	"net/http"
	"time"
)

// placeholderFS serves a single index.html telling the user to build the frontend.
type placeholderFS struct{}

func (placeholderFS) Open(name string) (fs.File, error) {
	if name == "." || name == "index.html" {
		return &placeholderFile{}, nil
	}
	return nil, fs.ErrNotExist
}

type placeholderFile struct{ offset int }

var placeholderHTML = []byte(`<!DOCTYPE html><html><body><p>Frontend not built. Run <code>make build-frontend</code>.</p></body></html>`)

func (f *placeholderFile) Stat() (fs.FileInfo, error) { return placeholderInfo{}, nil }
func (f *placeholderFile) Read(b []byte) (int, error) {
	if f.offset >= len(placeholderHTML) {
		return 0, fs.ErrNotExist
	}
	n := copy(b, placeholderHTML[f.offset:])
	f.offset += n
	return n, nil
}
func (f *placeholderFile) Close() error { return nil }

type placeholderInfo struct{}

func (placeholderInfo) Name() string      { return "index.html" }
func (placeholderInfo) Size() int64       { return int64(len(placeholderHTML)) }
func (placeholderInfo) Mode() fs.FileMode { return 0o444 }
func (placeholderInfo) ModTime() time.Time { return time.Time{} }
func (placeholderInfo) IsDir() bool       { return false }
func (placeholderInfo) Sys() any          { return nil }

var staticFS fs.FS = placeholderFS{}

func spaFileServer(fsys http.FileSystem) http.Handler {
	return http.FileServer(fsys)
}
