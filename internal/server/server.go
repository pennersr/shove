package server

import (
	"context"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"gitlab.com/pennersr/shove/internal/queue"
	"gitlab.com/pennersr/shove/internal/services"
	"log"
	"net/http"
	"sync"
)

// Server ...
type Server struct {
	server       *http.Server
	shuttingDown bool
	queueFactory queue.QueueFactory
	workers      map[string]*worker
	feedbackLock sync.Mutex
	feedback     []tokenFeedback
}

// NewServer ...
func NewServer(addr string, qf queue.QueueFactory) (s *Server) {
	mux := http.NewServeMux()

	h := &http.Server{
		Addr:    addr,
		Handler: mux,
	}
	s = &Server{
		server:       h,
		queueFactory: qf,
		workers:      make(map[string]*worker),
		feedback:     make([]tokenFeedback, 0),
	}
	mux.HandleFunc("/api/push/", s.handlePush)
	mux.HandleFunc("/api/feedback", s.handleFeedback)
	mux.Handle("/metrics", promhttp.Handler())
	return s
}

// Serve ...
func (s *Server) Serve() (err error) {
	log.Println("Shove server started")
	err = s.server.ListenAndServe()
	if s.shuttingDown {
		err = nil
	}
	return
}

// Shutdown ...
func (s *Server) Shutdown(ctx context.Context) (err error) {
	s.shuttingDown = true
	s.server.Shutdown(ctx)
	if err = s.server.Shutdown(ctx); err != nil {
		log.Printf("[ERROR] Shutting down Shove server: %v\n", err)
		return
	}
	log.Println("Shove server stopped")
	for _, w := range s.workers {
		err = w.shutdown()
		if err != nil {
			return
		}
	}
	return
}

// AddService ...
func (s *Server) AddService(pp services.PushService, workers int, squash services.SquashConfig) (err error) {
	log.Printf("Initializing %s service", pp)
	q, err := s.queueFactory.NewQueue(pp.ID())
	if err != nil {
		return
	}
	w, err := newWorker(pp, q)
	if err != nil {
		return
	}
	go w.serve(workers, squash, s)
	s.workers[pp.ID()] = w
	return
}
