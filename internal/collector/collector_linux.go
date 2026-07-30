//go:build linux

package collector

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"sort"
	"syscall"
	"time"

	"github.com/haruto/fleetpulse/internal/schema"
)

const linuxCPUStatSampleInterval = 100 * time.Millisecond

func collectCPU(ctx context.Context) schema.CPUSection {
	coreCount := runtime.NumCPU()
	first, err := os.ReadFile("/proc/stat")
	if err != nil {
		return linuxCPUFallback(coreCount)
	}
	if err := linuxWait(ctx, linuxCPUStatSampleInterval); err != nil {
		return linuxCPUFallback(coreCount)
	}
	second, err := os.ReadFile("/proc/stat")
	if err != nil {
		return linuxCPUFallback(coreCount)
	}

	section, err := linuxCPUFromProcStat(first, second, coreCount)
	if err != nil {
		return linuxCPUFallback(coreCount)
	}
	return section
}

func linuxCPUFallback(coreCount int) schema.CPUSection {
	return schema.CPUSection{
		SectionStatus: schema.SectionStatus{Status: schema.StatusAvailable, Scope: schema.ScopeHost},
		CoreCount:     coreCount,
	}
}

func collectMemory(context.Context) schema.MemorySection {
	out, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return schema.MemorySection{
			SectionStatus: unavailable(fmt.Sprintf("memory collection failed: %v", err)),
		}
	}

	section, err := linuxMemoryFromMemInfo(out)
	if err != nil {
		return schema.MemorySection{
			SectionStatus: unavailable(fmt.Sprintf("memory collection failed: %v", err)),
		}
	}
	return section
}

func linuxWait(ctx context.Context, duration time.Duration) error {
	timer := time.NewTimer(duration)
	defer timer.Stop()

	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func collectDisks(context.Context) schema.DisksSection {
	out, err := os.ReadFile("/proc/mounts")
	if err != nil {
		return schema.DisksSection{
			SectionStatus: unavailable(fmt.Sprintf("disk collection failed: %v", err)),
			Volumes:       []schema.Volume{},
		}
	}

	mounts, err := linuxMountsFromProcMounts(out)
	if err != nil {
		return schema.DisksSection{
			SectionStatus: unavailable(fmt.Sprintf("disk collection failed: %v", err)),
			Volumes:       []schema.Volume{},
		}
	}

	seen := map[string]struct{}{}
	volumes := make([]schema.Volume, 0, len(mounts))
	for _, mount := range mounts {
		if _, ok := seen[mount.mountPoint]; ok {
			continue
		}
		seen[mount.mountPoint] = struct{}{}

		var stat syscall.Statfs_t
		if err := syscall.Statfs(mount.mountPoint, &stat); err != nil {
			continue
		}
		volume, ok := linuxVolumeFromStat(mount.mountPoint, mount.filesystemType, linuxMountStat{
			blockSize:       uint64(stat.Bsize),
			blockCount:      stat.Blocks,
			availableBlocks: stat.Bavail,
		})
		if ok {
			volumes = append(volumes, volume)
		}
	}

	if len(volumes) == 0 {
		return schema.DisksSection{
			SectionStatus: unavailable("disk collection returned no usable mounted filesystems"),
			Volumes:       []schema.Volume{},
		}
	}
	sort.Slice(volumes, func(i, j int) bool {
		return volumes[i].MountPoint < volumes[j].MountPoint
	})
	return schema.DisksSection{
		SectionStatus: schema.SectionStatus{Status: schema.StatusAvailable, Scope: schema.ScopeHost},
		Volumes:       volumes,
	}
}

func collectGPU(ctx context.Context) schema.GPUSection {
	return newLinuxGPUCollector().collect(ctx)
}

func collectSystem(_ context.Context, _ time.Time) schema.SystemSection {
	uptimeOut, uptimeErr := os.ReadFile("/proc/uptime")
	loadOut, loadErr := os.ReadFile("/proc/loadavg")
	if uptimeErr != nil && loadErr != nil {
		return schema.SystemSection{
			SectionStatus: unavailable(fmt.Sprintf("system collection failed: uptime: %v; load average: %v", uptimeErr, loadErr)),
		}
	}

	section, err := linuxSystemFromProc(uptimeOut, loadOut)
	if err != nil {
		return schema.SystemSection{
			SectionStatus: unavailable(fmt.Sprintf("system collection failed: %v", err)),
		}
	}
	return section
}
