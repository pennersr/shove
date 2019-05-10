package server

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"time"
)

var (
	pushSuccessCounter = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "shove_push_success_total",
		Help: "The total number of successful push notifications sent",
	}, []string{
		"service",
	})

	pushErrorCounter = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "shove_push_error_total",
		Help: "The total number of push notifications errored",
	}, []string{
		"service",
	})
)

// CountPush ...
func (s *Server) CountPush(serviceID string, success bool, duration time.Duration) {
	if success {
		pushSuccessCounter.WithLabelValues(serviceID).Inc()
	} else {
		pushErrorCounter.WithLabelValues(serviceID).Inc()
	}

}
