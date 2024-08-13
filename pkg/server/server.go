// HTTP server implementation for OAuth callback
package server

import "net/http"

type Server struct {
	svc   *http.Server
	state string
	code  chan string
}

func NewServer(address, port, state string, code chan string) *Server {
	return &Server{
		svc: &http.Server{
			Addr: address + ":" + port,
		},
		state: state,
		code:  code,
	}
}

func (s *Server) Start() error {
	s.svc.Handler = http.HandlerFunc(s.Handler)
	return s.svc.ListenAndServe()
}

func (s *Server) Stop() error {
	return s.svc.Close()
}

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

	w.Write([]byte("Code received. You can close this tab now."))

	s.code <- code
}
