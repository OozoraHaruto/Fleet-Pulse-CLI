package collector

import (
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/haruto/fleetpulse/internal/schema"
)

type linuxMount struct {
	mountPoint     string
	filesystemType string
}

type linuxMountStat struct {
	blockSize       uint64
	blockCount      uint64
	availableBlocks uint64
}

type linuxCPUStat struct {
	total uint64
	idle  uint64
}

func linuxCPUFromProcStat(firstOut []byte, secondOut []byte, coreCount int) (schema.CPUSection, error) {
	first, err := linuxCPUStatsFromProcStat(firstOut)
	if err != nil {
		return schema.CPUSection{}, err
	}
	second, err := linuxCPUStatsFromProcStat(secondOut)
	if err != nil {
		return schema.CPUSection{}, err
	}

	firstAggregate, ok := first["cpu"]
	if !ok {
		return schema.CPUSection{}, errors.New("aggregate cpu line missing from /proc/stat")
	}
	secondAggregate, ok := second["cpu"]
	if !ok {
		return schema.CPUSection{}, errors.New("aggregate cpu line missing from /proc/stat")
	}

	utilization, ok := linuxCPUUtilization(firstAggregate, secondAggregate)
	if !ok {
		return schema.CPUSection{}, errors.New("aggregate cpu counters did not advance")
	}

	section := schema.CPUSection{
		SectionStatus: schema.SectionStatus{Status: schema.StatusAvailable, Scope: schema.ScopeHost},
		CoreCount:     coreCount,
		Utilization:   &utilization,
	}

	coreNames := linuxCPUCoreNames(first, second)
	section.PerCoreUtilization = make([]float64, 0, len(coreNames))
	for _, name := range coreNames {
		coreUtilization, ok := linuxCPUUtilization(first[name], second[name])
		if ok {
			section.PerCoreUtilization = append(section.PerCoreUtilization, coreUtilization)
		}
	}

	return section, nil
}

func linuxCPUStatsFromProcStat(out []byte) (map[string]linuxCPUStat, error) {
	stats := map[string]linuxCPUStat{}
	for _, line := range strings.Split(string(out), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 0 || !strings.HasPrefix(fields[0], "cpu") {
			continue
		}
		if len(fields) < 5 {
			return nil, fmt.Errorf("%s: cpu counters missing from /proc/stat", fields[0])
		}

		values := make([]uint64, len(fields)-1)
		for i, field := range fields[1:] {
			value, err := strconv.ParseUint(field, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("%s counter %d: %w", fields[0], i+1, err)
			}
			values[i] = value
		}

		idle := values[3]
		if len(values) > 4 {
			idle += values[4]
		}

		nonIdle := values[0] + values[1] + values[2]
		if len(values) > 5 {
			nonIdle += values[5]
		}
		if len(values) > 6 {
			nonIdle += values[6]
		}
		if len(values) > 7 {
			nonIdle += values[7]
		}

		stats[fields[0]] = linuxCPUStat{
			total: idle + nonIdle,
			idle:  idle,
		}
	}
	if len(stats) == 0 {
		return nil, errors.New("cpu lines missing from /proc/stat")
	}
	return stats, nil
}

func linuxCPUUtilization(first linuxCPUStat, second linuxCPUStat) (float64, bool) {
	if second.total < first.total || second.idle < first.idle {
		return 0, false
	}
	totalDelta := second.total - first.total
	if totalDelta == 0 {
		return 0, false
	}
	idleDelta := second.idle - first.idle
	if idleDelta > totalDelta {
		return 0, false
	}

	utilization := (float64(totalDelta-idleDelta) / float64(totalDelta)) * 100
	if utilization < 0 {
		utilization = 0
	}
	if utilization > 100 {
		utilization = 100
	}
	return utilization, true
}

func linuxCPUCoreNames(first map[string]linuxCPUStat, second map[string]linuxCPUStat) []string {
	coreNames := []string{}
	for name := range first {
		if name == "cpu" || !strings.HasPrefix(name, "cpu") {
			continue
		}
		if _, ok := second[name]; ok {
			coreNames = append(coreNames, name)
		}
	}
	sort.Slice(coreNames, func(i, j int) bool {
		left, leftErr := strconv.Atoi(strings.TrimPrefix(coreNames[i], "cpu"))
		right, rightErr := strconv.Atoi(strings.TrimPrefix(coreNames[j], "cpu"))
		if leftErr == nil && rightErr == nil {
			return left < right
		}
		return coreNames[i] < coreNames[j]
	})
	return coreNames
}

