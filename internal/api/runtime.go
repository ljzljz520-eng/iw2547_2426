package api

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"
)

type Runtime struct {
	server   *http.Server
	listener net.Listener
}

func Listen(address string, handler http.Handler) (*Runtime, error) {
	if address == "" {
		address = "127.0.0.1:8080"
	}
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return nil, fmt.Errorf("listen: %w", err)
	}
	server := &http.Server{Handler: handler, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 30 * time.Second, IdleTimeout: 60 * time.Second}
	return &Runtime{server: server, listener: listener}, nil
}
func (runtime *Runtime) Address() string { return runtime.listener.Addr().String() }
func (runtime *Runtime) Serve() error {
	err := runtime.server.Serve(runtime.listener)
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}
func (runtime *Runtime) Shutdown(ctx context.Context) error { return runtime.server.Shutdown(ctx) }
