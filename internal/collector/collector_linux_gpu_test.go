package collector

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/haruto/fleetpulse/internal/schema"
)

func TestLinuxGPUFreshCacheUsesSavedDetectorWithoutRediscovery(t *testing.T) {
	cachePath := filepath.Join(t.TempDir(), "gpu-discovery.json")
	now := time.Date(2026, 7, 26, 12, 0, 0, 0, time.UTC)
	writeLinuxGPUCacheFixture(t, cachePath, linuxGPUDiscovery{
		Version:         linuxGPUCacheVersion,
		Detector:        "cached",
		Status:          schema.StatusAvailable,
		Vendor:          "NVIDIA",
		Model:           "RTX 4000",
		HostFingerprint: linuxGPUHostFingerprint{Hostname: "node-a", Platform: "linux", Architecture: runtime.GOARCH},
		DiscoveredAt:    now.Add(-time.Hour),
	})
	detector := &fakeLinuxGPUDetector{
		detectorName: "cached",
		collectSection: schema.GPUSection{
			SectionStatus: schema.SectionStatus{Status: schema.StatusAvailable, Scope: schema.ScopeHost},
			Devices:       []schema.GPUDevice{{Vendor: "NVIDIA", Model: "RTX 4000"}},
		},
	}
	collector := testLinuxGPUCollector(cachePath, now, detector)

	section := collector.collect(context.Background())

	if section.Status != schema.StatusAvailable {
		t.Fatalf("Status = %q, want %q", section.Status, schema.StatusAvailable)
	}
	if detector.discoverCalls != 0 {
		t.Fatalf("discover calls = %d, want 0", detector.discoverCalls)
	}
	if detector.collectCalls != 1 {
		t.Fatalf("collect calls = %d, want 1", detector.collectCalls)
	}
}

func TestLinuxGPUStaleCacheRunsRediscovery(t *testing.T) {
	cachePath := filepath.Join(t.TempDir(), "gpu-discovery.json")
	now := time.Date(2026, 7, 26, 12, 0, 0, 0, time.UTC)
	writeLinuxGPUCacheFixture(t, cachePath, linuxGPUDiscovery{
		Version:         linuxGPUCacheVersion,
		Detector:        "cached",
		Status:          schema.StatusAvailable,
		Vendor:          "Old",
		Model:           "Old GPU",
		HostFingerprint: linuxGPUHostFingerprint{Hostname: "node-a", Platform: "linux", Architecture: runtime.GOARCH},
		DiscoveredAt:    now.Add(-25 * time.Hour),
	})
	detector := &fakeLinuxGPUDetector{
		detectorName: "cached",
		discovery: linuxGPUDiscovery{
			Detector: "cached",
			Status:   schema.StatusAvailable,
			Vendor:   "NVIDIA",
			Model:    "RTX 4000",
		},
		collectSection: schema.GPUSection{
			SectionStatus: schema.SectionStatus{Status: schema.StatusAvailable, Scope: schema.ScopeHost},
			Devices:       []schema.GPUDevice{{Vendor: "NVIDIA", Model: "RTX 4000"}},
		},
	}
	collector := testLinuxGPUCollector(cachePath, now, detector)

	section := collector.collect(context.Background())

	if section.Status != schema.StatusAvailable {
		t.Fatalf("Status = %q, want %q", section.Status, schema.StatusAvailable)
	}
	if detector.discoverCalls != 1 {
		t.Fatalf("discover calls = %d, want 1", detector.discoverCalls)
	}
	got := readLinuxGPUCacheFixture(t, cachePath)
	if got.Model != "RTX 4000" {
		t.Fatalf("cached model = %q, want RTX 4000", got.Model)
	}
}

