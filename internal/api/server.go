package api

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"example.com/roomsnapshot/internal/model"
	"example.com/roomsnapshot/internal/service"
	"example.com/roomsnapshot/internal/store"
)

type Server struct {
	service *service.Service
	static  http.Handler
	mux     *http.ServeMux
}

func NewServer(application *service.Service, static http.Handler) *Server {
	server := &Server{service: application, static: static, mux: http.NewServeMux()}
	server.routes()
	return server
}
func (server *Server) Handler() http.Handler { return requestLog(server.mux) }
func (server *Server) routes() {
	server.mux.HandleFunc("GET /api/health", server.health)
	server.mux.HandleFunc("GET /api/statistics", server.statistics)
	server.mux.HandleFunc("GET /api/batches", server.batches)
	server.mux.HandleFunc("POST /api/snapshots", server.capture)
	server.mux.HandleFunc("POST /api/verify/latest", server.verifyLatest)
	server.mux.HandleFunc("GET /api/reports", server.reports)
	server.mux.HandleFunc("POST /api/reports", server.generateReports)
	server.mux.HandleFunc("GET /api/minutes/{minute}", server.minuteSummary)
	if server.static != nil {
		server.mux.Handle("/", server.static)
	}
}
func (server *Server) health(writer http.ResponseWriter, _ *http.Request) {
	writeJSON(writer, http.StatusOK, map[string]string{"status": "ready"})
}
func (server *Server) statistics(writer http.ResponseWriter, _ *http.Request) {
	value, err := server.service.Statistics()
	if err != nil {
		writeError(writer, err)
		return
	}
	writeJSON(writer, http.StatusOK, value)
}
func (server *Server) batches(writer http.ResponseWriter, request *http.Request) {
	minimum, _ := strconv.Atoi(request.URL.Query().Get("minimum_details"))
	values, err := server.service.QueryBatches(service.BatchFilter{Status: request.URL.Query().Get("status"), MinimumDetails: minimum})
	if err != nil {
		writeError(writer, err)
		return
	}
	writeJSON(writer, http.StatusOK, values)
}

type capturePayload struct {
	Minute            string `json:"minute"`
	TemperatureMilliC int    `json:"temperature_milli_c"`
	CurrentMilliAmp   int    `json:"current_milli_amp"`
	Alert             string `json:"alert"`
}

func (server *Server) capture(writer http.ResponseWriter, request *http.Request) {
	var payload capturePayload
	if err := decodeJSON(request.Body, &payload); err != nil {
		writeJSON(writer, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	minute, err := time.Parse(time.RFC3339, payload.Minute)
	if err != nil {
		writeJSON(writer, http.StatusBadRequest, map[string]string{"error": "minute must use RFC3339"})
		return
	}
	reading := model.SensorReading{ID: "reading-" + minute.UTC().Format("200601021504"), RecordedMinute: model.NormalizeMinute(minute), TemperatureMilliC: payload.TemperatureMilliC, CurrentMilliAmp: payload.CurrentMilliAmp, Sequence: 0}
	alert := model.AlertSummary{ID: "alert-" + minute.UTC().Format("200601021504"), RecordedMinute: model.NormalizeMinute(minute), Severity: "info", Code: "OPERATOR", Message: strings.TrimSpace(payload.Alert), Sequence: 1}
	value, err := server.service.Capture(service.CaptureRequest{Minute: minute, Readings: []model.SensorReading{reading}, Alerts: []model.AlertSummary{alert}})
	if err != nil {
		writeError(writer, err)
		return
	}
	writeJSON(writer, http.StatusCreated, value)
}
func (server *Server) verifyLatest(writer http.ResponseWriter, _ *http.Request) {
	values, err := server.service.VerifyLatest()
	if err != nil {
		writeError(writer, err)
		return
	}
	writeJSON(writer, http.StatusOK, values)
}
func (server *Server) reports(writer http.ResponseWriter, _ *http.Request) {
	values, err := server.service.Reports()
	if err != nil {
		writeError(writer, err)
		return
	}
	writeJSON(writer, http.StatusOK, values)
}
func (server *Server) generateReports(writer http.ResponseWriter, _ *http.Request) {
	values, err := server.service.GenerateReports([]int{4, 16, 64, 256})
	if err != nil {
		writeError(writer, err)
		return
	}
	writeJSON(writer, http.StatusCreated, values)
}
func (server *Server) minuteSummary(writer http.ResponseWriter, request *http.Request) {
	minute, err := time.Parse("200601021504", request.PathValue("minute"))
	if err != nil {
		writeJSON(writer, http.StatusBadRequest, map[string]string{"error": "minute must use YYYYMMDDHHMM"})
		return
	}
	value, err := server.service.SummarizeMinute(minute)
	if err != nil {
		writeError(writer, err)
		return
	}
	writeJSON(writer, http.StatusOK, value)
}
func decodeJSON(reader io.Reader, target any) error {
	decoder := json.NewDecoder(io.LimitReader(reader, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return errors.New("request must contain one JSON document")
	}
	return nil
}
func writeJSON(writer http.ResponseWriter, status int, value any) {
	writer.Header().Set("content-type", "application/json")
	writer.WriteHeader(status)
	json.NewEncoder(writer).Encode(value)
}
func writeError(writer http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	if errors.Is(err, store.ErrNotFound) {
		status = http.StatusNotFound
	}
	writeJSON(writer, status, map[string]string{"error": err.Error()})
}
func requestLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("x-content-type-options", "nosniff")
		next.ServeHTTP(writer, request)
	})
}
