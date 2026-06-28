package main

import (
	"context"
	"crypto/rsa"
	"flag"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/GagarinRu/metrics/internal/agent"
	"github.com/GagarinRu/metrics/internal/config"
	"github.com/GagarinRu/metrics/internal/crypto"
	"github.com/GagarinRu/metrics/internal/logger"
	"go.uber.org/zap"
)

func shutdownSignals() []os.Signal {
	sigs := []os.Signal{os.Interrupt, syscall.SIGTERM}
	if runtime.GOOS != "windows" {
		sigs = append(sigs, syscall.SIGQUIT)
	}
	return sigs
}

func main() {
	printBuildInfo()

	opts := config.AgentOptions{
		ServerAddr:     "http://localhost:8080",
		LogLevel:       "info",
		PollInterval:   2,
		ReportInterval: 10,
		RateLimit:      1,
	}

	if configPath := config.ConfigPath(); configPath != "" {
		file, err := config.ReadAgentJSON(configPath)
		if err != nil {
			panic("failed to load config: " + err.Error())
		}
		opts, err = config.ApplyAgentJSON(opts, file)
		if err != nil {
			panic("failed to apply config: " + err.Error())
		}
	}

	var (
		pollInterval   int
		logLevel       string
		reportInterval int
		serverAddr     string
		key            string
		cryptoKey      string
		rateLimit      int
	)
	flag.IntVar(&pollInterval, "p", opts.PollInterval, "Poll interval in seconds")
	flag.IntVar(&reportInterval, "r", opts.ReportInterval, "Report interval in seconds")
	flag.StringVar(&serverAddr, "a", opts.ServerAddr, "Server address")
	flag.StringVar(&logLevel, "l", opts.LogLevel, "Log level")
	flag.StringVar(&key, "k", opts.Key, "Key for hash calculation")
	flag.StringVar(&cryptoKey, "crypto-key", opts.CryptoKeyPath, "Path to public key for encryption")
	flag.IntVar(&rateLimit, "rate-limit", opts.RateLimit, "Rate limit (concurrent batch requests)")
	flag.StringVar(new(string), "c", "", "Path to JSON config file")
	flag.StringVar(new(string), "config", "", "Path to JSON config file")
	flag.Parse()

	visited := make(map[string]bool)
	flag.Visit(func(f *flag.Flag) {
		visited[f.Name] = true
	})
	if visited["p"] {
		opts.PollInterval = pollInterval
	}
	if visited["r"] {
		opts.ReportInterval = reportInterval
	}
	if visited["a"] {
		opts.ServerAddr = serverAddr
	}
	if visited["l"] {
		opts.LogLevel = logLevel
	}
	if visited["k"] {
		opts.Key = key
	}
	if visited["crypto-key"] {
		opts.CryptoKeyPath = cryptoKey
	}
	if visited["rate-limit"] {
		opts.RateLimit = rateLimit
	}

	opts = config.ApplyAgentEnv(opts)

	if err := logger.Initialize(opts.LogLevel); err != nil {
		panic("Failed to initialize logger: " + err.Error())
	}
	defer func() { _ = logger.Log.Sync() }()
	if flag.NArg() > 0 {
		logger.Log.Error("Unknown arguments", zap.Strings("args", flag.Args()))
		exit(1)
	}
	if opts.PollInterval <= 0 || opts.ReportInterval <= 0 {
		logger.Log.Error("Poll and report intervals must be positive",
			zap.Int("poll_interval", opts.PollInterval),
			zap.Int("report_interval", opts.ReportInterval))
		exit(1)
	}

	var publicKey *rsa.PublicKey
	if opts.CryptoKeyPath != "" {
		var err error
		publicKey, err = crypto.LoadPublicKey(opts.CryptoKeyPath)
		if err != nil {
			logger.Log.Fatal("Failed to load public key", zap.String("path", opts.CryptoKeyPath), zap.Error(err))
		}
	}

	cfg := agent.Config{
		PollInterval:   time.Duration(opts.PollInterval) * time.Second,
		ReportInterval: time.Duration(opts.ReportInterval) * time.Second,
		ServerAddr:     opts.ServerAddr,
		UseGzip:        true,
		Key:            opts.Key,
		PublicKey:      publicKey,
		RateLimit:      opts.RateLimit,
	}
	a := agent.NewAgent(cfg)
	logger.Log.Info("Starting agent",
		zap.Int("poll_interval", opts.PollInterval),
		zap.Int("report_interval", opts.ReportInterval),
		zap.Int("rate_limit", opts.RateLimit))
	logger.Log.Info("Sending metrics to", zap.String("server_addr", cfg.ServerAddr))

	ctx, stop := signal.NotifyContext(context.Background(), shutdownSignals()...)
	defer stop()

	go func() {
		if err := a.Run(ctx); err != nil && err != context.Canceled {
			logger.Log.Error("Agent error", zap.Error(err))
			exit(1)
		}
	}()

	<-ctx.Done()
	logger.Log.Info("Received shutdown signal")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := a.Shutdown(shutdownCtx); err != nil {
		logger.Log.Fatal("Agent shutdown failed", zap.Error(err))
	}
	logger.Log.Info("Agent stopped gracefully")
}