func TestLinuxGPUFingerprintMismatchRunsRediscovery(t *testing.T) {
	cachePath := filepath.Join(t.TempDir(), "gpu-discovery.json")
	now := time.Date(2026, 7, 26, 12, 0, 0, 0, time.UTC)
	writeLinuxGPUCacheFixture(t, cachePath, linuxGPUDiscovery{
		Version:         linuxGPUCacheVersion,
		Detector:        "cached",
		Status:          schema.StatusAvailable,
		Vendor:          "NVIDIA",
		Model:           "Old GPU",
		HostFingerprint: linuxGPUHostFingerprint{Hostname: "different-node", Platform: "linux", Architecture: runtime.GOARCH},
		DiscoveredAt:    now.Add(-time.Hour),
	})
	detector := &fakeLinuxGPUDetector{
		detectorName: "cached",
		discovery: linuxGPUDiscovery{
			Detector: "cached",
			Status:   schema.StatusAvailable,
			Vendor:   "NVIDIA",
			Model:    "Current GPU",
		},
		collectSection: schema.GPUSection{
			SectionStatus: schema.SectionStatus{Status: schema.StatusAvailable, Scope: schema.ScopeHost},
			Devices:       []schema.GPUDevice{{Vendor: "NVIDIA", Model: "Current GPU"}},
		},
	}
	collector := testLinuxGPUCollector(cachePath, now, detector)

	section := collector.collect(context.Background())

	if section.Status != schema.StatusAvailable {
		t.Fatalf("Status = %q, want %q", section.Status, schema.StatusAvailable)
	}
	if detector.discoverCalls != 1 {
		t.Fatalf("discover calls = %d, want 1", detector.discoverCalls)
	}
}

func TestLinuxGPUDeviceFingerprintMismatchRunsRediscovery(t *testing.T) {
	root := t.TempDir()
	devicePath := filepath.Join(root, "sys/class/drm/card0/device")
	writeFile(t, filepath.Join(devicePath, "vendor"), "0x1002\n")
	writeFile(t, filepath.Join(devicePath, "device"), "0x73bf\n")
	cachePath := filepath.Join(t.TempDir(), "gpu-discovery.json")
	now := time.Date(2026, 7, 26, 12, 0, 0, 0, time.UTC)
	writeLinuxGPUCacheFixture(t, cachePath, linuxGPUDiscovery{
		Version:  linuxGPUCacheVersion,
		Detector: "cached",
		Status:   schema.StatusAvailable,
		Vendor:   "AMD",
		Model:    "Old Radeon",
		Devices: []linuxGPUDevice{{
			Path:     devicePath,
			VendorID: "0x1002",
			DeviceID: "0x9999",
		}},
		HostFingerprint: linuxGPUHostFingerprint{
			Hostname:     "node-a",
			Platform:     "linux",
			Architecture: runtime.GOARCH,
			DeviceIDs:    []string{"0x1002/0x9999@" + devicePath},
		},
		DiscoveredAt: now.Add(-time.Hour),
	})
	detector := &fakeLinuxGPUDetector{
		detectorName: "cached",
		discovery: linuxGPUDiscovery{
			Detector: "cached",
			Status:   schema.StatusAvailable,
			Vendor:   "AMD",
			Model:    "Current Radeon",
		},
		collectSection: schema.GPUSection{
			SectionStatus: schema.SectionStatus{Status: schema.StatusAvailable, Scope: schema.ScopeHost},
			Devices:       []schema.GPUDevice{{Vendor: "AMD", Model: "Current Radeon"}},
		},
	}
	collector := testLinuxGPUCollector(cachePath, now, detector)

	section := collector.collect(context.Background())

	if section.Status != schema.StatusAvailable {
		t.Fatalf("Status = %q, want %q", section.Status, schema.StatusAvailable)
	}
	if detector.discoverCalls != 1 {
		t.Fatalf("discover calls = %d, want 1", detector.discoverCalls)
	}
}

