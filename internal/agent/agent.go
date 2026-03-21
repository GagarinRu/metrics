package agent

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
	"github.com/GagarinRu/metrics/internal/metrics"
	"github.com/GagarinRu/metrics/internal/models"
)

type Agent struct {
	metrics        *metrics.Metrics
	pollInterval   time.Duration
	reportInterval time.Duration
	serverAddr     string
	client         *http.Client
	useGzip        bool
}

type Config struct {
	PollInterval   time.Duration
	ReportInterval time.Duration
	ServerAddr     string
}

func NewAgent(cfg Config) *Agent {
	return &Agent{
		metrics:        metrics.NewMetrics(),
		pollInterval:   cfg.PollInterval,
		reportInterval: cfg.ReportInterval,
		serverAddr:     cfg.ServerAddr,
		client:         &http.Client{Timeout: 5 * time.Second},
		useGzip:        true,
	}
}

func (a *Agent) Run() error {
	done := make(chan error, 2)
	go func() {
		ticker := time.NewTicker(a.pollInterval)
		defer ticker.Stop()
		a.metrics.UpdateRuntimeMetrics()
		for range ticker.C {
			a.metrics.UpdateRuntimeMetrics()
		}
	}()
	go func() {
		ticker := time.NewTicker(a.reportInterval)
		defer ticker.Stop()

		for range ticker.C {
			if err := a.sendAllMetrics(); err != nil {
				fmt.Printf("Error sending metrics: %v\n", err)
			}
		}
	}()
	<-done
	return nil
}

func (a *Agent) sendAllMetrics() error {
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

func (a *Agent) sendMetric(metricType, name string, value interface{}) error {
	var reqBody models.Metrics
	reqBody.ID = name
	reqBody.MType = metricType

	switch metricType {
	case "gauge":
		if v, ok := value.(float64); ok {
			reqBody.Value = &v
		} else {
			return fmt.Errorf("invalid gauge value type: %T", value)
		}
	case "counter":
		if v, ok := value.(int64); ok {
			reqBody.Delta = &v
		} else {
			return fmt.Errorf("invalid counter value type: %T", value)
		}
	default:
		return fmt.Errorf("unsupported metric type: %s", metricType)
	}
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return err
	}
	bodyReader := bytes.NewReader(jsonData)
	if a.useGzip {
		var buf bytes.Buffer
		zw := gzip.NewWriter(&buf)
		if _, err := zw.Write(jsonData); err != nil {
			return err
		}
		if err := zw.Close(); err != nil {
			return err
		}
		bodyReader = bytes.NewReader(buf.Bytes())
	}
	url := fmt.Sprintf("%s/update", a.serverAddr)
	req, err := http.NewRequest(http.MethodPost, url, bodyReader)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept-Encoding", "gzip")
	if a.useGzip {
		req.Header.Set("Content-Encoding", "gzip")
	}
	resp, err := a.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	var respBody io.Reader = resp.Body
	if resp.Header.Get("Content-Encoding") == "gzip" {
		zr, err := gzip.NewReader(resp.Body)
		if err != nil {
			return fmt.Errorf("failed to create gzip reader: %w", err)
		}
		defer zr.Close()
		respBody = zr
	}
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(respBody)
		return fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(b))
	}
	return nil
}
