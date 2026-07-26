package api_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/haruto/fleetpulse/internal/api"
	"github.com/haruto/fleetpulse/internal/schema"
)

type staticService struct {
	snapshot schema.Snapshot
}

func (s staticService) Snapshot(context.Context) schema.Snapshot {
	return s.snapshot
}

func (s staticService) CPU(context.Context) schema.CPUSection {
	return s.snapshot.CPU
}

func (s staticService) Memory(context.Context) schema.MemorySection {
	return s.snapshot.Memory
}

func (s staticService) Disks(context.Context) schema.DisksSection {
	return s.snapshot.Disks
}

func (s staticService) GPU(context.Context) schema.GPUSection {
	return s.snapshot.GPU
}

func (s staticService) System(context.Context) schema.SystemSection {
	return s.snapshot.System
}

func TestHealthReturnsJSON(t *testing.T) {
	handler := api.NewHandler(staticService{snapshot: testSnapshot()})

	res := request(handler, http.MethodGet, "/health")

	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusOK)
	}
	assertJSONContentType(t, res)

	var body map[string]string
	if err := json.Unmarshal(res.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["status"] != "ok" {
		t.Fatalf("status body = %q, want ok", body["status"])
	}
	if body["schema_version"] != "v1" {
		t.Fatalf("schema_version = %q, want v1", body["schema_version"])
	}
}

func TestVersionedEndpointsReturnExpectedSections(t *testing.T) {
	snapshot := testSnapshot()
	handler := api.NewHandler(staticService{snapshot: snapshot})

	tests := []struct {
		path       string
		assertBody func(*testing.T, []byte)
	}{
		{
			path: "/v1/stats",
			assertBody: func(t *testing.T, body []byte) {
				var got schema.Snapshot
				if err := json.Unmarshal(body, &got); err != nil {
					t.Fatal(err)
				}
				if got.Target.Hostname != "node-a" {
					t.Fatalf("hostname = %q, want node-a", got.Target.Hostname)
				}
			},
		},
		{
			path: "/v1/cpu",
			assertBody: func(t *testing.T, body []byte) {
				var got schema.CPUSection
				if err := json.Unmarshal(body, &got); err != nil {
					t.Fatal(err)
				}
				if got.CoreCount != 8 {
					t.Fatalf("core_count = %d, want 8", got.CoreCount)
				}
			},
		},
		{
			path: "/v1/memory",
			assertBody: func(t *testing.T, body []byte) {
				var got schema.MemorySection
				if err := json.Unmarshal(body, &got); err != nil {
					t.Fatal(err)
				}
				if got.Status != schema.StatusUnavailable {
					t.Fatalf("memory status = %q, want %q", got.Status, schema.StatusUnavailable)
				}
			},
		},
		{
			path: "/v1/disks",
			assertBody: func(t *testing.T, body []byte) {
				var got schema.DisksSection
				if err := json.Unmarshal(body, &got); err != nil {
					t.Fatal(err)
				}
				if len(got.Volumes) != 1 {
					t.Fatalf("volume count = %d, want 1", len(got.Volumes))
				}
			},
		},
		{
			path: "/v1/gpu",
			assertBody: func(t *testing.T, body []byte) {
				var got schema.GPUSection
				if err := json.Unmarshal(body, &got); err != nil {
					t.Fatal(err)
				}
				if got.Status != schema.StatusUnsupported {
					t.Fatalf("gpu status = %q, want %q", got.Status, schema.StatusUnsupported)
				}
			},
		},
		{
			path: "/v1/system",
			assertBody: func(t *testing.T, body []byte) {
				var got schema.SystemSection
				if err := json.Unmarshal(body, &got); err != nil {
					t.Fatal(err)
				}
				if got.Status != schema.StatusAvailable {
					t.Fatalf("system status = %q, want %q", got.Status, schema.StatusAvailable)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			res := request(handler, http.MethodGet, tt.path)
			if res.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d; body = %s", res.Code, http.StatusOK, res.Body.String())
			}
			assertJSONContentType(t, res)
			tt.assertBody(t, res.Body.Bytes())
		})
	}
}

func TestRoutesReturnJSONErrors(t *testing.T) {
	handler := api.NewHandler(staticService{snapshot: testSnapshot()})

	tests := []struct {
		method string
		path   string
		want   int
	}{
		{method: http.MethodPost, path: "/v1/stats", want: http.StatusMethodNotAllowed},
		{method: http.MethodGet, path: "/missing", want: http.StatusNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.method+" "+tt.path, func(t *testing.T) {
			res := request(handler, tt.method, tt.path)
			if res.Code != tt.want {
				t.Fatalf("status = %d, want %d", res.Code, tt.want)
			}
			assertJSONContentType(t, res)
			var body map[string]string
			if err := json.Unmarshal(res.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			if body["error"] == "" {
				t.Fatalf("error body is empty: %s", res.Body.String())
			}
		})
	}
}

func request(handler http.Handler, method string, path string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, nil)
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)
	return res
}

func assertJSONContentType(t *testing.T, res *httptest.ResponseRecorder) {
	t.Helper()
	if got := res.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("Content-Type = %q, want application/json", got)
	}
}

func testSnapshot() schema.Snapshot {
	total := uint64(100)
	used := uint64(40)
	free := uint64(60)
	percent := 40.0
	return schema.Snapshot{
		Timestamp:     time.Date(2026, 7, 26, 1, 2, 3, 0, time.UTC),
		SchemaVersion: "v1",
		Target: schema.TargetIdentity{
			Hostname:     "node-a",
			Platform:     "darwin",
			Architecture: "arm64",
		},
		System: schema.SystemSection{
			SectionStatus: schema.SectionStatus{Status: schema.StatusAvailable, Scope: schema.ScopeHost},
			UptimeSeconds: ptr(uint64(123)),
			LoadAverage:   &schema.LoadAverage{OneMinute: 1.1, FiveMinutes: 1.2, FifteenMinutes: 1.3},
		},
		CPU: schema.CPUSection{
			SectionStatus: schema.SectionStatus{Status: schema.StatusAvailable, Scope: schema.ScopeHost},
			CoreCount:     8,
		},
		Memory: schema.MemorySection{
			SectionStatus: schema.SectionStatus{Status: schema.StatusUnavailable, Scope: schema.ScopeUnavailable, Error: "permission denied"},
		},
		Disks: schema.DisksSection{
			SectionStatus: schema.SectionStatus{Status: schema.StatusAvailable, Scope: schema.ScopeHost},
			Volumes: []schema.Volume{
				{
					MountPoint:  "/",
					TotalBytes:  &total,
					UsedBytes:   &used,
					FreeBytes:   &free,
					PercentUsed: &percent,
					Health:      &schema.DiskHealth{Status: "unsupported"},
				},
			},
		},
		GPU: schema.GPUSection{
			SectionStatus: schema.SectionStatus{Status: schema.StatusUnsupported, Scope: schema.ScopeUnavailable},
		},
	}
}

func ptr[T any](v T) *T {
	return &v
}
