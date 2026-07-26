//go:build !darwin

package collector

import (
	"context"
	"time"

	"github.com/haruto/fleetpulse/internal/schema"
)

func collectMemory(context.Context) schema.MemorySection {
	return schema.MemorySection{
		SectionStatus: unsupported("memory collection requires a platform-specific collector"),
	}
}

func collectDisks(context.Context) schema.DisksSection {
	return schema.DisksSection{
		SectionStatus: unsupported("disk collection requires a platform-specific collector"),
		Volumes:       []schema.Volume{},
	}
}

func collectGPU(context.Context) schema.GPUSection {
	return schema.GPUSection{
		SectionStatus: unsupported("gpu collection requires a vendor-specific collector"),
	}
}

func collectSystem(context.Context, time.Time) schema.SystemSection {
	return schema.SystemSection{
		SectionStatus: unsupported("system uptime and load require a platform-specific collector"),
	}
}
