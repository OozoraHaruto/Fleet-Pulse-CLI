package api

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/haruto/fleetpulse/internal/schema"
)

type Service interface {
	Snapshot(context.Context) schema.Snapshot
	CPU(context.Context) schema.CPUSection
	Memory(context.Context) schema.MemorySection
	Disks(context.Context) schema.DisksSection
	GPU(context.Context) schema.GPUSection
	System(context.Context) schema.SystemSection
}

func NewHandler(service Service) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		writeError(w, http.StatusNotFound, "not found")
	})
	mux.HandleFunc("/health", getOnly(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{
			"status":         "ok",
			"schema_version": "v1",
		})
	}))
	mux.HandleFunc("/v1/stats", getOnly(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, service.Snapshot(r.Context()))
	}))
	mux.HandleFunc("/v1/cpu", getOnly(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, service.CPU(r.Context()))
	}))
	mux.HandleFunc("/v1/memory", getOnly(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, service.Memory(r.Context()))
	}))
	mux.HandleFunc("/v1/disks", getOnly(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, service.Disks(r.Context()))
	}))
	mux.HandleFunc("/v1/gpu", getOnly(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, service.GPU(r.Context()))
	}))
	mux.HandleFunc("/v1/system", getOnly(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, service.System(r.Context()))
	}))
	return mux
}

func getOnly(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		next(w, r)
	}
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		http.Error(w, `{"error":"failed to encode response"}`, http.StatusInternalServerError)
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
