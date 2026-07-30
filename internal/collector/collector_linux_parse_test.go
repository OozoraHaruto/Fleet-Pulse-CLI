package collector

import (
	"testing"

	"github.com/haruto/fleetpulse/internal/schema"
)

func TestLinuxMemoryFromMemInfoReturnsHostMemory(t *testing.T) {
	memInfo := []byte(`MemTotal:        4096000 kB
MemFree:          512000 kB
MemAvailable:    3072000 kB
Buffers:          128000 kB
Cached:           640000 kB
SwapCached:            0 kB
`)

	section, err := linuxMemoryFromMemInfo(memInfo)

	if err != nil {
		t.Fatalf("linuxMemoryFromMemInfo returned error: %v", err)
	}
	if section.Status != schema.StatusAvailable {
		t.Fatalf("Status = %q, want %q", section.Status, schema.StatusAvailable)
	}
	if section.Scope != schema.ScopeHost {
		t.Fatalf("Scope = %q, want %q", section.Scope, schema.ScopeHost)
	}
	if got := linuxValue(t, section.TotalBytes); got != 4_194_304_000 {
		t.Fatalf("TotalBytes = %d, want %d", got, uint64(4_194_304_000))
	}
	if got := linuxValue(t, section.FreeBytes); got != 524_288_000 {
		t.Fatalf("FreeBytes = %d, want %d", got, uint64(524_288_000))
	}
	if got := linuxValue(t, section.AvailableBytes); got != 3_145_728_000 {
		t.Fatalf("AvailableBytes = %d, want %d", got, uint64(3_145_728_000))
	}
	if got := linuxValue(t, section.UsedBytes); got != 1_048_576_000 {
		t.Fatalf("UsedBytes = %d, want %d", got, uint64(1_048_576_000))
	}
	if section.PercentUsed == nil || *section.PercentUsed != 25 {
		t.Fatalf("PercentUsed = %v, want 25", section.PercentUsed)
	}
}

func TestLinuxMemoryFromMemInfoDerivesAvailableWhenMissing(t *testing.T) {
	memInfo := []byte(`MemTotal:        1024000 kB
MemFree:          128000 kB
Buffers:           32000 kB
Cached:           256000 kB
SReclaimable:      64000 kB
Shmem:             16000 kB
`)

	section, err := linuxMemoryFromMemInfo(memInfo)

	if err != nil {
		t.Fatalf("linuxMemoryFromMemInfo returned error: %v", err)
	}
	if got := linuxValue(t, section.AvailableBytes); got != 475_136_000 {
		t.Fatalf("AvailableBytes = %d, want %d", got, uint64(475_136_000))
	}
	if got := linuxValue(t, section.UsedBytes); got != 573_440_000 {
		t.Fatalf("UsedBytes = %d, want %d", got, uint64(573_440_000))
	}
	if section.PercentUsed == nil || *section.PercentUsed != 54.6875 {
		t.Fatalf("PercentUsed = %v, want 54.6875", section.PercentUsed)
	}
}

func TestLinuxCPUFromProcStatReturnsUtilization(t *testing.T) {
	first := []byte(`cpu  100 0 50 850 0 0 0 0 0 0
cpu0 60 0 20 420 0 0 0 0 0 0
cpu1 40 0 30 430 0 0 0 0 0 0
`)
	second := []byte(`cpu  150 0 80 970 0 0 0 0 0 0
cpu0 90 0 30 480 0 0 0 0 0 0
cpu1 60 0 50 490 0 0 0 0 0 0
`)

	section, err := linuxCPUFromProcStat(first, second, 2)

	if err != nil {
		t.Fatalf("linuxCPUFromProcStat returned error: %v", err)
	}
	if section.Status != schema.StatusAvailable {
		t.Fatalf("Status = %q, want %q", section.Status, schema.StatusAvailable)
	}
	if section.Scope != schema.ScopeHost {
		t.Fatalf("Scope = %q, want %q", section.Scope, schema.ScopeHost)
	}
	if section.CoreCount != 2 {
		t.Fatalf("CoreCount = %d, want 2", section.CoreCount)
	}
	if section.Utilization == nil || *section.Utilization != 40 {
		t.Fatalf("Utilization = %v, want 40", section.Utilization)
	}
	if len(section.PerCoreUtilization) != 2 {
		t.Fatalf("PerCoreUtilization length = %d, want 2", len(section.PerCoreUtilization))
	}
	if section.PerCoreUtilization[0] != 40 {
		t.Fatalf("PerCoreUtilization[0] = %v, want 40", section.PerCoreUtilization[0])
	}
	if section.PerCoreUtilization[1] != 40 {
		t.Fatalf("PerCoreUtilization[1] = %v, want 40", section.PerCoreUtilization[1])
	}
}

