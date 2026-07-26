//go:build darwin

package collector

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/haruto/fleetpulse/internal/schema"
)

var (
	darwinVMStatPageSizePattern = regexp.MustCompile(`page size of (\d+) bytes`)
	darwinVMStatLinePattern     = regexp.MustCompile(`^"?([^":]+)"?:\s+(\d+)\.$`)
	darwinBootTimePattern       = regexp.MustCompile(`sec\s*=\s*(\d+)`)
	darwinLoadAveragePattern    = regexp.MustCompile(`load averages?:\s*([0-9.]+)\s+([0-9.]+)\s+([0-9.]+)`)
)

func collectMemory(ctx context.Context) schema.MemorySection {
	out, err := darwinCommand(ctx, "/usr/bin/vm_stat")
	if err != nil {
		return schema.MemorySection{
			SectionStatus: unavailable(fmt.Sprintf("memory collection failed: %v", err)),
		}
	}

	section, err := darwinMemoryFromVMStat(out, darwinPhysicalMemoryBytes())
	if err != nil {
		return schema.MemorySection{
			SectionStatus: unavailable(fmt.Sprintf("memory collection failed: %v", err)),
		}
	}
	return section
}

func collectDisks(context.Context) schema.DisksSection {
	n, err := syscall.Getfsstat(nil, 0)
	if err != nil {
		return schema.DisksSection{
			SectionStatus: unavailable(fmt.Sprintf("disk collection failed: %v", err)),
			Volumes:       []schema.Volume{},
		}
	}
	if n <= 0 {
		return schema.DisksSection{
			SectionStatus: unavailable("disk collection returned no mounted filesystems"),
			Volumes:       []schema.Volume{},
		}
	}

	stats := make([]syscall.Statfs_t, n)
	n, err = syscall.Getfsstat(stats, 0)
	if err != nil {
		return schema.DisksSection{
			SectionStatus: unavailable(fmt.Sprintf("disk collection failed: %v", err)),
			Volumes:       []schema.Volume{},
		}
	}

	volumes := make([]schema.Volume, 0, n)
	for _, stat := range stats[:n] {
		volume, ok := darwinVolumeFromStatfs(stat)
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
	return schema.DisksSection{
		SectionStatus: schema.SectionStatus{Status: schema.StatusAvailable, Scope: schema.ScopeHost},
		Volumes:       volumes,
	}
}

func collectGPU(context.Context) schema.GPUSection {
	cpuBrand, _ := darwinSysctlString("machdep.cpu.brand_string")
	hardwareModel, _ := darwinSysctlString("hw.model")
	return darwinGPUFromHardwareIdentity(cpuBrand, hardwareModel)
}

func collectSystem(ctx context.Context, now time.Time) schema.SystemSection {
	section := schema.SystemSection{
		SectionStatus: schema.SectionStatus{Status: schema.StatusAvailable, Scope: schema.ScopeHost},
	}

	bootTime, bootErr := darwinBootTimeSeconds(ctx)
	if bootErr == nil && bootTime > 0 && now.Unix() >= bootTime {
		uptime := uint64(now.Unix() - bootTime)
		section.UptimeSeconds = &uptime
	}

	out, loadCmdErr := darwinCommand(ctx, "/usr/bin/uptime")
	if loadCmdErr == nil {
		load, err := darwinParseLoadAverage(string(out))
		if err == nil {
			section.LoadAverage = &load
		} else {
			loadCmdErr = err
		}
	}

	if section.UptimeSeconds == nil && section.LoadAverage == nil {
		return schema.SystemSection{
			SectionStatus: unavailable(fmt.Sprintf("system collection failed: boot time: %v; load average: %v", bootErr, loadCmdErr)),
		}
	}
	return section
}

func darwinCommand(ctx context.Context, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	return cmd.Output()
}

func darwinMemoryFromVMStat(out []byte, physicalMemoryBytes uint64) (schema.MemorySection, error) {
	text := string(out)
	pageSizeMatches := darwinVMStatPageSizePattern.FindStringSubmatch(text)
	if len(pageSizeMatches) != 2 {
		return schema.MemorySection{}, errors.New("page size missing from vm_stat output")
	}
	pageSize, err := strconv.ParseUint(pageSizeMatches[1], 10, 64)
	if err != nil {
		return schema.MemorySection{}, fmt.Errorf("page size: %w", err)
	}

	pages := map[string]uint64{}
	for _, line := range strings.Split(text, "\n") {
		matches := darwinVMStatLinePattern.FindStringSubmatch(strings.TrimSpace(line))
		if len(matches) != 3 {
			continue
		}
		count, err := strconv.ParseUint(matches[2], 10, 64)
		if err != nil {
			return schema.MemorySection{}, fmt.Errorf("%s: %w", matches[1], err)
		}
		pages[strings.ToLower(matches[1])] = count
	}

	free := pageBytes(pageSize, pages["pages free"])
	available := pageBytes(pageSize, pages["pages free"]+pages["pages inactive"]+pages["pages speculative"]+pages["pages purgeable"])
	total := physicalMemoryBytes
	if total == 0 {
		total = pageBytes(pageSize, pages["pages free"]+pages["pages active"]+pages["pages inactive"]+pages["pages speculative"]+pages["pages wired down"]+pages["pages occupied by compressor"])
	}
	if total == 0 {
		return schema.MemorySection{}, errors.New("memory totals missing from vm_stat output")
	}
	if available > total {
		available = total
	}
	used := total - available

	return schema.MemorySection{
		SectionStatus:  schema.SectionStatus{Status: schema.StatusAvailable, Scope: schema.ScopeHost},
		TotalBytes:     &total,
		UsedBytes:      &used,
		FreeBytes:      &free,
		AvailableBytes: &available,
	}, nil
}

func pageBytes(pageSize uint64, pages uint64) uint64 {
	return pageSize * pages
}

func darwinPhysicalMemoryBytes() uint64 {
	value, err := darwinSysctlUint64("hw.memsize")
	if err != nil {
		return 0
	}
	return value
}

func darwinBootTimeSeconds(ctx context.Context) (int64, error) {
	value, err := syscall.Sysctl("kern.boottime")
	if err == nil {
		bytesValue := []byte(value)
		if len(bytesValue) >= 8 {
			return int64(binary.LittleEndian.Uint64(bytesValue[:8])), nil
		}
	}

	out, cmdErr := darwinCommand(ctx, "/usr/sbin/sysctl", "kern.boottime")
	if cmdErr != nil {
		if err != nil {
			return 0, err
		}
		return 0, cmdErr
	}
	return darwinParseBootTime(string(out))
}

func darwinSysctlUint64(name string) (uint64, error) {
	value, err := syscall.Sysctl(name)
	if err != nil {
		return 0, err
	}
	bytesValue := []byte(value)
	if len(bytesValue) >= 8 {
		return binary.LittleEndian.Uint64(bytesValue[:8]), nil
	}

	text := strings.Trim(value, "\x00 \n\t")
	if text == "" {
		return 0, errors.New("empty sysctl value")
	}
	return strconv.ParseUint(text, 10, 64)
}

func darwinSysctlString(name string) (string, error) {
	value, err := syscall.Sysctl(name)
	if err != nil {
		return "", err
	}
	return strings.Trim(value, "\x00 \n\t"), nil
}

func darwinParseBootTime(out string) (int64, error) {
	matches := darwinBootTimePattern.FindStringSubmatch(out)
	if len(matches) != 2 {
		return 0, errors.New("boot time seconds missing")
	}
	value, err := strconv.ParseInt(matches[1], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("boot time seconds: %w", err)
	}
	return value, nil
}

func darwinParseLoadAverage(out string) (schema.LoadAverage, error) {
	matches := darwinLoadAveragePattern.FindStringSubmatch(out)
	if len(matches) != 4 {
		return schema.LoadAverage{}, errors.New("load averages missing")
	}

	values := [3]float64{}
	for i := range values {
		value, err := strconv.ParseFloat(matches[i+1], 64)
		if err != nil {
			return schema.LoadAverage{}, fmt.Errorf("load average %d: %w", i+1, err)
		}
		values[i] = value
	}
	return schema.LoadAverage{
		OneMinute:      values[0],
		FiveMinutes:    values[1],
		FifteenMinutes: values[2],
	}, nil
}

func darwinVolumeFromStatfs(stat syscall.Statfs_t) (schema.Volume, bool) {
	mountPoint := int8ArrayToString(stat.Mntonname[:])
	fsType := int8ArrayToString(stat.Fstypename[:])
	source := int8ArrayToString(stat.Mntfromname[:])
	if darwinSkipVolume(mountPoint, fsType, source) {
		return schema.Volume{}, false
	}

	total := uint64(stat.Bsize) * stat.Blocks
	free := uint64(stat.Bsize) * stat.Bavail
	used := total - free
	percentUsed := 0.0
	if total > 0 {
		percentUsed = (float64(used) / float64(total)) * 100
	}

	return schema.Volume{
		MountPoint:     mountPoint,
		FilesystemType: fsType,
		TotalBytes:     &total,
		UsedBytes:      &used,
		FreeBytes:      &free,
		PercentUsed:    &percentUsed,
		Health:         &schema.DiskHealth{Status: "unsupported"},
	}, true
}

func darwinSkipVolume(mountPoint string, fsType string, source string) bool {
	if mountPoint == "" || fsType == "" {
		return true
	}
	if fsType == "devfs" || fsType == "autofs" {
		return true
	}
	if strings.HasPrefix(source, "map ") {
		return true
	}
	if strings.HasPrefix(mountPoint, "/private/var/run/") {
		return true
	}
	if strings.HasPrefix(mountPoint, "/Library/Developer/CoreSimulator/") {
		return true
	}
	if strings.HasPrefix(mountPoint, "/System/Volumes/") && mountPoint != "/System/Volumes/Data" {
		return true
	}
	return false
}

func int8ArrayToString(values []int8) string {
	buf := make([]byte, 0, len(values))
	for _, value := range values {
		if value == 0 {
			break
		}
		buf = append(buf, byte(value))
	}
	return string(buf)
}

func darwinGPUFromHardwareIdentity(cpuBrand string, hardwareModel string) schema.GPUSection {
	cpuBrand = strings.TrimSpace(cpuBrand)
	if strings.HasPrefix(cpuBrand, "Apple ") {
		model := cpuBrand
		if !strings.HasSuffix(model, " GPU") {
			model += " GPU"
		}
		return schema.GPUSection{
			SectionStatus: schema.SectionStatus{Status: schema.StatusAvailable, Scope: schema.ScopeHost},
			Devices: []schema.GPUDevice{
				{Vendor: "Apple", Model: model},
			},
		}
	}

	reason := "privacy-safe gpu collector could not identify GPU"
	if strings.TrimSpace(hardwareModel) != "" {
		reason += " for " + strings.TrimSpace(hardwareModel)
	}
	return schema.GPUSection{
		SectionStatus: unsupported(reason),
	}
}
