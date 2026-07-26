package collector

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/haruto/fleetpulse/internal/schema"
)

const (
	linuxGPUCachePath                    = "/var/lib/fleetpulse/gpu-discovery.json"
	linuxGPUCacheVersion                 = 1
	linuxGPUAvailableDiscoveryTTL        = 24 * time.Hour
	linuxGPUUnsupportedDiscoveryTTL      = time.Hour
	linuxGPUFailureRediscoveryThreshold  = 2
	linuxGPUUnsupportedDiscoveryDetector = "none"
)

type linuxGPUCollector struct {
	cachePath  string
	root       string
	now        func() time.Time
	hostname   func() string
	readFile   func(string) ([]byte, error)
	writeFile  func(string, []byte, os.FileMode) error
	mkdirAll   func(string, os.FileMode) error
	lookPath   func(string) (string, error)
	runCommand func(context.Context, string, ...string) ([]byte, error)
	detectors  []linuxGPUDetector
}

type linuxGPUDetector interface {
	name() string
	discover(context.Context, linuxGPUCollector) (linuxGPUDiscovery, bool)
	collect(context.Context, linuxGPUCollector, linuxGPUDiscovery) (schema.GPUSection, error)
}

type linuxGPUDiscovery struct {
	Version         int                     `json:"version"`
	Detector        string                  `json:"detector"`
	Status          schema.Status           `json:"status"`
	Vendor          string                  `json:"vendor,omitempty"`
	Model           string                  `json:"model,omitempty"`
	Devices         []linuxGPUDevice        `json:"devices,omitempty"`
	HostFingerprint linuxGPUHostFingerprint `json:"host_fingerprint"`
	DiscoveredAt    time.Time               `json:"discovered_at"`
	LastFailureAt   *time.Time              `json:"last_failure_at,omitempty"`
	FailureCount    int                     `json:"failure_count"`
}

type linuxGPUDevice struct {
	Path     string `json:"path,omitempty"`
	Command  string `json:"command,omitempty"`
	VendorID string `json:"vendor_id,omitempty"`
	DeviceID string `json:"device_id,omitempty"`
	Vendor   string `json:"vendor,omitempty"`
	Model    string `json:"model,omitempty"`
}

type linuxGPUHostFingerprint struct {
	Hostname     string   `json:"hostname"`
	Platform     string   `json:"platform"`
	Architecture string   `json:"architecture"`
	DeviceIDs    []string `json:"device_ids,omitempty"`
}

type raspberryPiLinuxGPUDetector struct{}
type nvidiaLinuxGPUDetector struct{}
type amdLinuxGPUDetector struct{}
type intelLinuxGPUDetector struct{}
type drmLinuxGPUDetector struct{}

func newLinuxGPUCollector() linuxGPUCollector {
	return linuxGPUCollector{
		cachePath:  linuxGPUCachePath,
		root:       "/",
		now:        func() time.Time { return time.Now().UTC() },
		hostname:   linuxGPUHostname,
		readFile:   os.ReadFile,
		writeFile:  os.WriteFile,
		mkdirAll:   os.MkdirAll,
		lookPath:   exec.LookPath,
		runCommand: linuxGPURunCommand,
		detectors: []linuxGPUDetector{
			raspberryPiLinuxGPUDetector{},
			nvidiaLinuxGPUDetector{},
			amdLinuxGPUDetector{},
			intelLinuxGPUDetector{},
			drmLinuxGPUDetector{},
		},
	}
}

func linuxGPUHostname() string {
	hostname, err := os.Hostname()
	if err != nil || hostname == "" {
		return "unknown"
	}
	return hostname
}

func linuxGPURunCommand(ctx context.Context, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	return cmd.Output()
}

func (c linuxGPUCollector) collect(ctx context.Context) schema.GPUSection {
	c = c.withDefaults()
	if discovery, ok := c.loadDiscovery(); ok && c.discoveryFresh(discovery) && c.fingerprintMatches(discovery) {
		return c.collectCached(ctx, discovery)
	}
	return c.rediscover(ctx)
}