func TestLinuxGPUCachedFailureRediscoveriesAtThreshold(t *testing.T) {
	cachePath := filepath.Join(t.TempDir(), "gpu-discovery.json")
	now := time.Date(2026, 7, 26, 12, 0, 0, 0, time.UTC)
	writeLinuxGPUCacheFixture(t, cachePath, linuxGPUDiscovery{
		Version:         linuxGPUCacheVersion,
		Detector:        "cached",
		Status:          schema.StatusAvailable,
		Vendor:          "NVIDIA",
		Model:           "Broken GPU",
		HostFingerprint: linuxGPUHostFingerprint{Hostname: "node-a", Platform: "linux", Architecture: runtime.GOARCH},
		DiscoveredAt:    now.Add(-time.Hour),
		FailureCount:    linuxGPUFailureRediscoveryThreshold - 1,
	})
	detector := &fakeLinuxGPUDetector{
		detectorName: "cached",
		collectErr:   errors.New("driver not ready"),
		discovery: linuxGPUDiscovery{
			Detector: "cached",
			Status:   schema.StatusAvailable,
			Vendor:   "NVIDIA",
			Model:    "Recovered GPU",
		},
		rediscoveredSection: schema.GPUSection{
			SectionStatus: schema.SectionStatus{Status: schema.StatusAvailable, Scope: schema.ScopeHost},
			Devices:       []schema.GPUDevice{{Vendor: "NVIDIA", Model: "Recovered GPU"}},
		},
	}
	collector := testLinuxGPUCollector(cachePath, now, detector)

	section := collector.collect(context.Background())

	if section.Status != schema.StatusAvailable {
		t.Fatalf("Status = %q, want %q", section.Status, schema.StatusAvailable)
	}
	if detector.discoverCalls != 1 {
		t.Fatalf("discover calls = %d, want 1", detector.discoverCalls)
	}
	got := readLinuxGPUCacheFixture(t, cachePath)
	if got.FailureCount != 0 {
		t.Fatalf("FailureCount = %d, want 0 after successful rediscovery", got.FailureCount)
	}
}

func TestLinuxGPUCachedFailureBelowThresholdReturnsUnavailable(t *testing.T) {
	cachePath := filepath.Join(t.TempDir(), "gpu-discovery.json")
	now := time.Date(2026, 7, 26, 12, 0, 0, 0, time.UTC)
	writeLinuxGPUCacheFixture(t, cachePath, linuxGPUDiscovery{
		Version:         linuxGPUCacheVersion,
		Detector:        "cached",
		Status:          schema.StatusAvailable,
		Vendor:          "NVIDIA",
		Model:           "RTX 4000",
		HostFingerprint: linuxGPUHostFingerprint{Hostname: "node-a", Platform: "linux", Architecture: runtime.GOARCH},
		DiscoveredAt:    now.Add(-time.Hour),
	})
	detector := &fakeLinuxGPUDetector{
		detectorName: "cached",
		collectErr:   errors.New("driver not ready"),
	}
	collector := testLinuxGPUCollector(cachePath, now, detector)

	section := collector.collect(context.Background())

	if section.Status != schema.StatusUnavailable {
		t.Fatalf("Status = %q, want %q", section.Status, schema.StatusUnavailable)
	}
	if detector.discoverCalls != 0 {
		t.Fatalf("discover calls = %d, want 0", detector.discoverCalls)
	}
	got := readLinuxGPUCacheFixture(t, cachePath)
	if got.FailureCount != 1 {
		t.Fatalf("FailureCount = %d, want 1", got.FailureCount)
	}
	if got.LastFailureAt == nil || !got.LastFailureAt.Equal(now) {
		t.Fatalf("LastFailureAt = %v, want %s", got.LastFailureAt, now)
	}
}

