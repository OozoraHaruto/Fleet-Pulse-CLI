package collector

import (
	"context"
	"os"
	"runtime"
	"time"

	"github.com/haruto/fleetpulse/internal/schema"
)

const schemaVersion = "v1"

type Service struct {
	now func() time.Time
}

func NewService() *Service {
	return &Service{now: func() time.Time { return time.Now().UTC() }}
}

func (s *Service) Snapshot(ctx context.Context) schema.Snapshot {
	return schema.Snapshot{
		Timestamp:     s.now(),
		SchemaVersion: schemaVersion,
		Target:        s.target(),
		System:        s.System(ctx),
		CPU:           s.CPU(ctx),
		Memory:        s.Memory(ctx),
		Disks:         s.Disks(ctx),
		GPU:           s.GPU(ctx),
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

func unsupported(reason string) schema.SectionStatus {
	return schema.SectionStatus{
		Status: schema.StatusUnsupported,
		Scope:  schema.ScopeUnavailable,
		Error:  reason,
	}
}
