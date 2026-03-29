package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
	"github.com/GagarinRu/metrics/internal/agent"
	"github.com/GagarinRu/metrics/internal/logger"
	"go.uber.org/zap"
)

func getEnvInt(envName string, defaultValue int) int {
	if envVal := os.Getenv(envName); envVal != "" {
		if val, err := strconv.Atoi(envVal); err == nil {
			if val > 0 {
				return val
			}
			logger.Log.Warn("Environment variable must be positive, using default",
				zap.String("env", envName),
				zap.Int("value", val),
				zap.Int("default", defaultValue))
		}
	}
	return defaultValue
}

func main() {
	var (
		pollInterval   int
		logLevel        string
		reportInterval int
		serverAddr     string
	)
	flag.IntVar(&pollInterval, "p", 2, "Poll interval in seconds")
	flag.IntVar(&reportInterval, "r", 10, "Report interval in seconds")
	flag.StringVar(&serverAddr, "a", "http://localhost:8080", "Server address")
	flag.StringVar(&logLevel, "l", "info", "Log level")
	flag.Parse()
	if envLogLevel := os.Getenv("LOG_LEVEL"); envLogLevel != "" {
        logLevel = envLogLevel
	}
    if err := logger.Initialize(logLevel); err != nil {
         panic("Failed to initialize logger: " + err.Error())
     }
    defer logger.Log.Sync()
    if flag.NArg() > 0 {
		logger.Log.Error("Unknown arguments", zap.Strings("args", flag.Args()))
		os.Exit(1)
	}
	if envAddr := os.Getenv("ADDRESS"); envAddr != "" {
		serverAddr = envAddr
	}
	pollInterval = getEnvInt("POLL_INTERVAL", pollInterval)
	reportInterval = getEnvInt("REPORT_INTERVAL", reportInterval)
	cfg := agent.Config{
		PollInterval:   time.Duration(pollInterval) * time.Second,
		ReportInterval: time.Duration(reportInterval) * time.Second,
		ServerAddr:     serverAddr,
		UseGzip:        true,
	}
	a := agent.NewAgent(cfg)
	logger.Log.Info("Starting agent",
		zap.Int("poll_interval", pollInterval),
		zap.Int("report_interval", reportInterval))
	logger.Log.Info("Sending metrics to", zap.String("server_addr", cfg.ServerAddr))
	go func() {
		if err := a.Run(context.Background()); err != nil && err != context.Canceled {
			logger.Log.Error("Agent error", zap.Error(err))
			os.Exit(1)
		}
	}()
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan
	logger.Log.Info("Received shutdown signal")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := a.Shutdown(ctx); err != nil {
		logger.Log.Fatal("Agent shutdown failed", zap.Error(err))
	}
	logger.Log.Info("Agent stopped gracefully")
}