func (c linuxGPUCollector) withDefaults() linuxGPUCollector {
	if c.cachePath == "" {
		c.cachePath = linuxGPUCachePath
	}
	if c.root == "" {
		c.root = "/"
	}
	if c.now == nil {
		c.now = func() time.Time { return time.Now().UTC() }
	}
	if c.hostname == nil {
		c.hostname = linuxGPUHostname
	}
	if c.readFile == nil {
		c.readFile = os.ReadFile
	}
	if c.writeFile == nil {
		c.writeFile = os.WriteFile
	}
	if c.mkdirAll == nil {
		c.mkdirAll = os.MkdirAll
	}
	if c.lookPath == nil {
		c.lookPath = exec.LookPath
	}
	if c.runCommand == nil {
		c.runCommand = linuxGPURunCommand
	}
	if c.detectors == nil {
		c.detectors = newLinuxGPUCollector().detectors
	}
	return c
}

func (c linuxGPUCollector) collectCached(ctx context.Context, discovery linuxGPUDiscovery) schema.GPUSection {
	if discovery.Status == schema.StatusUnsupported {
		return schema.GPUSection{
			SectionStatus: unsupported("gpu collection requires a vendor-specific collector"),
		}
	}

	detector := c.detectorByName(discovery.Detector)
	if detector == nil {
		return c.rediscover(ctx)
	}
	section, err := detector.collect(ctx, c, discovery)
	if err == nil {
		if discovery.FailureCount != 0 || discovery.LastFailureAt != nil {
			discovery.FailureCount = 0
			discovery.LastFailureAt = nil
			_ = c.saveDiscovery(discovery)
		}
		return section
	}

	discovery.FailureCount++
	now := c.now()
	discovery.LastFailureAt = &now
	_ = c.saveDiscovery(discovery)
	if discovery.FailureCount >= linuxGPUFailureRediscoveryThreshold {
		return c.rediscover(ctx)
	}
	return schema.GPUSection{
		SectionStatus: unavailable("gpu collection failed: " + err.Error()),
	}
}

func (c linuxGPUCollector) rediscover(ctx context.Context) schema.GPUSection {
	for _, detector := range c.detectors {
		discovery, ok := detector.discover(ctx, c)
		if !ok {
			continue
		}
		discovery = c.prepareDiscovery(discovery)
		_ = c.saveDiscovery(discovery)

		section, err := detector.collect(ctx, c, discovery)
		if err != nil {
			return schema.GPUSection{
				SectionStatus: unavailable("gpu collection failed: " + err.Error()),
			}
		}
		return section
	}

	discovery := c.prepareDiscovery(linuxGPUDiscovery{
		Detector: linuxGPUUnsupportedDiscoveryDetector,
		Status:   schema.StatusUnsupported,
	})
	_ = c.saveDiscovery(discovery)
	return schema.GPUSection{
		SectionStatus: unsupported("gpu collection requires a vendor-specific collector"),
	}
}

func (c linuxGPUCollector) detectorByName(name string) linuxGPUDetector {
	for _, detector := range c.detectors {
		if detector.name() == name {
			return detector
		}
	}
	return nil
}

func (c linuxGPUCollector) loadDiscovery() (linuxGPUDiscovery, bool) {
	out, err := c.readFile(c.cachePath)
	if err != nil {
		return linuxGPUDiscovery{}, false
	}
	var discovery linuxGPUDiscovery
	if err := json.Unmarshal(out, &discovery); err != nil {
		return linuxGPUDiscovery{}, false
	}
	if discovery.Version != linuxGPUCacheVersion || discovery.Status == "" {
		return linuxGPUDiscovery{}, false
	}
	return discovery, true
}

