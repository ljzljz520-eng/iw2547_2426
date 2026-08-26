package api

import (
	"example.com/roomsnapshot/internal/batch"
	"example.com/roomsnapshot/internal/cryptoiface"
	"example.com/roomsnapshot/internal/report"
	"example.com/roomsnapshot/internal/service"
	"example.com/roomsnapshot/internal/store"
	"example.com/roomsnapshot/internal/verify"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestHealthEndpoint(t *testing.T) {
	repository, err := store.Open(t.TempDir() + "/api.db")
	if err != nil {
		t.Fatal(err)
	}
	defer repository.Close()
	codec, _ := cryptoiface.NewCodec([]byte("0123456789abcdef"))
	sealer := batch.NewSealer(codec)
	epoch := time.Date(2026, 8, 26, 0, 0, 0, 0, time.UTC)
	verifier := verify.New(sealer, nil)
	application := service.New(repository, sealer, verifier, report.NewRunner(sealer, verifier, epoch), nil)
	request := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	response := httptest.NewRecorder()
	NewServer(application, nil).Handler().ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status=%d", response.Code)
	}
}
