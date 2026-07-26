package api

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/haruto/fleetpulse/internal/schema"
)

type Options struct {
	AuthEnabled bool
	BearerToken string
}

type Service interface {
	Snapshot(context.Context) schema.Snapshot
	CPU(context.Context) schema.CPUSection
	Memory(context.Context) schema.MemorySection
	Disks(context.Context) schema.DisksSection
	GPU(context.Context) schema.GPUSection
	System(context.Context) schema.SystemSection
	Health(context.Context) schema.Health
}

func NewHandler(service Service) http.Handler {
	return NewHandlerWithOptions(service, Options{})
}

func NewHandlerWithOptions(service Service, options Options) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		writeError(w, http.StatusNotFound, "not found")
	})
	mux.HandleFunc("/health", getOnly(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, service.Health(r.Context()))
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
	if options.AuthEnabled {
		return authMiddleware(mux, options.BearerToken)
	}
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

func authMiddleware(next http.Handler, bearerToken string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
		if !ok || subtle.ConstantTimeCompare([]byte(got), []byte(bearerToken)) != 1 {
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		next.ServeHTTP(w, r)
	})
}
