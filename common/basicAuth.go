package common

import (
	"encoding/base64"
	"net/http"
	"strings"
)

// ChainHandlers provides a cleaner interface for chaining middleware for single routes.
// Middleware functions are simple HTTP handlers (w http.ResponseWriter, r *http.Request)
//
//  r.HandleFunc("/login", use(loginHandler, rateLimit, csrf))
//
// See https://gist.github.com/elithrar/7600878#comment-955958 for how to extend it to suit simple http.Handler's
func ChainHandlers(h http.HandlerFunc, middleware ...func(http.HandlerFunc) http.HandlerFunc) http.HandlerFunc {
	for _, m := range middleware {
		h = m(h)
	}

	return h
}

// Leverages nemo's answer in http://stackoverflow.com/a/21937924/556573
func BasicAuthHandler(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
		s := strings.SplitN(r.Header.Get("Authorization"), " ", 2)
		if len(s) == 2 {
			b, err := base64.StdEncoding.DecodeString(s[1])
			if err == nil {
				pair := strings.SplitN(string(b), ":", 2)
				if len(pair) == 2 && pair[0] == "archon" && pair[1] == "helloGoogle" {
					h.ServeHTTP(w, r)
					return
				}
			}
		}
		http.Error(w, "Not authorized", http.StatusUnauthorized)
	}
}

