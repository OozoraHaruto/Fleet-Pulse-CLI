package collector

import (
	"context"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/haruto/fleetpulse/internal/schema"
)

const schemaVersion = "v1"

type Options struct {
	Now      func() time.Time
	CacheTTL time.Duration
	Collect  func(context.Context) (schema.Snapshot, error)
}

type Service struct {
	now       func() time.Time
	cacheTTL  time.Duration
	collect   func(context.Context) (schema.Snapshot, error)
	mu        sync.Mutex
	cache     schema.Snapshot
	cacheAt   time.Time
	hasCache  bool
	lastError string
}

func NewService() *Service {
	return NewServiceWithOptions(Options{})
}

func NewServiceWithOptions(options Options) *Service {
	if options.Now == nil {
		options.Now = func() time.Time { return time.Now().UTC() }
	}
	service := &Service{
		now:      options.Now,
		cacheTTL: options.CacheTTL,
	}
	if options.Collect != nil {
		service.collect = options.Collect
	} else {
		service.collect = service.collectLive
	}
	return service
}

func (s *Service) Snapshot(ctx context.Context) schema.Snapshot {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := s.now()
	if s.hasCache && s.cacheTTL > 0 && now.Sub(s.cacheAt) < s.cacheTTL {
		return s.cache
	}

	snapshot, err := s.collect(ctx)
	if err != nil {
		s.lastError = err.Error()
		if s.hasCache {
			return s.cache
		}
		return s.unavailableSnapshot(now, err)
	}

	s.lastError = ""
	s.cache = snapshot
	s.cacheAt = now
	s.hasCache = true
	return snapshot
}

func (s *Service) collectLive(ctx context.Context) (schema.Snapshot, error) {
	return schema.Snapshot{
		Timestamp:     s.now(),
		SchemaVersion: schemaVersion,
		Target:        s.target(),
		System:        s.System(ctx),
		CPU:           s.CPU(ctx),
		Memory:        s.Memory(ctx),
		Disks:         s.Disks(ctx),
		GPU:           s.GPU(ctx),
	}, nil
}

func (s *Service) Health(context.Context) schema.Health {
	s.mu.Lock()
	defer s.mu.Unlock()

	collectionStatus := "ok"
	if s.lastError != "" {
		collectionStatus = "degraded"
	}
	return schema.Health{
		Status:           "ok",
		SchemaVersion:    schemaVersion,
		CollectionStatus: collectionStatus,
		Timestamp:        s.now(),
		LastError:        s.lastError,
	}
}

func (s *Service) CPU(context.Context) schema.CPUSection {
	return schema.CPUSection{
		SectionStatus: schema.SectionStatus{
			Status: schema.StatusAvailable,
			Scope:  schema.ScopeHost,
		},
		CoreCount: runtime.NumCPU(),
	}
}

func (s *Service) Memory(context.Context) schema.MemorySection {
	return schema.MemorySection{
		SectionStatus: unsupported("memory collection requires a platform-specific collector"),
	}
}

func (s *Service) Disks(context.Context) schema.DisksSection {
	return schema.DisksSection{
		SectionStatus: unsupported("disk collection requires a platform-specific collector"),
		Volumes:       []schema.Volume{},
	}
}

func (s *Service) GPU(context.Context) schema.GPUSection {
	return schema.GPUSection{
		SectionStatus: unsupported("gpu collection requires a vendor-specific collector"),
	}
}

func (s *Service) System(context.Context) schema.SystemSection {
	return schema.SystemSection{
		SectionStatus: unsupported("system uptime and load require a platform-specific collector"),
	}
}

func (s *Service) target() schema.TargetIdentity {
	hostname, err := os.Hostname()
	if err != nil || hostname == "" {
		hostname = "unknown"
	}

	return schema.TargetIdentity{
		Hostname:     hostname,
		Platform:     runtime.GOOS,
		Architecture: runtime.GOARCH,
	}
}

func (s *Service) unavailableSnapshot(now time.Time, err error) schema.Snapshot {
	status := schema.SectionStatus{
		Status: schema.StatusUnavailable,
		Scope:  schema.ScopeUnavailable,
		Error:  err.Error(),
	}
	return schema.Snapshot{
		Timestamp:     now,
		SchemaVersion: schemaVersion,
		Target:        s.target(),
		System:        schema.SystemSection{SectionStatus: status},
		CPU:           schema.CPUSection{SectionStatus: status},
		Memory:        schema.MemorySection{SectionStatus: status},
		Disks:         schema.DisksSection{SectionStatus: status, Volumes: []schema.Volume{}},
		GPU:           schema.GPUSection{SectionStatus: status},
	}
}

func unsupported(reason string) schema.SectionStatus {
	return schema.SectionStatus{
		Status: schema.StatusUnsupported,
		Scope:  schema.ScopeUnavailable,
		Error:  reason,
	}
}