func (c linuxGPUCollector) saveDiscovery(discovery linuxGPUDiscovery) error {
	discovery = c.prepareDiscovery(discovery)
	out, err := json.MarshalIndent(discovery, "", "  ")
	if err != nil {
		return err
	}
	if err := c.mkdirAll(filepath.Dir(c.cachePath), 0o700); err != nil {
		return err
	}
	return c.writeFile(c.cachePath, append(out, '\n'), 0o600)
}

func (c linuxGPUCollector) prepareDiscovery(discovery linuxGPUDiscovery) linuxGPUDiscovery {
	discovery.Version = linuxGPUCacheVersion
	if discovery.DiscoveredAt.IsZero() {
		discovery.DiscoveredAt = c.now()
	}
	if discovery.Status == "" {
		discovery.Status = schema.StatusAvailable
	}
	if discovery.HostFingerprint.Hostname == "" {
		discovery.HostFingerprint = c.hostFingerprint(discovery.Devices)
	}
	return discovery
}

func (c linuxGPUCollector) discoveryFresh(discovery linuxGPUDiscovery) bool {
	ttl := linuxGPUAvailableDiscoveryTTL
	if discovery.Status == schema.StatusUnsupported {
		ttl = linuxGPUUnsupportedDiscoveryTTL
	}
	return c.now().Sub(discovery.DiscoveredAt) < ttl
}

func (c linuxGPUCollector) fingerprintMatches(discovery linuxGPUDiscovery) bool {
	if discovery.HostFingerprint.Hostname != c.hostname() ||
		discovery.HostFingerprint.Platform != "linux" ||
		discovery.HostFingerprint.Architecture != runtime.GOARCH {
		return false
	}
	if len(discovery.HostFingerprint.DeviceIDs) == 0 {
		return true
	}
	return stringSlicesEqual(discovery.HostFingerprint.DeviceIDs, c.hostFingerprint(discovery.Devices).DeviceIDs)
}

func (c linuxGPUCollector) hostFingerprint(devices []linuxGPUDevice) linuxGPUHostFingerprint {
	deviceIDs := make([]string, 0, len(devices))
	for _, device := range devices {
		if device.Path == "" {
			continue
		}
		vendorID := strings.ToLower(device.VendorID)
		if current := c.readTrimmed(filepath.Join(device.Path, "vendor")); current != "" {
			vendorID = strings.ToLower(current)
		}
		deviceID := strings.ToLower(device.DeviceID)
		if current := c.readTrimmed(filepath.Join(device.Path, "device")); current != "" {
			deviceID = strings.ToLower(current)
		}
		if vendorID == "" && deviceID == "" {
			continue
		}
		deviceIDs = append(deviceIDs, vendorID+"/"+deviceID+"@"+device.Path)
	}
	return linuxGPUHostFingerprint{
		Hostname:     c.hostname(),
		Platform:     "linux",
		Architecture: runtime.GOARCH,
		DeviceIDs:    deviceIDs,
	}
}

