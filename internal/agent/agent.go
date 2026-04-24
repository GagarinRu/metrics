package agent

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
	"github.com/GagarinRu/metrics/internal/logger"
	"github.com/GagarinRu/metrics/internal/metrics"
	"github.com/GagarinRu/metrics/internal/models"
	"go.uber.org/zap"
)

const (
	maxRetries     = 3
	retryInterval1 = 1 * time.Second
	retryInterval2 = 3 * time.Second
	retryInterval3 = 5 * time.Second
)

func isRetryableError(err error) bool {
	if err == nil {
		return false
	}
	var netErr net.Error
	if errors.As(err, &netErr) {
		return netErr.Temporary() || netErr.Timeout()
	}
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		return urlErr.Temporary() || urlErr.Timeout()
	}
	return false
}

func (a *Agent) sendWithRetry(req *http.Request) (*http.Response, error) {
	intervals := []time.Duration{retryInterval1, retryInterval2, retryInterval3}

	var lastErr error
	for i := 0; i <= maxRetries; i++ {
		resp, err := a.client.Do(req)
		if err == nil {
			return resp, nil
		}
		lastErr = err
		if !isRetryableError(err) {
			return nil, err
		}
		if i < maxRetries {
			logger.Log.Warn("Request failed, retrying",
				zap.Error(err),
				zap.Int("attempt", i+1),
				zap.Duration("interval", intervals[i]))
			time.Sleep(intervals[i])
		}
	}
	return nil, lastErr
}

type metricJob struct {
	metricType string
	name       string
	value      interface{}
}

type Agent struct {
	metrics        *metrics.Metrics
	pollInterval   time.Duration
	reportInterval time.Duration
	serverAddr     string
	client         *http.Client
	useGzip        bool
	useBatch       bool
	key            string
	rateLimit      int
	ctx            context.Context
	cancel         context.CancelFunc
	wg             sync.WaitGroup
	jobs           chan metricJob
}

type Config struct {
	PollInterval   time.Duration
	ReportInterval time.Duration
	ServerAddr     string
	UseGzip        bool
	UseBatch       *bool
	Key            string
	RateLimit      int
}

func NewAgent(cfg Config) *Agent {
	ctx, cancel := context.WithCancel(context.Background())
	useBatch := true
	if cfg.UseBatch != nil {
		useBatch = *cfg.UseBatch
	}
	rateLimit := cfg.RateLimit
	if rateLimit <= 0 {
		rateLimit = 1
	}
	return &Agent{
		metrics:        metrics.NewMetrics(),
		pollInterval:   cfg.PollInterval,
		reportInterval: cfg.ReportInterval,
		serverAddr:     cfg.ServerAddr,
		client:         &http.Client{Timeout: 5 * time.Second},
		useGzip:        cfg.UseGzip,
		useBatch:       useBatch,
		key:            cfg.Key,
		rateLimit:      rateLimit,
		ctx:            ctx,
		cancel:         cancel,
		jobs:           make(chan metricJob, rateLimit*2),
	}
}

func calculateHash(data []byte, key string) []byte {
	hash := sha256.Sum256(append(data, []byte(key)...))
	return hash[:]
}

func (a *Agent) collectMetrics() {
	defer a.wg.Done()
	ticker := time.NewTicker(a.pollInterval)
	defer ticker.Stop()
	a.metrics.UpdateRuntimeMetrics()
	a.metrics.UpdateSystemMetrics()
	for {
		select {
		case <-ticker.C:
			a.metrics.UpdateRuntimeMetrics()
			a.metrics.UpdateSystemMetrics()
		case <-a.ctx.Done():
			logger.Log.Info("Stopping metrics collection goroutine")
			return
		}
	}
}

func (a *Agent) reportMetrics() {
	defer a.wg.Done()
	ticker := time.NewTicker(a.reportInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			a.sendMetricsToServer()
		case <-a.ctx.Done():
			logger.Log.Info("Sending final metrics before shutdown")
			a.sendMetricsToServer()
			return
		}
	}
}

func (a *Agent) workerPool() {
	defer a.wg.Done()
	for job := range a.jobs {
		switch job.metricType {
		case "gauge":
			if v, ok := job.value.(float64); ok {
				a.sendMetric(job.metricType, job.name, v)
			}
		case "counter":
			if v, ok := job.value.(int64); ok {
				a.sendMetric(job.metricType, job.name, v)
			}
		}
	}
}

func (a *Agent) Run(ctx context.Context) error {
	a.wg.Add(1)
	go a.collectMetrics()

	a.wg.Add(1)
	go a.reportMetrics()

	for i := 0; i < a.rateLimit; i++ {
		a.wg.Add(1)
		go a.workerPool()
	}

	<-ctx.Done()

	close(a.jobs)
	return ctx.Err()
}

func (a *Agent) Shutdown(ctx context.Context) error {
	logger.Log.Info("Shutting down agent")
	a.cancel()
	done := make(chan struct{})
	go func() {
		a.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
		logger.Log.Info("Agent stopped gracefully")
		return nil
	case <-ctx.Done():
		logger.Log.Error("Agent shutdown timeout", zap.Error(ctx.Err()))
		return ctx.Err()
	}
}

func (a *Agent) Close() {
	close(a.jobs)
}

func (a *Agent) collectAllMetrics() []models.Metrics {
	var allMetrics []models.Metrics
	for name, value := range a.metrics.GetGauges() {
		v := value
		allMetrics = append(allMetrics, models.Metrics{
			ID:    name,
			MType: "gauge",
			Value: &v,
		})
	}
	for name, value := range a.metrics.GetCounters() {
		d := value
		allMetrics = append(allMetrics, models.Metrics{
			ID:    name,
			MType: "counter",
			Delta: &d,
		})
	}
	return allMetrics
}

