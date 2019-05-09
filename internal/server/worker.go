package server

import (
	"context"
	"gitlab.com/pennersr/shove/internal/queue"
	"gitlab.com/pennersr/shove/internal/services"
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

func (w *worker) serve(fc services.FeedbackCollector) {
	w.service.Serve(w.ctx, w.queue, fc)
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
