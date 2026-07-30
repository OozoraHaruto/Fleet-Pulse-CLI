//go:build darwin

package collector

import (
	"syscall"
	"testing"

	"github.com/haruto/fleetpulse/internal/schema"
)

func TestDarwinParseIOStatCPUUsesLastSample(t *testing.T) {
	out := `          disk0               cpu    load average
    KB/t  tps  MB/s     us sy id   1m   5m   15m
   31.32   14  0.42      5  3 92  3.4  4.5   6.7
   12.00    1  0.01     11  7 82  3.1  4.0   5.9
`

	section, err := darwinCPUFromIOStat([]byte(out), 8)

	if err != nil {
		t.Fatalf("darwinCPUFromIOStat returned error: %v", err)
	}
	if section.Status != schema.StatusAvailable {
		t.Fatalf("Status = %q, want %q", section.Status, schema.StatusAvailable)
	}
	if section.Scope != schema.ScopeHost {
		t.Fatalf("Scope = %q, want %q", section.Scope, schema.ScopeHost)
	}
	if section.CoreCount != 8 {
		t.Fatalf("CoreCount = %d, want 8", section.CoreCount)
	}
	if section.Utilization == nil || *section.Utilization != 18 {
		t.Fatalf("Utilization = %v, want 18", section.Utilization)
	}
}

func TestDarwinMemoryFromVMStatReturnsHostMemory(t *testing.T) {
	vmStat := `Mach Virtual Memory Statistics: (page size of 16384 bytes)
Pages free:                                     100.
Pages active:                                   200.
Pages inactive:                                 300.
Pages speculative:                               50.
Pages wired down:                               400.
Pages purgeable:                                 25.
Pages occupied by compressor:                    75.
`

	section, err := darwinMemoryFromVMStat([]byte(vmStat), 20_480_000)

	if err != nil {
		t.Fatalf("darwinMemoryFromVMStat returned error: %v", err)
	}
	if section.Status != schema.StatusAvailable {
		t.Fatalf("Status = %q, want %q", section.Status, schema.StatusAvailable)
	}
	if section.Scope != schema.ScopeHost {
		t.Fatalf("Scope = %q, want %q", section.Scope, schema.ScopeHost)
	}
	if got := value(t, section.TotalBytes); got != 20_480_000 {
		t.Fatalf("TotalBytes = %d, want %d", got, uint64(20_480_000))
	}
	if got := value(t, section.FreeBytes); got != 1_638_400 {
		t.Fatalf("FreeBytes = %d, want %d", got, uint64(1_638_400))
	}
	if got := value(t, section.AvailableBytes); got != 7_782_400 {
		t.Fatalf("AvailableBytes = %d, want %d", got, uint64(7_782_400))
	}
	if got := value(t, section.UsedBytes); got != 12_697_600 {
		t.Fatalf("UsedBytes = %d, want %d", got, uint64(12_697_600))
	}
	if section.PercentUsed == nil || *section.PercentUsed != 62 {
		t.Fatalf("PercentUsed = %v, want 62", section.PercentUsed)
	}
}

func TestDarwinSystemParsersReturnUptimeAndLoad(t *testing.T) {
	boot, err := darwinParseBootTime("{ sec = 1721980000, usec = 0 } Mon Jul 26 08:26:40 2026")
	if err != nil {
		t.Fatalf("darwinParseBootTime returned error: %v", err)
	}
	if boot != 1721980000 {
		t.Fatalf("boot = %d, want %d", boot, int64(1721980000))
	}

	load, err := darwinParseLoadAverage("16:43  12 users, load averages: 5.14 5.66 5.81")
	if err != nil {
		t.Fatalf("darwinParseLoadAverage returned error: %v", err)
	}
	if load.OneMinute != 5.14 || load.FiveMinutes != 5.66 || load.FifteenMinutes != 5.81 {
		t.Fatalf("load = %+v, want 5.14/5.66/5.81", load)
	}
}

