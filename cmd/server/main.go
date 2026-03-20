package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"github.com/GagarinRu/metrics/internal/handler"
	"github.com/GagarinRu/metrics/internal/logger"
	"github.com/GagarinRu/metrics/internal/storage"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

func main() {
	var (
		addr     string
		logLevel string
	)
	flag.StringVar(&addr, "a", ":8080", "server address")
	flag.StringVar(&logLevel, "l", "info", "log level")
	flag.Parse()

	if envAddr := os.Getenv("ADDRESS"); envAddr != "" {
		addr = envAddr
	}
	if envLogLevel := os.Getenv("LOG_LEVEL"); envLogLevel != "" {
		logLevel = envLogLevel
	}
	if err := logger.Initialize(logLevel); err != nil {
		logger.Log.Fatal("Failed to initialize logger", zap.Error(err))
	}
	defer logger.Log.Sync()

	logger.Log.Info("Starting server", zap.String("address", addr))

	store := storage.NewMemStorage()
	h := handler.NewHandler(store)
	r := chi.NewRouter()
	r.Get("/", h.GetAllMetrics)
	r.Get("/value/{metricType}/{metricName}", h.GetMetric)
	r.Post("/update/{metricType}/{metricName}/{metricValue}", h.UpdateMetrics)

	loggedRouter := logger.RequestLogger(r)

	fmt.Printf("Server started on %s\n", addr)
	if err := http.ListenAndServe(addr, loggedRouter); err != nil {
		logger.Log.Fatal("Server failed", zap.Error(err))
	}
}
