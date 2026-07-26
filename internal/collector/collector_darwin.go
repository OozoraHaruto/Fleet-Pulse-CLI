//go:build darwin

package collector

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"runtime"
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
	darwinGPUUtilizationPattern = regexp.MustCompile(`(?i)GPU(?:\s+HW)?\s+active\s+residency:\s*([0-9.]+)%`)
	darwinGPUTemperaturePattern = regexp.MustCompile(`(?i)GPU\s+(?:die\s+)?temperature:\s*([0-9.]+)\s*C`)
)

func collectCPU(ctx context.Context) schema.CPUSection {
	coreCount := runtime.NumCPU()
	out, err := darwinCommand(ctx, "/usr/sbin/iostat", "-c", "2", "-w", "1")
	if err != nil {
		return schema.CPUSection{
			SectionStatus: schema.SectionStatus{Status: schema.StatusAvailable, Scope: schema.ScopeHost},
			CoreCount:     coreCount,
		}
	}

	section, err := darwinCPUFromIOStat(out, coreCount)
	if err != nil {
		return schema.CPUSection{
			SectionStatus: schema.SectionStatus{Status: schema.StatusAvailable, Scope: schema.ScopeHost},
			CoreCount:     coreCount,
		}
	}
	return section
}

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

func collectDisks(ctx context.Context) schema.DisksSection {
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
			volume.Health = darwinDiskHealth(ctx, volume.MountPoint, volume.FilesystemType)
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

func collectGPU(ctx context.Context) schema.GPUSection {
	if out, err := darwinCommand(ctx, "/usr/sbin/system_profiler", "SPDisplaysDataType", "-json"); err == nil {
		if section, parseErr := darwinGPUFromSystemProfiler(out); parseErr == nil && len(section.Devices) > 0 {
			if os.Geteuid() == 0 {
				powerCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
				defer cancel()
				if powerOut, powerErr := darwinCommand(powerCtx, "/usr/bin/powermetrics", "--samplers", "gpu_power", "-n", "1", "-i", "1000"); powerErr == nil {
					for i := range section.Devices {
						darwinApplyPowermetricsGPU(&section.Devices[i], powerOut)
					}
				}
			}
			return section
		}
	}

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

func darwinCPUFromIOStat(out []byte, coreCount int) (schema.CPUSection, error) {
	lines := strings.Split(string(out), "\n")
	var utilization *float64
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) < 6 {
			continue
		}
		values := make([]float64, len(fields))
		numeric := true
		for i, field := range fields {
			value, err := strconv.ParseFloat(field, 64)
			if err != nil {
				numeric = false
				break
			}
			values[i] = value
		}
		if !numeric {
			continue
		}

		idle := values[len(values)-4]
		used := 100 - idle
		if used < 0 {
			used = 0
		}
		if used > 100 {
			used = 100
		}
		utilization = &used
	}
	if utilization == nil {
		return schema.CPUSection{}, errors.New("cpu columns missing from iostat output")
	}

	return schema.CPUSection{
		SectionStatus: schema.SectionStatus{Status: schema.StatusAvailable, Scope: schema.ScopeHost},
		CoreCount:     coreCount,
		Utilization:   utilization,
	}, nil
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

func darwinDiskHealth(ctx context.Context, mountPoint string, fsType string) *schema.DiskHealth {
	if fsType == "smbfs" {
		return &schema.DiskHealth{Status: "unsupported"}
	}
	out, err := darwinCommand(ctx, "/usr/sbin/diskutil", "info", mountPoint)
	if err != nil {
		return &schema.DiskHealth{Status: "unavailable", Warnings: []string{err.Error()}}
	}
	health, ok := darwinDiskHealthFromDiskutil(out)
	if !ok {
		return &schema.DiskHealth{Status: "unsupported"}
	}
	return &health
}

func darwinDiskHealthFromDiskutil(out []byte) (schema.DiskHealth, bool) {
	fields := darwinColonFields(string(out))
	smart := fields["SMART Status"]
	if smart == "" {
		return schema.DiskHealth{}, false
	}

	switch strings.ToLower(smart) {
	case "verified":
		return schema.DiskHealth{Status: "available"}, true
	case "not supported", "unsupported":
		return schema.DiskHealth{Status: "unsupported"}, true
	default:
		return schema.DiskHealth{Status: "available", Warnings: []string{"SMART Status: " + smart}}, true
	}
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

func darwinGPUFromSystemProfiler(out []byte) (schema.GPUSection, error) {
	var payload struct {
		Displays []map[string]any `json:"SPDisplaysDataType"`
	}
	if err := json.Unmarshal(out, &payload); err != nil {
		return schema.GPUSection{}, err
	}
	if len(payload.Displays) == 0 {
		return schema.GPUSection{}, errors.New("display devices missing from system_profiler output")
	}

	devices := make([]schema.GPUDevice, 0, len(payload.Displays))
	for _, display := range payload.Displays {
		model := firstString(display, "_name", "sppci_model", "spdisplays_name")
		vendor := darwinNormalizeGPUVendor(firstString(display, "spdisplays_vendor", "sppci_vendor"))
		if model == "" {
			continue
		}
		if vendor == "" && strings.HasPrefix(model, "Apple ") {
			vendor = "Apple"
		}
		if vendor == "Apple" && !strings.HasSuffix(model, " GPU") {
			model += " GPU"
		}

		device := schema.GPUDevice{
			Vendor: vendor,
			Model:  model,
		}
		if memory, ok := darwinParseByteSize(firstString(display, "spdisplays_vram", "sppci_vram")); ok {
			device.MemoryTotalBytes = &memory
		}
		devices = append(devices, device)
	}
	if len(devices) == 0 {
		return schema.GPUSection{}, errors.New("usable display devices missing from system_profiler output")
	}

	return schema.GPUSection{
		SectionStatus: schema.SectionStatus{Status: schema.StatusAvailable, Scope: schema.ScopeHost},
		Devices:       devices,
	}, nil
}

func darwinApplyPowermetricsGPU(device *schema.GPUDevice, out []byte) {
	if matches := darwinGPUUtilizationPattern.FindStringSubmatch(string(out)); len(matches) == 2 {
		if value, err := strconv.ParseFloat(matches[1], 64); err == nil {
			device.Utilization = &value
		}
	}
	if matches := darwinGPUTemperaturePattern.FindStringSubmatch(string(out)); len(matches) == 2 {
		if value, err := strconv.ParseFloat(matches[1], 64); err == nil {
			device.TemperatureCelsius = &value
		}
	}
}

func firstString(values map[string]any, keys ...string) string {
	for _, key := range keys {
		if value, ok := values[key].(string); ok {
			value = strings.TrimSpace(value)
			if value != "" {
				return value
			}
		}
	}
	return ""
}

func darwinNormalizeGPUVendor(value string) string {
	value = strings.TrimSpace(value)
	value = strings.TrimPrefix(value, "sppci_vendor_")
	value = strings.TrimPrefix(value, "spdisplays_vendor_")
	if strings.EqualFold(value, "apple") {
		return "Apple"
	}
	return value
}

func darwinParseByteSize(value string) (uint64, bool) {
	fields := strings.Fields(strings.TrimSpace(value))
	if len(fields) < 2 {
		return 0, false
	}
	number, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		return 0, false
	}

	multiplier := float64(1)
	switch strings.ToLower(fields[1]) {
	case "kb", "kib":
		multiplier = 1024
	case "mb", "mib":
		multiplier = 1024 * 1024
	case "gb", "gib":
		multiplier = 1024 * 1024 * 1024
	case "tb", "tib":
		multiplier = 1024 * 1024 * 1024 * 1024
	default:
		return 0, false
	}
	return uint64(number * multiplier), true
}

func darwinColonFields(text string) map[string]string {
	fields := map[string]string{}
	for _, line := range strings.Split(text, "\n") {
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		fields[strings.TrimSpace(key)] = strings.TrimSpace(value)
	}
	return fields
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