func TestDarwinGPUFromHardwareIdentityReturnsAppleSiliconDevice(t *testing.T) {
	section := darwinGPUFromHardwareIdentity("Apple M2", "Mac14,2")

	if section.Status != schema.StatusAvailable {
		t.Fatalf("Status = %q, want %q", section.Status, schema.StatusAvailable)
	}
	if section.Scope != schema.ScopeHost {
		t.Fatalf("Scope = %q, want %q", section.Scope, schema.ScopeHost)
	}
	if len(section.Devices) != 1 {
		t.Fatalf("device count = %d, want 1", len(section.Devices))
	}
	if section.Devices[0].Vendor != "Apple" {
		t.Fatalf("Vendor = %q, want Apple", section.Devices[0].Vendor)
	}
	if section.Devices[0].Model != "Apple M2 GPU" {
		t.Fatalf("Model = %q, want Apple M2 GPU", section.Devices[0].Model)
	}
}

func TestDarwinGPUFromSystemProfilerReturnsDisplayDetails(t *testing.T) {
	out := []byte(`{
	  "SPDisplaysDataType": [
	    {
	      "_name": "Apple M2",
	      "sppci_vendor": "Apple",
	      "spdisplays_vram": "10 GB"
	    }
	  ]
	}`)

	section, err := darwinGPUFromSystemProfiler(out)

	if err != nil {
		t.Fatalf("darwinGPUFromSystemProfiler returned error: %v", err)
	}
	if section.Status != schema.StatusAvailable {
		t.Fatalf("Status = %q, want %q", section.Status, schema.StatusAvailable)
	}
	if len(section.Devices) != 1 {
		t.Fatalf("device count = %d, want 1", len(section.Devices))
	}
	device := section.Devices[0]
	if device.Vendor != "Apple" {
		t.Fatalf("Vendor = %q, want Apple", device.Vendor)
	}
	if device.Model != "Apple M2 GPU" {
		t.Fatalf("Model = %q, want Apple M2 GPU", device.Model)
	}
	if got := value(t, device.MemoryTotalBytes); got != 10*1024*1024*1024 {
		t.Fatalf("MemoryTotalBytes = %d, want %d", got, uint64(10*1024*1024*1024))
	}
}

func TestDarwinApplyUnifiedMemoryTotalFillsAppleSiliconGPU(t *testing.T) {
	device := schema.GPUDevice{Vendor: "Apple", Model: "Apple M2 GPU"}

	darwinApplyUnifiedMemoryTotal(&device, 16*1024*1024*1024)

	if got := value(t, device.MemoryTotalBytes); got != 16*1024*1024*1024 {
		t.Fatalf("MemoryTotalBytes = %d, want %d", got, uint64(16*1024*1024*1024))
	}
}

func TestDarwinUnifiedMemoryBytesFromVMStatUsesDerivedMemoryTotal(t *testing.T) {
	vmStat := []byte(`Mach Virtual Memory Statistics: (page size of 16384 bytes)
Pages free:                                     100.
Pages active:                                   200.
Pages inactive:                                 300.
Pages speculative:                               50.
Pages wired down:                               400.
Pages purgeable:                                 25.
Pages occupied by compressor:                    75.
`)

	total, ok := darwinUnifiedMemoryBytesFromVMStat(vmStat)

	if !ok {
		t.Fatal("darwinUnifiedMemoryBytesFromVMStat returned ok=false")
	}
	if total != 18_432_000 {
		t.Fatalf("total = %d, want %d", total, uint64(18_432_000))
	}
}

func TestDarwinApplyIORegistryGPUFillsMemoryUsedAndUtilization(t *testing.T) {
	device := schema.GPUDevice{Vendor: "Apple", Model: "Apple M2 GPU"}
	out := []byte(`+-o AGXAcceleratorG14G  <class AGXAcceleratorG14G, id 0x1000003e6, registered, matched, active, busy 0 (1522 ms), retain 62>
  | {
  |   "PerformanceStatistics" = {"In use system memory (driver)"=0,"Alloc system memory"=3455598592,"Tiler Utilization %"=0,"recoveryCount"=0,"lastRecoveryTime"=0,"Renderer Utilization %"=0,"TiledSceneBytes"=557056,"Device Utilization %"=4,"SplitSceneCount"=0,"Allocated PB Size"=73400320,"In use system memory"=632307712}
  |   "model" = "Apple M2"
  | }
`)

	darwinApplyIORegistryGPU(&device, out)

	if got := value(t, device.MemoryUsedBytes); got != 632307712 {
		t.Fatalf("MemoryUsedBytes = %d, want %d", got, uint64(632307712))
	}
	if device.Utilization == nil || *device.Utilization != 4 {
		t.Fatalf("Utilization = %v, want 4", device.Utilization)
	}
}

