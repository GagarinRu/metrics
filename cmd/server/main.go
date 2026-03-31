package main

import (
	"context"
	"flag"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"
	"github.com/GagarinRu/metrics/internal/handler"
	"github.com/GagarinRu/metrics/internal/logger"
	"github.com/GagarinRu/metrics/internal/storage"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	_ "github.com/lib/pq"
	"go.uber.org/zap"
)

func main() {
	var (
		addr            string
		logLevel        string
		storeInterval   int
		fileStoragePath string
		restore         bool
		databaseDSN     string
	)
	flag.StringVar(&addr, "a", ":8080", "Server address")
	flag.StringVar(&logLevel, "l", "info", "Log level")
	flag.IntVar(&storeInterval, "i", 300, "Interval storage in seconds")
	flag.StringVar(&fileStoragePath, "f", "metrics.json", "Path to file for storage metrics")
	flag.BoolVar(&restore, "r", false, "Restore metrics from a file at startup")
	flag.StringVar(&databaseDSN, "d", "", "Database DSN")
	flag.Parse()
	if envDsn := os.Getenv("DATABASE_DSN"); envDsn != "" {
		databaseDSN = strings.Trim(envDsn, `"'`)
	}
	if envAddr := os.Getenv("ADDRESS"); envAddr != "" {
		addr = envAddr
	}
	if envLogLevel := os.Getenv("LOG_LEVEL"); envLogLevel != "" {
		logLevel = envLogLevel
	}
	if envInterval := os.Getenv("STORE_INTERVAL"); envInterval != "" {
		if v, err := strconv.Atoi(envInterval); err == nil {
			storeInterval = v
		}
	}
	if envFile := os.Getenv("FILE_STORAGE_PATH"); envFile != "" {
		fileStoragePath = envFile
	}
	if envRestore := os.Getenv("RESTORE"); envRestore != "" {
		if v, err := strconv.ParseBool(envRestore); err == nil {
			restore = v
		}
	}

	if err := logger.Initialize(logLevel); err != nil {
		logger.Log.Fatal("Failed to initialize logger", zap.Error(err))
	}
	defer logger.Log.Sync()
	logger.Log.Info("Starting server",
		zap.String("address", addr),
		zap.String("file_storage_path", fileStoragePath),
		zap.Int("store_interval", storeInterval),
		zap.Bool("restore", restore),
		zap.String("database_dsn", databaseDSN),
	)

	store := storage.NewMemStorageWithFile(fileStoragePath, storeInterval, restore, databaseDSN)
	defer func() {
		store.Stop()
		store.Close()
	}()
	h := handler.NewHandler(store)
	r := chi.NewRouter()
	r.Use(middleware.StripSlashes)
	r.Get("/", h.GetAllMetrics)
	r.Get("/value/{metricType}/{metricName}", h.GetMetric)
	r.Post("/update/{metricType}/{metricName}/{metricValue}", h.UpdateMetrics)
	r.Post("/update", h.UpdateMetricsJSON)
	r.Post("/updates", h.UpdateMetricsBatch)
	r.Post("/value", h.GetMetricJSON)
	r.Get("/ping", h.PingDataBase)
	loggedRouter := logger.RequestLogger(gzipMiddleware(r))
	server := &http.Server{Addr: addr, Handler: loggedRouter}
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Log.Fatal("Server failed", zap.Error(err))
		}
	}()
	logger.Log.Info("Server started", zap.String("address", addr))
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		logger.Log.Fatal("Server shutdown failed", zap.Error(err))
	}
}

func gzipMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ow := w
		acceptEncoding := r.Header.Get("Accept-Encoding")
		supportsGzip := strings.Contains(acceptEncoding, "gzip")
		if supportsGzip {
			cw := newCompressWriter(w)
			ow = cw
			defer cw.Close()
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
			defer cr.Close()
		}
		next.ServeHTTP(ow, r)
	})
}
