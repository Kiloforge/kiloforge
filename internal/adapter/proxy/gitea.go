package proxy

import (
	"net/http"
	"net/http/httputil"
	"net/url"
)

// NewGiteaProxy creates a reverse proxy handler that forwards requests to Gitea.
// Use with http.StripPrefix("/gitea", proxy) to strip the path prefix before forwarding.
func NewGiteaProxy(targetURL string) http.Handler {
	target, err := url.Parse(targetURL)
	if err != nil {
		panic("invalid gitea proxy target URL: " + err.Error())
	}
	return httputil.NewSingleHostReverseProxy(target)
}