func stringSlicesEqual(a []string, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func (raspberryPiLinuxGPUDetector) name() string {
	return "raspberry-pi"
}

func (raspberryPiLinuxGPUDetector) discover(_ context.Context, c linuxGPUCollector) (linuxGPUDiscovery, bool) {
	for _, path := range []string{
		c.rootPath("proc/device-tree/model"),
		c.rootPath("sys/firmware/devicetree/base/model"),
	} {
		out, err := c.readFile(path)
		if err != nil {
			continue
		}
		model := strings.Trim(strings.TrimSpace(string(out)), "\x00")
		if !strings.Contains(strings.ToLower(model), "raspberry pi") {
			continue
		}
		return linuxGPUDiscovery{
			Detector: raspberryPiLinuxGPUDetector{}.name(),
			Status:   schema.StatusAvailable,
			Vendor:   "Broadcom",
			Model:    model + " VideoCore GPU",
			Devices:  []linuxGPUDevice{{Path: path, Vendor: "Broadcom", Model: model + " VideoCore GPU"}},
		}, true
	}
	for _, device := range c.drmDevices() {
		if !strings.EqualFold(device.VendorID, "0x14e4") && !strings.Contains(strings.ToLower(device.Model), "videocore") {
			continue
		}
		device.Vendor = "Broadcom"
		device.Model = "Broadcom VideoCore GPU"
		if device.DeviceID != "" {
			device.Model += " " + device.DeviceID
		}
		return linuxGPUDiscovery{
			Detector: raspberryPiLinuxGPUDetector{}.name(),
			Status:   schema.StatusAvailable,
			Vendor:   "Broadcom",
			Model:    device.Model,
			Devices:  []linuxGPUDevice{device},
		}, true
	}
	return linuxGPUDiscovery{}, false
}

func (raspberryPiLinuxGPUDetector) collect(_ context.Context, _ linuxGPUCollector, discovery linuxGPUDiscovery) (schema.GPUSection, error) {
	return linuxGPUSectionFromDiscovery(discovery), nil
}

func (nvidiaLinuxGPUDetector) name() string {
	return "nvidia"
}

func (nvidiaLinuxGPUDetector) discover(_ context.Context, c linuxGPUCollector) (linuxGPUDiscovery, bool) {
	path, err := c.lookPath("nvidia-smi")
	if err != nil || path == "" {
		return linuxGPUDiscovery{}, false
	}
	return linuxGPUDiscovery{
		Detector: nvidiaLinuxGPUDetector{}.name(),
		Status:   schema.StatusAvailable,
		Vendor:   "NVIDIA",
		Model:    "NVIDIA GPU",
		Devices:  []linuxGPUDevice{{Command: path, Vendor: "NVIDIA", Model: "NVIDIA GPU"}},
	}, true
}

func (nvidiaLinuxGPUDetector) collect(ctx context.Context, c linuxGPUCollector, discovery linuxGPUDiscovery) (schema.GPUSection, error) {
	command := "nvidia-smi"
	if len(discovery.Devices) > 0 && discovery.Devices[0].Command != "" {
		command = discovery.Devices[0].Command
	}
	out, err := c.runCommand(ctx, command,
		"--query-gpu=name,memory.total,memory.used,utilization.gpu,temperature.gpu",
		"--format=csv,noheader,nounits")
	if err != nil {
		return schema.GPUSection{}, err
	}
	return linuxGPUNVIDIASectionFromCSV(out)
}

func (amdLinuxGPUDetector) name() string {
	return "amd"
}

func (amdLinuxGPUDetector) discover(_ context.Context, c linuxGPUCollector) (linuxGPUDiscovery, bool) {
	return c.discoverSysfsGPU(amdLinuxGPUDetector{}.name(), "0x1002", "AMD", "AMD GPU")
}

func (amdLinuxGPUDetector) collect(_ context.Context, c linuxGPUCollector, discovery linuxGPUDiscovery) (schema.GPUSection, error) {
	return c.sysfsGPUSection(discovery), nil
}

func (intelLinuxGPUDetector) name() string {
	return "intel"
}

func (intelLinuxGPUDetector) discover(_ context.Context, c linuxGPUCollector) (linuxGPUDiscovery, bool) {
	return c.discoverSysfsGPU(intelLinuxGPUDetector{}.name(), "0x8086", "Intel", "Intel GPU")
}

func (intelLinuxGPUDetector) collect(_ context.Context, c linuxGPUCollector, discovery linuxGPUDiscovery) (schema.GPUSection, error) {
	return c.sysfsGPUSection(discovery), nil
}

func (drmLinuxGPUDetector) name() string {
	return "drm"
}

func (drmLinuxGPUDetector) discover(_ context.Context, c linuxGPUCollector) (linuxGPUDiscovery, bool) {
	devices := c.drmDevices()
	if len(devices) == 0 {
		return linuxGPUDiscovery{}, false
	}
	device := devices[0]
	vendor := device.VendorID
	if device.Vendor != "" {
		vendor = device.Vendor
	}
	model := "DRM GPU"
	if device.DeviceID != "" {
		model += " " + device.DeviceID
	}
	device.Vendor = vendor
	device.Model = model
	return linuxGPUDiscovery{
		Detector: drmLinuxGPUDetector{}.name(),
		Status:   schema.StatusAvailable,
		Vendor:   vendor,
		Model:    model,
		Devices:  []linuxGPUDevice{device},
	}, true
}

func (drmLinuxGPUDetector) collect(_ context.Context, c linuxGPUCollector, discovery linuxGPUDiscovery) (schema.GPUSection, error) {
	return c.sysfsGPUSection(discovery), nil
}

func linuxGPUNVIDIASectionFromCSV(out []byte) (schema.GPUSection, error) {
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	devices := make([]schema.GPUDevice, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		fields := strings.Split(line, ",")
		if len(fields) < 5 {
			return schema.GPUSection{}, errors.New("nvidia-smi output missing gpu fields")
		}
		for i := range fields {
			fields[i] = strings.TrimSpace(fields[i])
		}
		device := schema.GPUDevice{
			Vendor: "NVIDIA",
			Model:  fields[0],
		}
		if value, ok := linuxGPUMegabytesToBytes(fields[1]); ok {
			device.MemoryTotalBytes = &value
		}
		if value, ok := linuxGPUMegabytesToBytes(fields[2]); ok {
			device.MemoryUsedBytes = &value
		}
		if value, ok := linuxGPUFloat(fields[3]); ok {
			device.Utilization = &value
		}
		if value, ok := linuxGPUFloat(fields[4]); ok {
			device.TemperatureCelsius = &value
		}
		devices = append(devices, device)
	}
	if len(devices) == 0 {
		return schema.GPUSection{}, errors.New("nvidia-smi returned no gpu rows")
	}
	return schema.GPUSection{
		SectionStatus: schema.SectionStatus{Status: schema.StatusAvailable, Scope: schema.ScopeHost},
		Devices:       devices,
	}, nil
}

func linuxGPUMegabytesToBytes(value string) (uint64, bool) {
	parsed, ok := linuxGPUFloat(value)
	if !ok {
		return 0, false
	}
	return uint64(parsed * 1024 * 1024), true
}

func linuxGPUFloat(value string) (float64, bool) {
	value = strings.TrimSpace(value)
	if value == "" || strings.EqualFold(value, "[not supported]") || strings.EqualFold(value, "not supported") || strings.EqualFold(value, "n/a") {
		return 0, false
	}
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0, false
	}
	return parsed, true
}