func TestLinuxCPUFromProcStatRejectsMissingAggregateLine(t *testing.T) {
	first := []byte("intr 1 2 3\n")
	second := []byte("intr 2 3 4\n")

	_, err := linuxCPUFromProcStat(first, second, 4)

	if err == nil {
		t.Fatal("linuxCPUFromProcStat returned nil error, want missing aggregate error")
	}
}

func TestLinuxSystemFromProcReturnsUptimeAndLoad(t *testing.T) {
	section, err := linuxSystemFromProc([]byte("12345.67 8910.11\n"), []byte("0.42 1.25 2.50 1/234 5678\n"))

	if err != nil {
		t.Fatalf("linuxSystemFromProc returned error: %v", err)
	}
	if section.Status != schema.StatusAvailable {
		t.Fatalf("Status = %q, want %q", section.Status, schema.StatusAvailable)
	}
	if got := linuxValue(t, section.UptimeSeconds); got != 12345 {
		t.Fatalf("UptimeSeconds = %d, want %d", got, uint64(12345))
	}
	if section.LoadAverage == nil {
		t.Fatal("LoadAverage is nil")
	}
	if section.LoadAverage.OneMinute != 0.42 || section.LoadAverage.FiveMinutes != 1.25 || section.LoadAverage.FifteenMinutes != 2.50 {
		t.Fatalf("LoadAverage = %+v, want 0.42/1.25/2.50", section.LoadAverage)
	}
}

func TestLinuxVolumeFromStatReturnsCapacity(t *testing.T) {
	volume, ok := linuxVolumeFromStat("/", "ext4", linuxMountStat{
		blockSize:       4096,
		blockCount:      100,
		availableBlocks: 25,
	})

	if !ok {
		t.Fatal("linuxVolumeFromStat skipped usable ext4 volume")
	}
	if volume.MountPoint != "/" {
		t.Fatalf("MountPoint = %q, want /", volume.MountPoint)
	}
	if volume.FilesystemType != "ext4" {
		t.Fatalf("FilesystemType = %q, want ext4", volume.FilesystemType)
	}
	if got := linuxValue(t, volume.TotalBytes); got != 409600 {
		t.Fatalf("TotalBytes = %d, want %d", got, uint64(409600))
	}
	if got := linuxValue(t, volume.FreeBytes); got != 102400 {
		t.Fatalf("FreeBytes = %d, want %d", got, uint64(102400))
	}
	if got := linuxValue(t, volume.UsedBytes); got != 307200 {
		t.Fatalf("UsedBytes = %d, want %d", got, uint64(307200))
	}
	if volume.PercentUsed == nil || *volume.PercentUsed != 75 {
		t.Fatalf("PercentUsed = %v, want 75", volume.PercentUsed)
	}
	if volume.Health == nil || volume.Health.Status != "unsupported" {
		t.Fatalf("Health = %+v, want unsupported", volume.Health)
	}
}

func TestLinuxVolumeFromStatSkipsPseudoMounts(t *testing.T) {
	if _, ok := linuxVolumeFromStat("/proc", "proc", linuxMountStat{
		blockSize:       4096,
		blockCount:      1,
		availableBlocks: 0,
	}); ok {
		t.Fatal("linuxVolumeFromStat returned proc pseudo mount")
	}
}

func linuxValue(t *testing.T, ptr *uint64) uint64 {
	t.Helper()
	if ptr == nil {
		t.Fatal("value pointer is nil")
	}
	return *ptr
}
