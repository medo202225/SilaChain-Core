package rpc

import (
	"net/http"
	"time"
)

type Server struct {
	addr   string
	router *http.ServeMux
}

func NewServer(addr string) *Server {
	return &Server{
		addr:   addr,
		router: http.NewServeMux(),
	}
}

func (s *Server) Router() *http.ServeMux {
	return s.router
}

func (s *Server) Start() error {
	handler := applyGlobalMiddleware(s.router)

	srv := &http.Server{
		Addr:              s.addr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}

	return srv.ListenAndServe()
}
