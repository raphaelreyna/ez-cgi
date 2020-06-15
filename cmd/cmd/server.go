package cmd

import (
	"github.com/raphaelreyna/ez-cgi/pkg/cgi"
	"net/http"
)

type server struct {
	mux        *http.ServeMux
	port       string
	cgiHandler *cgi.Handler
}

func newServer() *server {
	mux := http.NewServeMux()
	return &server{
		mux: mux,
	}
}

func (s *server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.cgiHandler.ServeHTTP(w, r)
}
