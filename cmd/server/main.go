package main

import (
	"context"
	"crypto/rsa"
	"flag"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/GagarinRu/metrics/internal/audit"
	"github.com/GagarinRu/metrics/internal/config"
	"github.com/GagarinRu/metrics/internal/crypto"
	"github.com/GagarinRu/metrics/internal/handler"
	"github.com/GagarinRu/metrics/internal/logger"
	"github.com/GagarinRu/metrics/internal/storage"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	_ "github.com/lib/pq"
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

	opts := config.ServerOptions{
		Address:         ":8080",
		LogLevel:        "info",
		StoreInterval:   300,
		FileStoragePath: "metrics.json",
		Restore:         false,
	}

	if configPath := config.ConfigPath(); configPath != "" {
		file, err := config.ReadServerJSON(configPath)
		if err != nil {
			panic("failed to load config: " + err.Error())
		}
		opts, err = config.ApplyServerJSON(opts, file)
		if err != nil {
			panic("failed to apply config: " + err.Error())
		}
	}

	var (
		addr            string
		logLevel        string
		storeInterval   int
		fileStoragePath string
		restore         bool
		databaseDSN     string
		key             string
		cryptoKey       string
		auditFile       string
		auditURL        string
	)
	flag.StringVar(&addr, "a", opts.Address, "Server address")
	flag.StringVar(&logLevel, "l", opts.LogLevel, "Log level")
	flag.IntVar(&storeInterval, "i", opts.StoreInterval, "Interval storage in seconds")
	flag.StringVar(&fileStoragePath, "f", opts.FileStoragePath, "Path to file for storage metrics")
	flag.BoolVar(&restore, "r", opts.Restore, "Restore metrics from a file at startup")
	flag.StringVar(&databaseDSN, "d", opts.DatabaseDSN, "Database DSN")
	flag.StringVar(&key, "k", opts.Key, "Key for hash calculation")
	flag.StringVar(&cryptoKey, "crypto-key", opts.CryptoKeyPath, "Path to private key for decryption")
	flag.StringVar(&auditFile, "audit-file", opts.AuditFile, "Path to audit log file")
	flag.StringVar(&auditURL, "audit-url", opts.AuditURL, "URL for audit log delivery")
	flag.StringVar(new(string), "c", "", "Path to JSON config file")
	flag.StringVar(new(string), "config", "", "Path to JSON config file")
	flag.Parse()

	visited := make(map[string]bool)
	flag.Visit(func(f *flag.Flag) {
		visited[f.Name] = true
	})
	if visited["a"] {
		opts.Address = addr
	}
	if visited["l"] {
		opts.LogLevel = logLevel
	}
	if visited["i"] {
		opts.StoreInterval = storeInterval
	}
	if visited["f"] {
		opts.FileStoragePath = fileStoragePath
	}
	if visited["r"] {
		opts.Restore = restore
	}
	if visited["d"] {
		opts.DatabaseDSN = databaseDSN
	}
	if visited["k"] {
		opts.Key = key
	}
	if visited["crypto-key"] {
		opts.CryptoKeyPath = cryptoKey
	}
	if visited["audit-file"] {
		opts.AuditFile = auditFile
	}
	if visited["audit-url"] {
		opts.AuditURL = auditURL
	}

	opts = config.ApplyServerEnv(opts)

	if err := logger.Initialize(opts.LogLevel); err != nil {
		logger.Log.Fatal("Failed to initialize logger", zap.Error(err))
	}
	defer func() { _ = logger.Log.Sync() }()
	logger.Log.Info("Starting server",
		zap.String("address", opts.Address),
		zap.String("file_storage_path", opts.FileStoragePath),
		zap.Int("store_interval", opts.StoreInterval),
		zap.Bool("restore", opts.Restore),
		zap.String("database_dsn", opts.DatabaseDSN),
	)

	auditor := audit.NewPublisher(opts.AuditFile, opts.AuditURL)
	defer func() { _ = auditor.Close() }()
	store := storage.NewMemStorageWithFile(opts.FileStoragePath, opts.StoreInterval, opts.Restore, opts.DatabaseDSN)
	defer func() {
		store.Stop()
		_ = store.Close()
	}()
	var privateKey *rsa.PrivateKey
	if opts.CryptoKeyPath != "" {
		var err error
		privateKey, err = crypto.LoadPrivateKey(opts.CryptoKeyPath)
		if err != nil {
			logger.Log.Fatal("Failed to load private key", zap.String("path", opts.CryptoKeyPath), zap.Error(err))
		}
	}
	h := handler.NewHandler(store, opts.Key, privateKey, auditor)
	r := chi.NewRouter()
	r.Use(middleware.StripSlashes)
	r.Use(h.DecryptMiddleware)
	r.Use(h.HashMiddleware)
	r.Mount("/debug", middleware.Profiler())
	r.Get("/", h.GetAllMetrics)
	r.Get("/value/{metricType}/{metricName}", h.GetMetric)
	r.Post("/update/{metricType}/{metricName}/{metricValue}", h.UpdateMetrics)
	r.Post("/update", h.UpdateMetricsJSON)
	r.Post("/updates", h.UpdateMetricsBatch)
	r.Post("/value", h.GetMetricJSON)
	r.Get("/ping", h.PingDataBase)
	loggedRouter := logger.RequestLogger(gzipMiddleware(r))
	server := &http.Server{Addr: opts.Address, Handler: loggedRouter}
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Log.Fatal("Server failed", zap.Error(err))
		}
	}()
	logger.Log.Info("Server started", zap.String("address", opts.Address))

	ctx, stop := signal.NotifyContext(context.Background(), shutdownSignals()...)
	defer stop()
	<-ctx.Done()
	logger.Log.Info("Received shutdown signal")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Log.Fatal("Server shutdown failed", zap.Error(err))
	}
	logger.Log.Info("Server stopped gracefully")
}

func gzipMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ow := w
		acceptEncoding := r.Header.Get("Accept-Encoding")
		supportsGzip := strings.Contains(acceptEncoding, "gzip")
		if supportsGzip {
			cw := newCompressWriter(w)
			ow = cw
			defer func() { _ = cw.Close() }()
		}
		contentEncoding := r.Header.Get("Content-Encoding")
		sendsGzip := strings.Contains(contentEncoding, "gzip")
		if sendsGzip {
			cr, err := newCompressReader(r.Body)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			r.Body = cr
			defer func() { _ = cr.Close() }()
		}
		next.ServeHTTP(ow, r)
	})
}
