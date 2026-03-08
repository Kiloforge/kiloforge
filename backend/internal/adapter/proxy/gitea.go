package proxy

import (
	"net/http"
	"net/http/httputil"
	"net/url"
)

// NewGiteaProxy creates a reverse proxy handler that forwards requests to Gitea.
// Mounted as catch-all at "/" so Gitea's UI and assets load without path rewriting.
// If authUser is non-empty, it injects the X-WEBAUTH-USER header on every request
// so Gitea treats the user as authenticated via reverse proxy auth.
func NewGiteaProxy(targetURL, authUser string) http.Handler {
	target, err := url.Parse(targetURL)
	if err != nil {
		panic("invalid gitea proxy target URL: " + err.Error())
	}
	rp := httputil.NewSingleHostReverseProxy(target)
	if authUser == "" {
		return rp
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Header.Set("X-WEBAUTH-USER", authUser)
		rp.ServeHTTP(w, r)
	})
}