func TestDarwinApplyPowermetricsGPUFillsUtilizationAndTemperature(t *testing.T) {
	device := schema.GPUDevice{Vendor: "Apple", Model: "Apple M2 GPU"}
	out := []byte(`GPU HW active residency: 12.34%
GPU die temperature: 51.2 C
`)

	darwinApplyPowermetricsGPU(&device, out)

	if device.Utilization == nil || *device.Utilization != 12.34 {
		t.Fatalf("Utilization = %v, want 12.34", device.Utilization)
	}
	if device.TemperatureCelsius == nil || *device.TemperatureCelsius != 51.2 {
		t.Fatalf("TemperatureCelsius = %v, want 51.2", device.TemperatureCelsius)
	}
}

func TestDarwinGPUFromHardwareIdentityAvoidsUnknownDetails(t *testing.T) {
	section := darwinGPUFromHardwareIdentity("Intel(R) Core(TM) i7", "MacBookPro15,1")

	if section.Status != schema.StatusUnsupported {
		t.Fatalf("Status = %q, want %q", section.Status, schema.StatusUnsupported)
	}
	if len(section.Devices) != 0 {
		t.Fatalf("device count = %d, want 0", len(section.Devices))
	}
}

func TestDarwinParseDiskutilInfoReturnsHealth(t *testing.T) {
	out := []byte(`   Device Identifier:         disk3s1
   SMART Status:              Verified
   Solid State:               Yes
`)

	health, ok := darwinDiskHealthFromDiskutil(out)

	if !ok {
		t.Fatal("darwinDiskHealthFromDiskutil returned ok=false")
	}
	if health.Status != "available" {
		t.Fatalf("Status = %q, want available", health.Status)
	}
	if len(health.Warnings) != 0 {
		t.Fatalf("Warnings = %v, want none", health.Warnings)
	}
}

func TestDarwinVolumeFromStatfsReturnsCapacity(t *testing.T) {
	var stat syscall.Statfs_t
	stat.Bsize = 4096
	stat.Blocks = 100
	stat.Bavail = 25
	copyInt8(stat.Mntonname[:], "/")
	copyInt8(stat.Fstypename[:], "apfs")
	copyInt8(stat.Mntfromname[:], "/dev/disk3s1")

	volume, ok := darwinVolumeFromStatfs(stat)

	if !ok {
		t.Fatal("darwinVolumeFromStatfs skipped usable APFS volume")
	}
	if volume.MountPoint != "/" {
		t.Fatalf("MountPoint = %q, want /", volume.MountPoint)
	}
	if got := value(t, volume.TotalBytes); got != 409600 {
		t.Fatalf("TotalBytes = %d, want %d", got, uint64(409600))
	}
	if got := value(t, volume.FreeBytes); got != 102400 {
		t.Fatalf("FreeBytes = %d, want %d", got, uint64(102400))
	}
	if got := value(t, volume.UsedBytes); got != 307200 {
		t.Fatalf("UsedBytes = %d, want %d", got, uint64(307200))
	}
	if volume.PercentUsed == nil || *volume.PercentUsed != 75 {
		t.Fatalf("PercentUsed = %v, want 75", volume.PercentUsed)
	}
}

func TestDarwinVolumeFromStatfsSkipsPseudoMounts(t *testing.T) {
	var stat syscall.Statfs_t
	stat.Bsize = 512
	stat.Blocks = 1
	stat.Bavail = 0
	copyInt8(stat.Mntonname[:], "/dev")
	copyInt8(stat.Fstypename[:], "devfs")
	copyInt8(stat.Mntfromname[:], "devfs")

	if _, ok := darwinVolumeFromStatfs(stat); ok {
		t.Fatal("darwinVolumeFromStatfs returned devfs pseudo mount")
	}
}

func value(t *testing.T, ptr *uint64) uint64 {
	t.Helper()
	if ptr == nil {
		t.Fatal("value pointer is nil")
	}
	return *ptr
}

func copyInt8(dst []int8, value string) {
	for i, char := range []byte(value) {
		dst[i] = int8(char)
	}
}