func TestLinuxGPUUnsupportedCacheExpiresSooner(t *testing.T) {
	cachePath := filepath.Join(t.TempDir(), "gpu-discovery.json")
	now := time.Date(2026, 7, 26, 12, 0, 0, 0, time.UTC)
	writeLinuxGPUCacheFixture(t, cachePath, linuxGPUDiscovery{
		Version:         linuxGPUCacheVersion,
		Detector:        "none",
		Status:          schema.StatusUnsupported,
		HostFingerprint: linuxGPUHostFingerprint{Hostname: "node-a", Platform: "linux", Architecture: runtime.GOARCH},
		DiscoveredAt:    now.Add(-61 * time.Minute),
	})
	detector := &fakeLinuxGPUDetector{
		detectorName: "new",
		discovery: linuxGPUDiscovery{
			Detector: "new",
			Status:   schema.StatusAvailable,
			Vendor:   "Intel",
			Model:    "Integrated Graphics",
		},
		collectSection: schema.GPUSection{
			SectionStatus: schema.SectionStatus{Status: schema.StatusAvailable, Scope: schema.ScopeHost},
			Devices:       []schema.GPUDevice{{Vendor: "Intel", Model: "Integrated Graphics"}},
		},
	}
	collector := testLinuxGPUCollector(cachePath, now, detector)

	section := collector.collect(context.Background())

	if section.Status != schema.StatusAvailable {
		t.Fatalf("Status = %q, want %q", section.Status, schema.StatusAvailable)
	}
	if detector.discoverCalls != 1 {
		t.Fatalf("discover calls = %d, want 1", detector.discoverCalls)
	}
}

func TestLinuxGPUInvalidCacheIsIgnored(t *testing.T) {
	cachePath := filepath.Join(t.TempDir(), "gpu-discovery.json")
	if err := os.WriteFile(cachePath, []byte("{"), 0o600); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 7, 26, 12, 0, 0, 0, time.UTC)
	detector := &fakeLinuxGPUDetector{
		detectorName: "cached",
		discovery: linuxGPUDiscovery{
			Detector: "cached",
			Status:   schema.StatusAvailable,
			Vendor:   "AMD",
			Model:    "Radeon",
		},
		collectSection: schema.GPUSection{
			SectionStatus: schema.SectionStatus{Status: schema.StatusAvailable, Scope: schema.ScopeHost},
			Devices:       []schema.GPUDevice{{Vendor: "AMD", Model: "Radeon"}},
		},
	}
	collector := testLinuxGPUCollector(cachePath, now, detector)

	section := collector.collect(context.Background())

	if section.Status != schema.StatusAvailable {
		t.Fatalf("Status = %q, want %q", section.Status, schema.StatusAvailable)
	}
	if detector.discoverCalls != 1 {
		t.Fatalf("discover calls = %d, want 1", detector.discoverCalls)
	}
}

func TestLinuxGPUCacheWriteFailureDoesNotFailLiveDiscovery(t *testing.T) {
	now := time.Date(2026, 7, 26, 12, 0, 0, 0, time.UTC)
	detector := &fakeLinuxGPUDetector{
		detectorName: "cached",
		discovery: linuxGPUDiscovery{
			Detector: "cached",
			Status:   schema.StatusAvailable,
			Vendor:   "AMD",
			Model:    "Radeon",
		},
		collectSection: schema.GPUSection{
			SectionStatus: schema.SectionStatus{Status: schema.StatusAvailable, Scope: schema.ScopeHost},
			Devices:       []schema.GPUDevice{{Vendor: "AMD", Model: "Radeon"}},
		},
	}
	collector := testLinuxGPUCollector(filepath.Join(t.TempDir(), "gpu-discovery.json"), now, detector)
	collector.writeFile = func(string, []byte, os.FileMode) error {
		return errors.New("read-only state")
	}

	section := collector.collect(context.Background())

	if section.Status != schema.StatusAvailable {
		t.Fatalf("Status = %q, want %q", section.Status, schema.StatusAvailable)
	}
	if detector.collectCalls != 1 {
		t.Fatalf("collect calls = %d, want 1", detector.collectCalls)
	}
}

