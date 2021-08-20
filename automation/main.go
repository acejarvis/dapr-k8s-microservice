package main

import (
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/mux"
	"github.com/rs/cors"
)

func main() {
	var wg sync.WaitGroup

	// init log, currently, hard code log level to info, and output format to console
	log.Println("info", "console")

	s, err := NewServer(&wg)

	if err != nil {
		log.Fatal(err)
	}

	if s != nil {
		go s.WithMuxer().Start()
	}

	wg.Wait()

}

type Server struct {
	wg    *sync.WaitGroup
	muxer *mux.Router
}

// create a server struct, input is wait group
func NewServer(wg *sync.WaitGroup) (*Server, error) {
	s := &Server{
		wg: wg,
	}

	// add one job to wait group
	s.wg.Add(1)
	return s, nil
}

// set up endpoint with muxer
func (s *Server) WithMuxer() *Server {
	s.muxer = mux.NewRouter()
	// health check
	s.muxer.HandleFunc("/health", s.HandleHeathCheck).Methods("GET")

	// create a muxer, all other rest api are under this muxer
	subRouter := s.muxer.PathPrefix("/api").Subrouter()
	subRouter.HandleFunc("/", s.HandleHelloWorld).Methods("GET")
	subRouter.Use(s.Auth)

	return s
}

func (s *Server) Start() {
	log.Println("Dapr Automation Server start...", nil)

	c := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
	})

	handler := c.Handler(s.muxer)

	err := http.ListenAndServe(fmt.Sprintf("%s:%d", "0.0.0.0", 3000), handler)
	if err != nil {
		log.Fatal(err)
	}
	s.wg.Done()
}

func (s *Server) Auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r)
	})
}

func (s *Server) HandleHeathCheck(w http.ResponseWriter, r *http.Request) {
	log.Println("HandleHeathCheck")
}

func (s *Server) HandleHelloWorld(w http.ResponseWriter, r *http.Request) {
	log.Println("HandleHelloWorld")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Hello Dapr K8s!"))
}