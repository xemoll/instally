package app

import (
	"os"
	"os/signal"
	"syscall"
)

type ShutdownHandler struct {
	cancel chan struct{}
	done   chan struct{}
}

func NewShutdownHandler() *ShutdownHandler {
	h := &ShutdownHandler{
		cancel: make(chan struct{}),
		done:   make(chan struct{}),
	}
	go h.listen()
	return h
}

func (h *ShutdownHandler) listen() {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	select {
	case <-sig:
		close(h.cancel)
	case <-h.done:
		return
	}
	signal.Stop(sig)
}

func (h *ShutdownHandler) Cancelled() <-chan struct{} {
	return h.cancel
}

func (h *ShutdownHandler) Done() {
	close(h.done)
}