func TestLinuxGPURaspberryPiDetectorDiscoversModel(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "proc/device-tree/model"), "Raspberry Pi 5 Model B Rev 1.0\x00")
	collector := testLinuxGPUCollector(filepath.Join(t.TempDir(), "cache.json"), time.Now(), raspberryPiLinuxGPUDetector{})
	collector.root = root

	discovery, ok := raspberryPiLinuxGPUDetector{}.discover(context.Background(), collector)

	if !ok {
		t.Fatal("raspberry pi detector did not match model file")
	}
	if discovery.Detector != "raspberry-pi" {
		t.Fatalf("Detector = %q, want raspberry-pi", discovery.Detector)
	}
	if discovery.Vendor != "Broadcom" {
		t.Fatalf("Vendor = %q, want Broadcom", discovery.Vendor)
	}
	if discovery.Model != "Raspberry Pi 5 Model B Rev 1.0 VideoCore GPU" {
		t.Fatalf("Model = %q, want Raspberry Pi 5 Model B Rev 1.0 VideoCore GPU", discovery.Model)
	}
}

func TestLinuxGPURaspberryPiDetectorDiscoversVideoCoreDRM(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "sys/class/drm/card0/device/vendor"), "0x14e4\n")
	writeFile(t, filepath.Join(root, "sys/class/drm/card0/device/device"), "0x2711\n")
	collector := testLinuxGPUCollector(filepath.Join(t.TempDir(), "cache.json"), time.Now(), raspberryPiLinuxGPUDetector{})
	collector.root = root

	discovery, ok := raspberryPiLinuxGPUDetector{}.discover(context.Background(), collector)

	if !ok {
		t.Fatal("raspberry pi detector did not match VideoCore DRM device")
	}
	if discovery.Vendor != "Broadcom" {
		t.Fatalf("Vendor = %q, want Broadcom", discovery.Vendor)
	}
	if !strings.Contains(discovery.Model, "VideoCore") {
		t.Fatalf("Model = %q, want VideoCore identity", discovery.Model)
	}
}

func TestLinuxGPUNVIDIAParserReturnsMetrics(t *testing.T) {
	out := []byte("NVIDIA RTX A4000, 16384, 2048, 17, 58\n")

	section, err := linuxGPUNVIDIASectionFromCSV(out)

	if err != nil {
		t.Fatalf("linuxGPUNVIDIASectionFromCSV returned error: %v", err)
	}
	if section.Status != schema.StatusAvailable {
		t.Fatalf("Status = %q, want %q", section.Status, schema.StatusAvailable)
	}
	if len(section.Devices) != 1 {
		t.Fatalf("device count = %d, want 1", len(section.Devices))
	}
	device := section.Devices[0]
	if device.Vendor != "NVIDIA" || device.Model != "NVIDIA RTX A4000" {
		t.Fatalf("device identity = %q/%q, want NVIDIA/NVIDIA RTX A4000", device.Vendor, device.Model)
	}
	if got := linuxValue(t, device.MemoryTotalBytes); got != 16_384*1024*1024 {
		t.Fatalf("MemoryTotalBytes = %d, want %d", got, uint64(16_384*1024*1024))
	}
	if got := linuxValue(t, device.MemoryUsedBytes); got != 2_048*1024*1024 {
		t.Fatalf("MemoryUsedBytes = %d, want %d", got, uint64(2_048*1024*1024))
	}
	if device.Utilization == nil || *device.Utilization != 17 {
		t.Fatalf("Utilization = %v, want 17", device.Utilization)
	}
	if device.TemperatureCelsius == nil || *device.TemperatureCelsius != 58 {
		t.Fatalf("TemperatureCelsius = %v, want 58", device.TemperatureCelsius)
	}
}

