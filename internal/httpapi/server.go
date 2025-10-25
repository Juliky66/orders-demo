package httpapi

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"order/internal/cache"
)

type Server struct {
	cache cache.Cache
	mux   *http.ServeMux
}

func New(cache cache.Cache) *Server {
	s := &Server{cache: cache, mux: http.NewServeMux()}
	s.routes()
	return s
}

func (s *Server) routes() {
	// static
	s.mux.Handle("/", http.FileServer(http.Dir("./static")))

	// GET /orders/{id}
	s.mux.HandleFunc("/orders/", func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimPrefix(r.URL.Path, "/orders/")
		if id == "" {
			http.NotFound(w, r)
			return
		}
		if o, ok := s.cache.Get(id); ok {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(o)
			return
		}
		http.NotFound(w, r)
	})
}

func (s *Server) Listen(addr string) error {
	log.Println("http listen", addr)
	return http.ListenAndServe(addr, s.mux)
}