func (c linuxGPUCollector) discoverSysfsGPU(detectorName string, vendorID string, vendor string, modelPrefix string) (linuxGPUDiscovery, bool) {
	for _, device := range c.drmDevices() {
		if !strings.EqualFold(device.VendorID, vendorID) {
			continue
		}
		device.Vendor = vendor
		if device.Model == "" {
			device.Model = modelPrefix
			if device.DeviceID != "" {
				device.Model += " " + device.DeviceID
			}
		}
		return linuxGPUDiscovery{
			Detector: detectorName,
			Status:   schema.StatusAvailable,
			Vendor:   vendor,
			Model:    device.Model,
			Devices:  []linuxGPUDevice{device},
		}, true
	}
	return linuxGPUDiscovery{}, false
}

func (c linuxGPUCollector) drmDevices() []linuxGPUDevice {
	dir := c.rootPath("sys/class/drm")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}

	devices := []linuxGPUDevice{}
	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasPrefix(name, "card") || strings.Contains(name, "-") {
			continue
		}
		devicePath := filepath.Join(dir, name, "device")
		vendorID := c.readTrimmed(filepath.Join(devicePath, "vendor"))
		if vendorID == "" {
			continue
		}
		deviceID := c.readTrimmed(filepath.Join(devicePath, "device"))
		devices = append(devices, linuxGPUDevice{
			Path:     devicePath,
			VendorID: strings.ToLower(vendorID),
			DeviceID: strings.ToLower(deviceID),
		})
	}
	return devices
}

