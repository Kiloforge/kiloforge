package proxy

import (
	"net/http"
	"net/http/httputil"
	"net/url"
)

// NewGiteaProxy creates a reverse proxy handler that forwards requests to Gitea.
// Mounted at "/gitea/" with path stripping so requests like /gitea/user/login
// are forwarded to Gitea as /user/login.
// If authUser is non-empty, it injects the X-WEBAUTH-USER header on every request
// so Gitea treats the user as authenticated via reverse proxy auth.
func NewGiteaProxy(targetURL, authUser string) http.Handler {
	target, err := url.Parse(targetURL)
	if err != nil {
		panic("invalid gitea proxy target URL: " + err.Error())
	}
	rp := httputil.NewSingleHostReverseProxy(target)
	var handler http.Handler = rp
	if authUser != "" {
		handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.Header.Set("X-WEBAUTH-USER", authUser)
			rp.ServeHTTP(w, r)
		})
	}
	return http.StripPrefix("/gitea", handler)
}
