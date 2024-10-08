package server

import (
	"codeberg.org/pennersr/shove/internal/queue"
	"codeberg.org/pennersr/shove/internal/services"
	"context"
	"golang.org/x/exp/slog"
)

type worker struct {
	queue    queue.Queue
	service  services.PushService
	ctx      context.Context
	cancel   context.CancelFunc
	finished chan (bool)
}

func newWorker(pp services.PushService, queue queue.Queue) (w *worker, err error) {
	w = &worker{
		queue:    queue,
		service:  pp,
		finished: make(chan bool),
	}
	w.ctx, w.cancel = context.WithCancel(context.Background())
	return
}

func (w *worker) push(msg []byte) (err error) {
	if err = w.service.Validate(msg); err != nil {
		return
	}
	err = w.queue.Queue(msg)
	return
}

func (w *worker) serve(workers int, squash services.SquashConfig, fc services.FeedbackCollector) {
	pump := services.NewPump(workers, squash, w.service)
	err := pump.Serve(w.ctx, w.queue, fc)
	if err != nil {
		slog.Error("Serve failed", "error", err)
	}
	w.finished <- true
}

func (w *worker) shutdown() (err error) {
	if err = w.queue.Shutdown(); err != nil {
		return
	}
	w.cancel()
	<-w.finished
	return
}