func (c linuxGPUCollector) sysfsGPUSection(discovery linuxGPUDiscovery) schema.GPUSection {
	if len(discovery.Devices) == 0 {
		return linuxGPUSectionFromDiscovery(discovery)
	}

	devices := make([]schema.GPUDevice, 0, len(discovery.Devices))
	for _, discovered := range discovery.Devices {
		device := schema.GPUDevice{
			Vendor: discovered.Vendor,
			Model:  discovered.Model,
		}
		if device.Vendor == "" {
			device.Vendor = discovery.Vendor
		}
		if device.Model == "" {
			device.Model = discovery.Model
		}
		if value, ok := c.readUint64(filepath.Join(discovered.Path, "mem_info_vram_total")); ok {
			device.MemoryTotalBytes = &value
		}
		if value, ok := c.readUint64(filepath.Join(discovered.Path, "mem_info_vram_used")); ok {
			device.MemoryUsedBytes = &value
		}
		if value, ok := c.readFloat(filepath.Join(discovered.Path, "gpu_busy_percent")); ok {
			device.Utilization = &value
		}
		if value, ok := c.readHwmonTemperature(discovered.Path); ok {
			device.TemperatureCelsius = &value
		}
		devices = append(devices, device)
	}
	return schema.GPUSection{
		SectionStatus: schema.SectionStatus{Status: schema.StatusAvailable, Scope: schema.ScopeHost},
		Devices:       devices,
	}
}

func linuxGPUSectionFromDiscovery(discovery linuxGPUDiscovery) schema.GPUSection {
	device := schema.GPUDevice{
		Vendor: discovery.Vendor,
		Model:  discovery.Model,
	}
	if len(discovery.Devices) > 0 {
		if discovery.Devices[0].Vendor != "" {
			device.Vendor = discovery.Devices[0].Vendor
		}
		if discovery.Devices[0].Model != "" {
			device.Model = discovery.Devices[0].Model
		}
	}
	return schema.GPUSection{
		SectionStatus: schema.SectionStatus{Status: schema.StatusAvailable, Scope: schema.ScopeHost},
		Devices:       []schema.GPUDevice{device},
	}
}

func (c linuxGPUCollector) readHwmonTemperature(devicePath string) (float64, bool) {
	entries, err := os.ReadDir(filepath.Join(devicePath, "hwmon"))
	if err != nil {
		return 0, false
	}
	for _, entry := range entries {
		value, ok := c.readFloat(filepath.Join(devicePath, "hwmon", entry.Name(), "temp1_input"))
		if ok {
			return value / 1000, true
		}
	}
	return 0, false
}

func (c linuxGPUCollector) readUint64(path string) (uint64, bool) {
	value := c.readTrimmed(path)
	if value == "" {
		return 0, false
	}
	parsed, err := strconv.ParseUint(value, 10, 64)
	if err != nil {
		return 0, false
	}
	return parsed, true
}

func (c linuxGPUCollector) readFloat(path string) (float64, bool) {
	value := c.readTrimmed(path)
	if value == "" {
		return 0, false
	}
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0, false
	}
	return parsed, true
}

func (c linuxGPUCollector) readTrimmed(path string) string {
	out, err := c.readFile(path)
	if err != nil {
		return ""
	}
	return strings.Trim(strings.TrimSpace(string(out)), "\x00")
}

func (c linuxGPUCollector) rootPath(path string) string {
	if filepath.IsAbs(path) {
		path = strings.TrimPrefix(path, string(filepath.Separator))
	}
	if c.root == "/" {
		return filepath.Join(string(filepath.Separator), path)
	}
	return filepath.Join(c.root, path)
}
