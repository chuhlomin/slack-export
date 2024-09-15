// HTTP server implementation for OAuth callback
package main

import (
	"log"
	"net/http"
	"time"
)

// Server is an HTTP server that listens for OAuth callbacks.
type Server struct {
	svc   *http.Server
	state string
	code  chan string
}

// NewServer creates a new Server.
func NewServer(address, port, state string, code chan string) *Server {
	return &Server{
		svc: &http.Server{
			Addr:              address + ":" + port,
			ReadHeaderTimeout: 10 * time.Second,
		},
		state: state,
		code:  code,
	}
}

// Start starts the server.
func (s *Server) Start() error {
	s.svc.Handler = http.HandlerFunc(s.Handler)
	return s.svc.ListenAndServe()
}

// Stop stops the server.
func (s *Server) Stop() error {
	return s.svc.Close()
}

// Handler handles OAuth callback requests.
func (s *Server) Handler(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "code is required", http.StatusBadRequest)
		return
	}

	state := r.URL.Query().Get("state")
	if state != s.state {
		http.Error(w, "state mismatch", http.StatusBadRequest)
		return
	}

	_, err := w.Write([]byte("Code received. You can close this tab now."))
	if err != nil {
		log.Printf("could not write response: %v", err)
	}

	s.code <- code
}
