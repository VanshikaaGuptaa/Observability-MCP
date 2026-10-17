package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	cfg, err := loadConfig()
	if err != nil {
		log.Fatal(err)
	}

	// Keep the MCP subprocess alive for the API's lifetime.
	mcpCtx, cancelMCP := context.WithCancel(context.Background())
	defer cancelMCP()

	mcpClient, err := ConnectMCP(mcpCtx, cfg)
	if err != nil {
		log.Fatalf("MCP startup failed: %v", err)
	}
	defer mcpClient.Close()

	geminiKeys := NewGeminiKeyStore(cfg.GeminiAPIKey)
	if !geminiKeys.Status().Configured {
		log.Printf("GEMINI_API_KEY not set; configure a key in Settings to enable AI analysis")
	}
	gemini := NewGeminiAnalyzer(cfg, geminiKeys)

	api := &API{
		mcp:      mcpClient,
		gemini:   gemini,
		settings: geminiKeys,
		tester:   gemini,
		// Opt-in only. Settings are unauthenticated, so by default they
		// accept requests from the local machine only.
		allowRemoteSettings: os.Getenv("SETTINGS_ALLOW_REMOTE") == "true",
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", api.Health)
	mux.HandleFunc("/api/investigations", api.Investigate)
	mux.HandleFunc("/api/settings/gemini", api.GeminiSettings)
	mux.HandleFunc("/api/settings/gemini/test", api.TestGeminiSettings)

	server := &http.Server{
		Addr:              ":" + cfg.AgentPort,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Printf("SignalSleuth agent listening on :%s", cfg.AgentPort)
		if err := server.ListenAndServe(); err != nil &&
			!errors.Is(err, http.ErrServerClosed) {
			log.Printf("HTTP server error: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	shutdownCtx, cancel := context.WithTimeout(
		context.Background(),
		10*time.Second,
	)
	defer cancel()

	_ = server.Shutdown(shutdownCtx)
}
