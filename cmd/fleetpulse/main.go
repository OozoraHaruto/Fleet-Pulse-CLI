package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/haruto/fleetpulse/internal/api"
	"github.com/haruto/fleetpulse/internal/collector"
	"github.com/haruto/fleetpulse/internal/config"
	"github.com/haruto/fleetpulse/internal/token"
)

var (
	version = "dev"
	commit  = "unknown"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 || isFlag(args[0]) {
		return runServe(args, stdout, stderr)
	}

	switch args[0] {
	case "serve":
		return runServe(args[1:], stdout, stderr)
	case "token":
		return runToken(args[1:], stdout, stderr)
	case "diagnose":
		return runDiagnose(args[1:], stdout, stderr)
	case "config":
		return runConfig(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown command %q\n", args[0])
		return 2
	}
}

func runServe(args []string, stdout io.Writer, stderr io.Writer) int {
	cfg, ok := parseConfigFlags("serve", args, stderr)
	if !ok {
		return 2
	}
	if err := cfg.Validate(); err != nil {
		fmt.Fprintf(stderr, "configuration error: %v\n", err)
		return 1
	}

	bearerToken := ""
	if cfg.AuthEnabled {
		value, created, err := token.EnsureFile(cfg.TokenFile)
		if err != nil {
			fmt.Fprintf(stderr, "token provisioning failed: %v\n", err)
			return 1
		}
		bearerToken = value
		if created {
			fmt.Fprintf(stdout, "initial bearer token: %s\n", value)
		}
	}

	service := collector.NewServiceWithOptions(collector.Options{CacheTTL: cfg.CacheTTL})
	handler := api.NewHandlerWithOptions(service, api.Options{
		AuthEnabled: cfg.AuthEnabled,
		BearerToken: bearerToken,
	})

	fmt.Fprintf(stderr, "fleetpulse listening on %s\n", cfg.Addr)
	if err := http.ListenAndServe(cfg.Addr, handler); err != nil {
		fmt.Fprintf(stderr, "server failed: %v\n", err)
		return 1
	}
	return 0
}

func runToken(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "token command requires show or rotate")
		return 2
	}
	command := args[0]
	cfg, ok := parseConfigFlags("token "+command, args[1:], stderr)
	if !ok {
		return 2
	}

	switch command {
	case "show":
		value, err := token.LoadFile(cfg.TokenFile)
		if err != nil {
			fmt.Fprintf(stderr, "token show failed: %v\n", err)
			return 1
		}
		fmt.Fprintln(stdout, value)
		return 0
	case "rotate":
		value, err := token.RotateFile(cfg.TokenFile)
		if err != nil {
			fmt.Fprintf(stderr, "token rotate failed: %v\n", err)
			return 1
		}
		fmt.Fprintln(stdout, value)
		return 0
	default:
		fmt.Fprintf(stderr, "unknown token command %q\n", command)
		return 2
	}
}

func runDiagnose(args []string, stdout io.Writer, stderr io.Writer) int {
	cfg, ok := parseConfigFlags("diagnose", args, stderr)
	if !ok {
		return 2
	}
	_, tokenErr := os.Stat(cfg.TokenFile)
	report := map[string]any{
		"status":              "ok",
		"schema_version":      cfg.SchemaVersion,
		"version":             version,
		"commit":              commit,
		"addr":                cfg.Addr,
		"auth_enabled":        cfg.AuthEnabled,
		"non_local_bind":      config.IsNonLocalAddr(cfg.Addr),
		"token_file":          cfg.TokenFile,
		"token_file_present":  tokenErr == nil,
		"cache_ttl":           cfg.CacheTTL.String(),
		"collector_timeout":   cfg.CollectorTimeout.String(),
		"enabled_collectors":  cfg.EnabledCollectors,
		"service_name":        cfg.ServiceName,
		"deployment_target":   cfg.DeploymentTarget,
		"disk_health_enabled": cfg.DiskHealthEnabled,
	}
	return writeIndentedJSON(stdout, stderr, report)
}

func runConfig(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 || args[0] != "show" {
		fmt.Fprintln(stderr, "config command requires show")
		return 2
	}
	cfg, ok := parseConfigFlags("config show", args[1:], stderr)
	if !ok {
		return 2
	}
	return writeIndentedJSON(stdout, stderr, displayConfig(cfg))
}

func parseConfigFlags(name string, args []string, stderr io.Writer) (config.Config, bool) {
	defaults := config.Default()
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(stderr)
	configPath := fs.String("config", "", "path to JSON config file")
	addr := fs.String("addr", defaults.Addr, "address for the FleetPulse API server")
	authEnabled := fs.Bool("auth", defaults.AuthEnabled, "enable bearer token authentication")
	tokenFile := fs.String("token-file", defaults.TokenFile, "path to bearer token file")
	cacheTTL := fs.Duration("cache-ttl", defaults.CacheTTL, "snapshot cache TTL")
	collectorTimeout := fs.Duration("collector-timeout", defaults.CollectorTimeout, "collector timeout")
	logLevel := fs.String("log-level", defaults.LogLevel, "log level")
	serviceName := fs.String("service-name", defaults.ServiceName, "service name")
	deploymentTarget := fs.String("deployment-target", defaults.DeploymentTarget, "deployment target override")
	diskHealthEnabled := fs.Bool("disk-health", defaults.DiskHealthEnabled, "enable disk health collection")

	if err := fs.Parse(args); err != nil {
		return config.Config{}, false
	}
	cfg, err := config.LoadFile(*configPath)
	if err != nil {
		fmt.Fprintf(stderr, "config load failed: %v\n", err)
		return config.Config{}, false
	}
	cfg, err = config.ApplyEnv(cfg, os.LookupEnv)
	if err != nil {
		fmt.Fprintf(stderr, "environment config failed: %v\n", err)
		return config.Config{}, false
	}

	fs.Visit(func(f *flag.Flag) {
		switch f.Name {
		case "addr":
			cfg.Addr = *addr
		case "auth":
			cfg.AuthEnabled = *authEnabled
		case "token-file":
			cfg.TokenFile = *tokenFile
		case "cache-ttl":
			cfg.CacheTTL = *cacheTTL
		case "collector-timeout":
			cfg.CollectorTimeout = *collectorTimeout
		case "log-level":
			cfg.LogLevel = *logLevel
		case "service-name":
			cfg.ServiceName = *serviceName
		case "deployment-target":
			cfg.DeploymentTarget = *deploymentTarget
		case "disk-health":
			cfg.DiskHealthEnabled = *diskHealthEnabled
		}
	})
	return cfg, true
}

func writeIndentedJSON(stdout io.Writer, stderr io.Writer, value any) int {
	encoder := json.NewEncoder(stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(value); err != nil {
		fmt.Fprintf(stderr, "json encode failed: %v\n", err)
		return 1
	}
	return 0
}

func displayConfig(cfg config.Config) map[string]any {
	return map[string]any{
		"addr":                cfg.Addr,
		"auth_enabled":        cfg.AuthEnabled,
		"token_file":          cfg.TokenFile,
		"cache_ttl":           cfg.CacheTTL.String(),
		"collector_timeout":   cfg.CollectorTimeout.String(),
		"enabled_collectors":  cfg.EnabledCollectors,
		"log_level":           cfg.LogLevel,
		"schema_version":      cfg.SchemaVersion,
		"service_name":        cfg.ServiceName,
		"deployment_target":   cfg.DeploymentTarget,
		"disk_health_enabled": cfg.DiskHealthEnabled,
	}
}

func isFlag(arg string) bool {
	return len(arg) > 0 && arg[0] == '-'
}