func TestLinuxGPUSysfsDetectorDiscoversAMDAndIntel(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "sys/class/drm/card0/device/vendor"), "0x1002\n")
	writeFile(t, filepath.Join(root, "sys/class/drm/card0/device/device"), "0x73bf\n")
	writeFile(t, filepath.Join(root, "sys/class/drm/card1/device/vendor"), "0x8086\n")
	writeFile(t, filepath.Join(root, "sys/class/drm/card1/device/device"), "0x46a6\n")
	collector := testLinuxGPUCollector(filepath.Join(t.TempDir(), "cache.json"), time.Now(), amdLinuxGPUDetector{}, intelLinuxGPUDetector{})
	collector.root = root

	amd, amdOK := amdLinuxGPUDetector{}.discover(context.Background(), collector)
	intel, intelOK := intelLinuxGPUDetector{}.discover(context.Background(), collector)

	if !amdOK {
		t.Fatal("amd detector did not match vendor id")
	}
	if amd.Vendor != "AMD" || !strings.Contains(amd.Model, "0x73bf") {
		t.Fatalf("amd identity = %q/%q, want AMD containing 0x73bf", amd.Vendor, amd.Model)
	}
	if !intelOK {
		t.Fatal("intel detector did not match vendor id")
	}
	if intel.Vendor != "Intel" || !strings.Contains(intel.Model, "0x46a6") {
		t.Fatalf("intel identity = %q/%q, want Intel containing 0x46a6", intel.Vendor, intel.Model)
	}
}

func TestLinuxGPUGenericDRMFallbackDiscoversIdentity(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "sys/class/drm/card0/device/vendor"), "0x1af4\n")
	writeFile(t, filepath.Join(root, "sys/class/drm/card0/device/device"), "0x1050\n")
	collector := testLinuxGPUCollector(filepath.Join(t.TempDir(), "cache.json"), time.Now(), drmLinuxGPUDetector{})
	collector.root = root

	discovery, ok := drmLinuxGPUDetector{}.discover(context.Background(), collector)

	if !ok {
		t.Fatal("drm detector did not match drm device")
	}
	if discovery.Detector != "drm" {
		t.Fatalf("Detector = %q, want drm", discovery.Detector)
	}
	if discovery.Vendor != "0x1af4" || !strings.Contains(discovery.Model, "0x1050") {
		t.Fatalf("identity = %q/%q, want 0x1af4 containing 0x1050", discovery.Vendor, discovery.Model)
	}
}

func TestLinuxGPUSysfsCollectionReturnsReadableMetrics(t *testing.T) {
	root := t.TempDir()
	devicePath := filepath.Join(root, "sys/class/drm/card0/device")
	writeFile(t, filepath.Join(devicePath, "vendor"), "0x1002\n")
	writeFile(t, filepath.Join(devicePath, "device"), "0x73bf\n")
	writeFile(t, filepath.Join(devicePath, "mem_info_vram_total"), "8589934592\n")
	writeFile(t, filepath.Join(devicePath, "mem_info_vram_used"), "2147483648\n")
	writeFile(t, filepath.Join(devicePath, "gpu_busy_percent"), "12\n")
	writeFile(t, filepath.Join(devicePath, "hwmon/hwmon0/temp1_input"), "54000\n")
	collector := testLinuxGPUCollector(filepath.Join(t.TempDir(), "cache.json"), time.Now(), amdLinuxGPUDetector{})
	collector.root = root
	discovery, ok := amdLinuxGPUDetector{}.discover(context.Background(), collector)
	if !ok {
		t.Fatal("amd detector did not match vendor id")
	}

	section, err := amdLinuxGPUDetector{}.collect(context.Background(), collector, discovery)

	if err != nil {
		t.Fatalf("amd collect returned error: %v", err)
	}
	device := section.Devices[0]
	if got := linuxValue(t, device.MemoryTotalBytes); got != 8_589_934_592 {
		t.Fatalf("MemoryTotalBytes = %d, want %d", got, uint64(8_589_934_592))
	}
	if got := linuxValue(t, device.MemoryUsedBytes); got != 2_147_483_648 {
		t.Fatalf("MemoryUsedBytes = %d, want %d", got, uint64(2_147_483_648))
	}
	if device.Utilization == nil || *device.Utilization != 12 {
		t.Fatalf("Utilization = %v, want 12", device.Utilization)
	}
	if device.TemperatureCelsius == nil || *device.TemperatureCelsius != 54 {
		t.Fatalf("TemperatureCelsius = %v, want 54", device.TemperatureCelsius)
	}
}

