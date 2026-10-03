package httpapi

import (
	"net/http"
	"strings"
)

func withCORS(next http.Handler, allowed string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		origin := request.Header.Get("Origin")
		if origin == "" {
			next.ServeHTTP(w, request)
			return
		}

		if origin != allowed {
			if request.Method == http.MethodOptions {
				http.Error(w, "origin is not allowed", http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, request)
			return
		}

		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		appendVary(w.Header(), "Origin")
		if request.Method == http.MethodOptions {
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type")
			w.Header().Set("Access-Control-Max-Age", "600")
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, request)
	})
}

func appendVary(header http.Header, value string) {
	for _, current := range header.Values("Vary") {
		for _, token := range strings.Split(current, ",") {
			if strings.EqualFold(strings.TrimSpace(token), value) {
				return
			}
		}
	}
	header.Add("Vary", value)
}