func (a *Agent) sendMetricsToServer() {
	allMetrics := a.collectAllMetrics()
	if len(allMetrics) == 0 {
		return
	}
	a.sendBatch(allMetrics)
}

func (a *Agent) sendBatch(metrics []models.Metrics) error {
	jsonData, err := json.Marshal(metrics)
	if err != nil {
		logger.Log.Error("Failed to marshal batch", zap.Error(err))
		return err
	}
	bodyReader := bytes.NewReader(jsonData)
	if a.useGzip {
		var buf bytes.Buffer
		zw := gzip.NewWriter(&buf)
		if _, err := zw.Write(jsonData); err != nil {
			logger.Log.Error("Failed to compress batch", zap.Error(err))
			return err
		}
		if err := zw.Close(); err != nil {
			logger.Log.Error("Failed to close gzip writer", zap.Error(err))
			return err
		}
		bodyReader = bytes.NewReader(buf.Bytes())
	}
	serverAddr := a.serverAddr
	if !strings.Contains(serverAddr, "://") {
		serverAddr = "http://" + serverAddr
	}
	url := serverAddr + "/updates"
	req, err := http.NewRequest(http.MethodPost, url, bodyReader)
	if err != nil {
		logger.Log.Error("Failed to create HTTP request", zap.String("url", url), zap.Error(err))
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept-Encoding", "gzip")
	if a.useGzip {
		req.Header.Set("Content-Encoding", "gzip")
	}
	if a.key != "" {
		hash := calculateHash(jsonData, a.key)
		req.Header.Set("HashSHA256", fmt.Sprintf("%x", hash))
	}
	resp, err := a.sendWithRetry(req)
	if err != nil {
		logger.Log.Error("HTTP request failed",
			zap.String("url", url),
			zap.Error(err))
		return err
	}
	defer resp.Body.Close()
	var respBody io.Reader = resp.Body
	if resp.Header.Get("Content-Encoding") == "gzip" {
		zr, err := gzip.NewReader(resp.Body)
		if err != nil {
			logger.Log.Error("Failed to create gzip reader", zap.Error(err))
			return fmt.Errorf("failed to create gzip reader: %w", err)
		}
		defer zr.Close()
		respBody = zr
	}
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(respBody)
		logger.Log.Error("Server returned error",
			zap.Int("status_code", resp.StatusCode),
			zap.String("response_body", string(b)))
		return fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(b))
	}
	return nil
}

func (a *Agent) sendMetric(metricType, name string, value interface{}) error {
	var reqBody models.Metrics
	reqBody.ID = name
	reqBody.MType = metricType

	switch metricType {
	case "gauge":
		if v, ok := value.(float64); ok {
			reqBody.Value = &v
		} else {
			logger.Log.Error("Invalid gauge value type",
				zap.String("name", name),
				zap.Any("value", value))
			return fmt.Errorf("invalid gauge value type: %T", value)
		}
	case "counter":
		if v, ok := value.(int64); ok {
			reqBody.Delta = &v
		} else {
			logger.Log.Error("Invalid counter value type",
				zap.String("name", name),
				zap.Any("value", value))
			return fmt.Errorf("invalid counter value type: %T", value)
		}
	default:
		logger.Log.Error("Unsupported metric type",
			zap.String("type", metricType),
			zap.String("name", name))
		return fmt.Errorf("unsupported metric type: %s", metricType)

	}
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		logger.Log.Error("Failed to marshal metric",
			zap.String("type", metricType),
			zap.String("name", name),
			zap.Error(err))
		return err
	}
	bodyReader := bytes.NewReader(jsonData)
	if a.useGzip {
		var buf bytes.Buffer
		zw := gzip.NewWriter(&buf)
		if _, err := zw.Write(jsonData); err != nil {
			logger.Log.Error("Failed to compress metric data",
				zap.String("type", metricType),
				zap.String("name", name),
				zap.Error(err))
			return err
		}
		if err := zw.Close(); err != nil {
			logger.Log.Error("Failed to close gzip writer",
				zap.Error(err))
			return err
		}
		bodyReader = bytes.NewReader(buf.Bytes())
	}
	serverAddr := a.serverAddr
	if !strings.Contains(serverAddr, "://") {
		serverAddr = "http://" + serverAddr
	}
	url := serverAddr + "/update"
	req, err := http.NewRequest(http.MethodPost, url, bodyReader)
	if err != nil {
		logger.Log.Error("Failed to create HTTP request",
			zap.String("url", url),
			zap.Error(err))
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept-Encoding", "gzip")
	if a.useGzip {
		req.Header.Set("Content-Encoding", "gzip")
	}
	if a.key != "" {
		hash := calculateHash(jsonData, a.key)
		req.Header.Set("HashSHA256", fmt.Sprintf("%x", hash))
	}
	resp, err := a.sendWithRetry(req)
	if err != nil {
		logger.Log.Error("HTTP request failed",
			zap.String("url", url),
			zap.String("metric_type", metricType),
			zap.String("metric_name", name),
			zap.Error(err))
		return err
	}
	defer resp.Body.Close()
	var respBody io.Reader = resp.Body
	if resp.Header.Get("Content-Encoding") == "gzip" {
		zr, err := gzip.NewReader(resp.Body)
		if err != nil {
			logger.Log.Error("Failed to create gzip reader",
				zap.Error(err))
			return fmt.Errorf("failed to create gzip reader: %w", err)
		}
		defer zr.Close()
		respBody = zr
	}
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(respBody)
		logger.Log.Error("Server returned error",
			zap.Int("status_code", resp.StatusCode),
			zap.String("response_body", string(b)),
			zap.String("metric_type", metricType),
			zap.String("metric_name", name))
		return fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(b))
	}
	return nil
}