func TestLinuxGPUDetectorPriorityUsesSpecificDetectorBeforeDRM(t *testing.T) {
	cachePath := filepath.Join(t.TempDir(), "gpu-discovery.json")
	now := time.Date(2026, 7, 26, 12, 0, 0, 0, time.UTC)
	first := &fakeLinuxGPUDetector{
		detectorName: "raspberry-pi",
		discovery: linuxGPUDiscovery{
			Detector: "raspberry-pi",
			Status:   schema.StatusAvailable,
			Vendor:   "Broadcom",
			Model:    "VideoCore",
		},
		collectSection: schema.GPUSection{
			SectionStatus: schema.SectionStatus{Status: schema.StatusAvailable, Scope: schema.ScopeHost},
			Devices:       []schema.GPUDevice{{Vendor: "Broadcom", Model: "VideoCore"}},
		},
	}
	second := &fakeLinuxGPUDetector{
		detectorName: "drm",
		discovery:    linuxGPUDiscovery{Detector: "drm", Status: schema.StatusAvailable, Vendor: "generic", Model: "DRM GPU"},
	}
	collector := testLinuxGPUCollector(cachePath, now, first, second)

	section := collector.collect(context.Background())

	if section.Devices[0].Vendor != "Broadcom" {
		t.Fatalf("Vendor = %q, want Broadcom", section.Devices[0].Vendor)
	}
	if second.discoverCalls != 0 {
		t.Fatalf("fallback discover calls = %d, want 0", second.discoverCalls)
	}
}

type fakeLinuxGPUDetector struct {
	detectorName        string
	discovery           linuxGPUDiscovery
	collectSection      schema.GPUSection
	rediscoveredSection schema.GPUSection
	collectErr          error
	discoverCalls       int
	collectCalls        int
}

func (d *fakeLinuxGPUDetector) name() string {
	return d.detectorName
}

func (d *fakeLinuxGPUDetector) discover(context.Context, linuxGPUCollector) (linuxGPUDiscovery, bool) {
	d.discoverCalls++
	if d.discovery.Detector == "" {
		return linuxGPUDiscovery{}, false
	}
	return d.discovery, true
}

func (d *fakeLinuxGPUDetector) collect(context.Context, linuxGPUCollector, linuxGPUDiscovery) (schema.GPUSection, error) {
	d.collectCalls++
	if d.collectErr != nil {
		if d.discoverCalls > 0 && d.rediscoveredSection.Status != "" {
			return d.rediscoveredSection, nil
		}
		return schema.GPUSection{}, d.collectErr
	}
	return d.collectSection, nil
}

func testLinuxGPUCollector(cachePath string, now time.Time, detectors ...linuxGPUDetector) linuxGPUCollector {
	return linuxGPUCollector{
		cachePath: cachePath,
		root:      "/",
		now:       func() time.Time { return now },
		hostname:  func() string { return "node-a" },
		readFile:  os.ReadFile,
		writeFile: os.WriteFile,
		mkdirAll:  os.MkdirAll,
		lookPath: func(name string) (string, error) {
			return "", os.ErrNotExist
		},
		runCommand: func(context.Context, string, ...string) ([]byte, error) {
			return nil, errors.New("unexpected command")
		},
		detectors: detectors,
	}
}

func writeLinuxGPUCacheFixture(t *testing.T, path string, discovery linuxGPUDiscovery) {
	t.Helper()
	collector := testLinuxGPUCollector(path, discovery.DiscoveredAt)
	if err := collector.saveDiscovery(discovery); err != nil {
		t.Fatal(err)
	}
}

func readLinuxGPUCacheFixture(t *testing.T, path string) linuxGPUDiscovery {
	t.Helper()
	collector := testLinuxGPUCollector(path, time.Now())
	discovery, ok := collector.loadDiscovery()
	if !ok {
		t.Fatal("cache fixture could not be loaded")
	}
	return discovery
}

func writeFile(t *testing.T, path string, contents string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
}