func linuxMemoryFromMemInfo(out []byte) (schema.MemorySection, error) {
	values := map[string]uint64{}
	for _, line := range strings.Split(string(out), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		key := strings.TrimSuffix(fields[0], ":")
		value, err := strconv.ParseUint(fields[1], 10, 64)
		if err != nil {
			return schema.MemorySection{}, fmt.Errorf("%s: %w", key, err)
		}
		if len(fields) >= 3 && !strings.EqualFold(fields[2], "kB") {
			return schema.MemorySection{}, fmt.Errorf("%s: unsupported unit %q", key, fields[2])
		}
		values[key] = value * 1024
	}

	total := values["MemTotal"]
	if total == 0 {
		return schema.MemorySection{}, errors.New("MemTotal missing from /proc/meminfo")
	}

	free := values["MemFree"]
	available, ok := values["MemAvailable"]
	if !ok {
		available = free + values["Buffers"] + values["Cached"] + values["SReclaimable"]
		if shmem := values["Shmem"]; shmem < available {
			available -= shmem
		} else {
			available = 0
		}
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

func linuxSystemFromProc(uptimeOut []byte, loadOut []byte) (schema.SystemSection, error) {
	section := schema.SystemSection{
		SectionStatus: schema.SectionStatus{Status: schema.StatusAvailable, Scope: schema.ScopeHost},
	}

	var errs []string
	if uptime, err := linuxUptimeSeconds(uptimeOut); err == nil {
		section.UptimeSeconds = &uptime
	} else {
		errs = append(errs, "uptime: "+err.Error())
	}

	if load, err := linuxLoadAverage(loadOut); err == nil {
		section.LoadAverage = &load
	} else {
		errs = append(errs, "load average: "+err.Error())
	}

	if section.UptimeSeconds == nil && section.LoadAverage == nil {
		return schema.SystemSection{}, errors.New(strings.Join(errs, "; "))
	}
	return section, nil
}

func linuxUptimeSeconds(out []byte) (uint64, error) {
	fields := strings.Fields(string(out))
	if len(fields) == 0 {
		return 0, errors.New("uptime seconds missing from /proc/uptime")
	}
	seconds, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		return 0, fmt.Errorf("uptime seconds: %w", err)
	}
	if seconds < 0 {
		return 0, errors.New("uptime seconds is negative")
	}
	return uint64(seconds), nil
}

func linuxLoadAverage(out []byte) (schema.LoadAverage, error) {
	fields := strings.Fields(string(out))
	if len(fields) < 3 {
		return schema.LoadAverage{}, errors.New("load averages missing from /proc/loadavg")
	}

	values := [3]float64{}
	for i := range values {
		value, err := strconv.ParseFloat(fields[i], 64)
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

func linuxMountsFromProcMounts(out []byte) ([]linuxMount, error) {
	mounts := []linuxMount{}
	for _, line := range strings.Split(string(out), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}
		mountPoint, err := linuxUnescapeMountField(fields[1])
		if err != nil {
			return nil, fmt.Errorf("mount point %q: %w", fields[1], err)
		}
		fsType, err := linuxUnescapeMountField(fields[2])
		if err != nil {
			return nil, fmt.Errorf("filesystem type %q: %w", fields[2], err)
		}
		mounts = append(mounts, linuxMount{mountPoint: mountPoint, filesystemType: fsType})
	}
	if len(mounts) == 0 {
		return nil, errors.New("no mounts found in /proc/mounts")
	}
	return mounts, nil
}

func linuxUnescapeMountField(value string) (string, error) {
	var builder strings.Builder
	for i := 0; i < len(value); i++ {
		if value[i] != '\\' {
			builder.WriteByte(value[i])
			continue
		}
		if i+3 >= len(value) {
			return "", errors.New("incomplete escape sequence")
		}
		octal := value[i+1 : i+4]
		decoded, err := strconv.ParseUint(octal, 8, 8)
		if err != nil {
			return "", err
		}
		builder.WriteByte(byte(decoded))
		i += 3
	}
	return builder.String(), nil
}

func linuxVolumeFromStat(mountPoint string, fsType string, stat linuxMountStat) (schema.Volume, bool) {
	if linuxSkipVolume(mountPoint, fsType) {
		return schema.Volume{}, false
	}

	total := stat.blockSize * stat.blockCount
	free := stat.blockSize * stat.availableBlocks
	if free > total {
		free = total
	}
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

func linuxSkipVolume(mountPoint string, fsType string) bool {
	if mountPoint == "" || fsType == "" {
		return true
	}
	if strings.HasPrefix(mountPoint, "/proc") || strings.HasPrefix(mountPoint, "/sys") || strings.HasPrefix(mountPoint, "/dev") {
		return true
	}

	switch fsType {
	case "autofs", "bpf", "binfmt_misc", "cgroup", "cgroup2", "configfs", "debugfs",
		"devpts", "devtmpfs", "efivarfs", "fusectl", "hugetlbfs", "mqueue", "nsfs",
		"proc", "pstore", "ramfs", "rpc_pipefs", "securityfs", "sysfs", "tmpfs", "tracefs":
		return true
	default:
		return false
	}
}
