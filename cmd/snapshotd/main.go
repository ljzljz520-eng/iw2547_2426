package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"example.com/roomsnapshot/internal/api"
	"example.com/roomsnapshot/internal/batch"
	"example.com/roomsnapshot/internal/cryptoiface"
	"example.com/roomsnapshot/internal/report"
	"example.com/roomsnapshot/internal/service"
	"example.com/roomsnapshot/internal/store"
	"example.com/roomsnapshot/internal/verify"
)

func main() {
	address := flag.String("listen", "127.0.0.1:8080", "HTTP listen address")
	data := flag.String("data", "room-snapshots.db", "bbolt database path")
	web := flag.String("web", "web", "static console directory")
	flag.Parse()
	if err := run(*address, *data, *web); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func run(address, dataPath, webPath string) error {
	repository, err := store.Open(filepath.Clean(dataPath))
	if err != nil {
		return err
	}
	defer repository.Close()
	codec, err := cryptoiface.NewCodec([]byte("room-snapshot-fixed-key-v1"))
	if err != nil {
		return err
	}
	sealer := batch.NewSealer(codec)
	epoch := time.Date(2026, 8, 26, 0, 0, 0, 0, time.UTC)
	verifier := verify.New(sealer, func() time.Time { return epoch })
	runner := report.NewRunner(sealer, verifier, epoch)
	application := service.New(repository, sealer, verifier, runner, func() time.Time { return epoch })
	server := api.NewServer(application, http.FileServer(http.Dir(webPath)))
	runtime, err := api.Listen(address, server.Handler())
	if err != nil {
		return err
	}
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	done := make(chan error, 1)
	go func() { done <- runtime.Serve() }()
	fmt.Printf("room snapshot server listening on %s\n", runtime.Address())
	select {
	case err := <-done:
		return err
	case <-signals:
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return runtime.Shutdown(ctx)
	}
}
