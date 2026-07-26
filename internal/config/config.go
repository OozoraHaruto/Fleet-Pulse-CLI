package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Addr              string        `json:"addr"`
	AuthEnabled       bool          `json:"auth_enabled"`
	TokenFile         string        `json:"token_file"`
	CacheTTL          time.Duration `json:"cache_ttl"`
	CollectorTimeout  time.Duration `json:"collector_timeout"`
	EnabledCollectors []string      `json:"enabled_collectors"`
	LogLevel          string        `json:"log_level"`
	SchemaVersion     string        `json:"schema_version"`
	ServiceName       string        `json:"service_name"`
	DeploymentTarget  string        `json:"deployment_target"`
	DiskHealthEnabled bool          `json:"disk_health_enabled"`
}

func Default() Config {
	return Config{
		Addr:              "127.0.0.1:8080",
		AuthEnabled:       false,
		TokenFile:         defaultTokenFile(),
		CacheTTL:          10 * time.Second,
		CollectorTimeout:  2 * time.Second,
		EnabledCollectors: []string{"cpu", "memory", "disks", "gpu", "system"},
		LogLevel:          "info",
		SchemaVersion:     "v1",
		ServiceName:       "fleetpulse",
		DeploymentTarget:  runtime.GOOS,
		DiskHealthEnabled: true,
	}
}

func LoadFile(path string) (Config, error) {
	cfg := Default()
	if path == "" {
		return cfg, nil
	}

	body, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}

	var raw struct {
		Addr              *string  `json:"addr"`
		AuthEnabled       *bool    `json:"auth_enabled"`
		TokenFile         *string  `json:"token_file"`
		CacheTTL          *string  `json:"cache_ttl"`
		CollectorTimeout  *string  `json:"collector_timeout"`
		EnabledCollectors []string `json:"enabled_collectors"`
		LogLevel          *string  `json:"log_level"`
		SchemaVersion     *string  `json:"schema_version"`
		ServiceName       *string  `json:"service_name"`
		DeploymentTarget  *string  `json:"deployment_target"`
		DiskHealthEnabled *bool    `json:"disk_health_enabled"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return Config{}, err
	}

	if raw.Addr != nil {
		cfg.Addr = *raw.Addr
	}
	if raw.AuthEnabled != nil {
		cfg.AuthEnabled = *raw.AuthEnabled
	}
	if raw.TokenFile != nil {
		cfg.TokenFile = *raw.TokenFile
	}
	if raw.CacheTTL != nil {
		parsed, err := time.ParseDuration(*raw.CacheTTL)
		if err != nil {
			return Config{}, fmt.Errorf("cache_ttl: %w", err)
		}
		cfg.CacheTTL = parsed
	}
	if raw.CollectorTimeout != nil {
		parsed, err := time.ParseDuration(*raw.CollectorTimeout)
		if err != nil {
			return Config{}, fmt.Errorf("collector_timeout: %w", err)
		}
		cfg.CollectorTimeout = parsed
	}
	if raw.EnabledCollectors != nil {
		cfg.EnabledCollectors = raw.EnabledCollectors
	}
	if raw.LogLevel != nil {
		cfg.LogLevel = *raw.LogLevel
	}
	if raw.SchemaVersion != nil {
		cfg.SchemaVersion = *raw.SchemaVersion
	}
	if raw.ServiceName != nil {
		cfg.ServiceName = *raw.ServiceName
	}
	if raw.DeploymentTarget != nil {
		cfg.DeploymentTarget = *raw.DeploymentTarget
	}
	if raw.DiskHealthEnabled != nil {
		cfg.DiskHealthEnabled = *raw.DiskHealthEnabled
	}

	return cfg, nil
}

func ApplyEnv(cfg Config, lookup func(string) (string, bool)) (Config, error) {
	if lookup == nil {
		lookup = os.LookupEnv
	}
	if value, ok := lookup("FLEETPULSE_ADDR"); ok {
		cfg.Addr = value
	}
	if value, ok := lookup("FLEETPULSE_AUTH_ENABLED"); ok {
		parsed, err := strconv.ParseBool(value)
		if err != nil {
			return Config{}, fmt.Errorf("FLEETPULSE_AUTH_ENABLED: %w", err)
		}
		cfg.AuthEnabled = parsed
	}
	if value, ok := lookup("FLEETPULSE_TOKEN_FILE"); ok {
		cfg.TokenFile = value
	}
	if value, ok := lookup("FLEETPULSE_CACHE_TTL"); ok {
		parsed, err := time.ParseDuration(value)
		if err != nil {
			return Config{}, fmt.Errorf("FLEETPULSE_CACHE_TTL: %w", err)
		}
		cfg.CacheTTL = parsed
	}
	if value, ok := lookup("FLEETPULSE_COLLECTOR_TIMEOUT"); ok {
		parsed, err := time.ParseDuration(value)
		if err != nil {
			return Config{}, fmt.Errorf("FLEETPULSE_COLLECTOR_TIMEOUT: %w", err)
		}
		cfg.CollectorTimeout = parsed
	}
	return cfg, nil
}

func (c Config) Validate() error {
	if c.Addr == "" {
		return errors.New("addr is required")
	}
	if c.CacheTTL < 0 {
		return errors.New("cache_ttl must not be negative")
	}
	if c.CollectorTimeout <= 0 {
		return errors.New("collector_timeout must be positive")
	}
	if IsNonLocalAddr(c.Addr) && !c.AuthEnabled {
		return fmt.Errorf("authentication is required when binding to non-local address %q", c.Addr)
	}
	if c.AuthEnabled && c.TokenFile == "" {
		return errors.New("token_file is required when authentication is enabled")
	}
	return nil
}

func IsNonLocalAddr(addr string) bool {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		host = addr
	}
	host = strings.Trim(host, "[]")
	if host == "" || host == "0.0.0.0" || host == "::" {
		return true
	}
	if strings.EqualFold(host, "localhost") {
		return false
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return true
	}
	return !ip.IsLoopback()
}

func defaultTokenFile() string {
	if runtime.GOOS == "windows" {
		return `C:\ProgramData\FleetPulse\token`
	}
	return "/var/lib/fleetpulse/token"
}
