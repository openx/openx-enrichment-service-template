package config

import (
	"os"
	"strconv"
	"time"
)

// Config represents the service configuration
type Config struct {
	Port         int
	LogLevel     string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration

	// Simulation parameters
	SimulateLatency   bool
	LatencyMeanMs     int
	LatencyStdDevMs   int
	SimulateCPULoad   bool
	CPULoadPercentage int

	// GCS configuration
	GCSInboxBucket string
	GCSCredentials string // Path to credentials file or "default" for workload identity
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	return &Config{
		Port:         8080,
		LogLevel:     "info",
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
		IdleTimeout:  120 * time.Second,

		SimulateLatency:   false,
		LatencyMeanMs:     5,
		LatencyStdDevMs:   2,
		SimulateCPULoad:   false,
		CPULoadPercentage: 50,

		GCSInboxBucket: "",
		GCSCredentials: "default", // Default to workload identity
	}
}

// LoadFromEnv loads configuration from environment variables
func LoadFromEnv() *Config {
	config := DefaultConfig()

	if port := os.Getenv("PORT"); port != "" {
		if p, err := strconv.Atoi(port); err == nil {
			config.Port = p
		}
	}

	if logLevel := os.Getenv("LOG_LEVEL"); logLevel != "" {
		config.LogLevel = logLevel
	}

	// Simulation parameters
	if simulateLatency := os.Getenv("SIMULATE_LATENCY"); simulateLatency != "" {
		config.SimulateLatency = simulateLatency == "true"
	}

	if latencyMean := os.Getenv("LATENCY_MEAN_MS"); latencyMean != "" {
		if mean, err := strconv.Atoi(latencyMean); err == nil {
			config.LatencyMeanMs = mean
		}
	}

	if latencyStdDev := os.Getenv("LATENCY_STDDEV_MS"); latencyStdDev != "" {
		if stddev, err := strconv.Atoi(latencyStdDev); err == nil {
			config.LatencyStdDevMs = stddev
		}
	}

	if simulateCPULoad := os.Getenv("SIMULATE_CPU_LOAD"); simulateCPULoad != "" {
		config.SimulateCPULoad = simulateCPULoad == "true"
	}

	if cpuLoadPercentage := os.Getenv("CPU_LOAD_PERCENTAGE"); cpuLoadPercentage != "" {
		if percentage, err := strconv.Atoi(cpuLoadPercentage); err == nil {
			config.CPULoadPercentage = percentage
		}
	}

	if inboxBucket := os.Getenv("GCS_INBOX_BUCKET"); inboxBucket != "" {
		config.GCSInboxBucket = inboxBucket
	}

	if credentials := os.Getenv("GCS_CREDENTIALS"); credentials != "" {
		config.GCSCredentials = credentials
	}

	return config
}
