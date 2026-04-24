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

func getEnvString(envName string, defaultValue string) string {
	if envVal := os.Getenv(envName); envVal != "" {
		return strings.Trim(envVal, `"'`)
	}
	return defaultValue
}

func getEnvBool(envName string) bool {
	if envVal := os.Getenv(envName); envVal != "" {
		if v, err := strconv.ParseBool(envVal); err == nil {
			return v
		}
	}
	return false
}

func getEnvInt(envName string, defaultValue int) int {
	if envVal := os.Getenv(envName); envVal != "" {
		if v, err := strconv.Atoi(envVal); err == nil {
			return v
		}
	}
	return defaultValue
}

func main() {
	var (
		addr            string
		logLevel        string
		storeInterval   int
		fileStoragePath string
		restore         bool
		databaseDSN     string
		key             string
	)
	flag.StringVar(&addr, "a", ":8080", "Server address")
	flag.StringVar(&logLevel, "l", "info", "Log level")
	flag.IntVar(&storeInterval, "i", 300, "Interval storage in seconds")
	flag.StringVar(&fileStoragePath, "f", "metrics.json", "Path to file for storage metrics")
	flag.BoolVar(&restore, "r", false, "Restore metrics from a file at startup")
	flag.StringVar(&databaseDSN, "d", "", "Database DSN")
	flag.StringVar(&key, "k", "", "Key for hash calculation")
	flag.Parse()
	databaseDSN = getEnvString("DATABASE_DSN", databaseDSN)
	addr = getEnvString("ADDRESS", addr)
	logLevel = getEnvString("LOG_LEVEL", logLevel)
	storeInterval = getEnvInt("STORE_INTERVAL", storeInterval)
	fileStoragePath = getEnvString("FILE_STORAGE_PATH", fileStoragePath)
	restore = getEnvBool("RESTORE")
	key = getEnvString("KEY", key)

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
	h := handler.NewHandler(store, key)
	r := chi.NewRouter()
	r.Use(middleware.StripSlashes)
	r.Use(h.HashMiddleware)
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
