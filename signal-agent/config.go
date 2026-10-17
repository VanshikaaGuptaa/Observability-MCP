package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Config struct {
	GeminiAPIKey  string
	GeminiModel   string
	MCPWorkingDir string
	MCPCommand    string
	AgentPort     string
}

func loadLocalEnv(path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}

	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)

		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}

		key = strings.TrimSpace(key)
		value = strings.Trim(strings.TrimSpace(value), `"'`)

		if _, exists := os.LookupEnv(key); !exists {
			_ = os.Setenv(key, value)
		}
	}
}

func loadConfig() (Config, error) {
	loadLocalEnv(".env")

	cfg := Config{
		GeminiAPIKey:  os.Getenv("GEMINI_API_KEY"),
		GeminiModel:   os.Getenv("GEMINI_MODEL"),
		MCPWorkingDir: os.Getenv("MCP_WORKING_DIR"),
		MCPCommand:    os.Getenv("MCP_COMMAND"),
		AgentPort:     os.Getenv("AGENT_PORT"),
	}

	// GEMINI_API_KEY is optional. When set it acts as the fallback key;
	// users can also supply a key at runtime through the Settings UI.
	cfg.GeminiAPIKey = strings.TrimSpace(cfg.GeminiAPIKey)
	if cfg.GeminiModel == "" {
		return cfg, fmt.Errorf("GEMINI_MODEL is required")
	}
	if cfg.MCPWorkingDir == "" {
		cfg.MCPWorkingDir = "../observability-mcp"
	}
	if cfg.MCPCommand == "" {
		cfg.MCPCommand = "go"
	}
	if cfg.AgentPort == "" {
		cfg.AgentPort = "8090"
	}

	absDir, err := filepath.Abs(cfg.MCPWorkingDir)
	if err != nil {
		return cfg, err
	}
	cfg.MCPWorkingDir = absDir

	if _, err := os.Stat(filepath.Join(cfg.MCPWorkingDir, "go.mod")); err != nil {
		return cfg, fmt.Errorf(
			"MCP module not found at %s: %w",
			cfg.MCPWorkingDir,
			err,
		)
	}

	return cfg, nil
}
