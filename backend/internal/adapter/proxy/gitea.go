package proxy

import (
	"net/http"
	"net/http/httputil"
	"net/url"
)

// NewGiteaProxy creates a reverse proxy handler that forwards requests to Gitea.
// Mounted as catch-all at "/" so Gitea's UI and assets load without path rewriting.
func NewGiteaProxy(targetURL string) http.Handler {
	target, err := url.Parse(targetURL)
	if err != nil {
		panic("invalid gitea proxy target URL: " + err.Error())
	}
	return httputil.NewSingleHostReverseProxy(target)
}
