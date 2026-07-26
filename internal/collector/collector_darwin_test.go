//go:build darwin

package collector

import (
	"syscall"
	"testing"

	"github.com/haruto/fleetpulse/internal/schema"
)

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

func TestDarwinGPUFromHardwareIdentityAvoidsUnknownDetails(t *testing.T) {
	section := darwinGPUFromHardwareIdentity("Intel(R) Core(TM) i7", "MacBookPro15,1")

	if section.Status != schema.StatusUnsupported {
		t.Fatalf("Status = %q, want %q", section.Status, schema.StatusUnsupported)
	}
	if len(section.Devices) != 0 {
		t.Fatalf("device count = %d, want 0", len(section.Devices))
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
