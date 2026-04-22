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

type Agent struct {
	metrics        *metrics.Metrics
	pollInterval   time.Duration
	reportInterval time.Duration
	serverAddr     string
	client         *http.Client
	useGzip        bool
	useBatch       bool
	key            string
	ctx            context.Context
	cancel         context.CancelFunc
	wg             sync.WaitGroup
}

type Config struct {
	PollInterval   time.Duration
	ReportInterval time.Duration
	ServerAddr     string
	UseGzip        bool
	UseBatch       *bool
	Key            string
}

func NewAgent(cfg Config) *Agent {
	ctx, cancel := context.WithCancel(context.Background())
	useBatch := true
	if cfg.UseBatch != nil {
		useBatch = *cfg.UseBatch
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
		ctx:            ctx,
		cancel:         cancel,
	}
}

func calculateHash(data []byte, key string) []byte {
	hash := sha256.Sum256(append(data, []byte(key)...))
	return hash[:]
}

func (a *Agent) Run(ctx context.Context) error {
	a.wg.Add(1)
	go func() {
		defer a.wg.Done()
		ticker := time.NewTicker(a.pollInterval)
		defer ticker.Stop()
		a.metrics.UpdateRuntimeMetrics()
		for {
			select {
			case <-ticker.C:
				a.metrics.UpdateRuntimeMetrics()
			case <-ctx.Done():
				logger.Log.Info("Stopping metrics collection goroutine")
				return
			}
		}
	}()
	a.wg.Add(1)
	go func() {
		defer a.wg.Done()
		ticker := time.NewTicker(a.reportInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				a.sendAllMetrics()
			case <-ctx.Done():
				logger.Log.Info("Sending final metrics before shutdown")
				a.sendAllMetrics()
				return
			}
		}
	}()
	<-ctx.Done()
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

func (a *Agent) sendAllMetrics() error {
	if a.useBatch {
		var metrics []models.Metrics
		for name, value := range a.metrics.GetGauges() {
			v := value
			metrics = append(metrics, models.Metrics{
				ID:    name,
				MType: "gauge",
				Value: &v,
			})
		}
		for name, value := range a.metrics.GetCounters() {
			d := value
			metrics = append(metrics, models.Metrics{
				ID:    name,
				MType: "counter",
				Delta: &d,
			})
		}
		if len(metrics) == 0 {
			return nil
		}
		return a.sendBatch(metrics)
	}
	for name, value := range a.metrics.GetGauges() {
		if err := a.sendMetric("gauge", name, value); err != nil {
			return fmt.Errorf("failed to send gauge metric %s: %w", name, err)
		}
	}
	for name, value := range a.metrics.GetCounters() {
		if err := a.sendMetric("counter", name, value); err != nil {
			return fmt.Errorf("failed to send counter metric %s: %w", name, err)
		}
	}
	return nil
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
