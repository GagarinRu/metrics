package agent

import (
    "fmt"
    "net/http"
    "strconv"
    "time"
    "github.com/GagarinRu/metrics/internal/metrics"
)

type Agent struct {
    metrics        *metrics.Metrics
    pollInterval   time.Duration
    reportInterval time.Duration
    serverAddr     string
    client         *http.Client
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
    var valueStr string
    switch v := value.(type) {
    case float64:
        valueStr = strconv.FormatFloat(v, 'f', -1, 64)
    case int64:
        valueStr = strconv.FormatInt(v, 10)
    default:
        return fmt.Errorf("unsupported metric type: %T", value)
    }
    url := fmt.Sprintf("%s/update/%s/%s/%s", a.serverAddr, metricType, name, valueStr)
    req, err := http.NewRequest(http.MethodPost, url, nil)
    if err != nil {
        return err
    }
    req.Header.Set("Content-Type", "text/plain")
    resp, err := a.client.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()
    if resp.StatusCode != http.StatusOK {
        return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
    }
    return nil
}
