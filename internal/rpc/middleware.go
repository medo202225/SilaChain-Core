package rpc

import "net/http"

func applyGlobalMiddleware(h http.Handler) http.Handler {
	if h == nil {
		return http.NewServeMux()
	}
	return h
}
