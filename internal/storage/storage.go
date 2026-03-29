package storage

import (
	"database/sql"
	"encoding/json"
	"log"
	"os"
	"sync"
	"time"
	"github.com/GagarinRu/metrics/internal/models"
	_ "github.com/lib/pq"
)

type Storage interface {
	UpdateGauge(name string, value float64)
	UpdateCounter(name string, delta int64)
	GetGauge(name string) (float64, bool)
	GetCounter(name string) (int64, bool)
	GetAllGauges() map[string]float64
	GetAllCounters() map[string]int64
	Ping() error
}

type MemStorage struct {
	metrics  map[string]models.Metrics
	filePath string
	interval int
	stopChan chan struct{}
	mu       sync.RWMutex
	db       *sql.DB
}

func NewMemStorage() *MemStorage {
	return NewMemStorageWithFile("", 0, false)
}

func NewMemStorageWithFile(filePath string, interval int, restore bool) *MemStorage {
	ms := &MemStorage{
		metrics:  make(map[string]models.Metrics),
		filePath: filePath,
		interval: interval,
		stopChan: make(chan struct{}),
	}
	if restore && filePath != "" {
		if err := ms.Load(); err != nil {
			log.Printf("error loading metrics from a file: %v", err)
		}
	}
	if interval > 0 && filePath != "" {
		go ms.runSaver()
	}
	return ms
}

func (ms *MemStorage) UpdateGauge(name string, value float64) {
	ms.mu.Lock()
	v := value
	ms.metrics[name] = models.Metrics{
		ID:    name,
		MType: "gauge",
		Value: &v,
	}
	ms.mu.Unlock()
	if ms.interval == 0 && ms.filePath != "" {
		if err := ms.Save(); err != nil {
			log.Printf("error saving metrics: %v", err)
		}
	}
}

func (ms *MemStorage) UpdateCounter(name string, delta int64) {
	ms.mu.Lock()
	if existing, ok := ms.metrics[name]; ok && existing.MType == "counter" {
		if existing.Delta == nil {
			existing.Delta = new(int64)
			*existing.Delta = delta
		} else {
			*existing.Delta += delta
		}
		ms.metrics[name] = existing
	} else {
		d := delta
		ms.metrics[name] = models.Metrics{
			ID:    name,
			MType: "counter",
			Delta: &d,
		}
	}
	ms.mu.Unlock()
	if ms.interval == 0 && ms.filePath != "" {
		if err := ms.Save(); err != nil {
			log.Printf("error saving metrics: %v", err)
		}
	}
}

func (ms *MemStorage) GetGauge(name string) (float64, bool) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()
	m, ok := ms.metrics[name]
	if !ok || m.MType != "gauge" || m.Value == nil {
		return 0, false
	}
	return *m.Value, true
}

func (ms *MemStorage) GetCounter(name string) (int64, bool) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()
	m, ok := ms.metrics[name]
	if !ok || m.MType != "counter" || m.Delta == nil {
		return 0, false
	}
	return *m.Delta, true
}

func (ms *MemStorage) GetAllGauges() map[string]float64 {
	ms.mu.RLock()
	defer ms.mu.RUnlock()
	result := make(map[string]float64)
	for name, m := range ms.metrics {
		if m.MType == "gauge" && m.Value != nil {
			result[name] = *m.Value
		}
	}
	return result
}

func (ms *MemStorage) GetAllCounters() map[string]int64 {
	ms.mu.RLock()
	defer ms.mu.RUnlock()
	result := make(map[string]int64)
	for name, m := range ms.metrics {
		if m.MType == "counter" && m.Delta != nil {
			result[name] = *m.Delta
		}
	}
	return result
}

func (ms *MemStorage) Save() error {
	ms.mu.RLock()
	metricsList := make([]models.Metrics, 0, len(ms.metrics))
	for _, m := range ms.metrics {
		metricsList = append(metricsList, m)
	}
	ms.mu.RUnlock()
	data, err := json.MarshalIndent(metricsList, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(ms.filePath, data, 0666)
}

func (ms *MemStorage) Load() error {
	data, err := os.ReadFile(ms.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	var metricsList []models.Metrics
	if err := json.Unmarshal(data, &metricsList); err != nil {
		return err
	}
	ms.mu.Lock()
	defer ms.mu.Unlock()
	for _, m := range metricsList {
		ms.metrics[m.ID] = m
	}
	return nil
}

func (ms *MemStorage) Stop() {
	close(ms.stopChan)
	if ms.filePath != "" {
		if err := ms.Save(); err != nil {
			log.Printf("error saving metrics when stopping: %v", err)
		}
	}
}

func (ms *MemStorage) runSaver() {
	ticker := time.NewTicker(time.Duration(ms.interval) * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if err := ms.Save(); err != nil {
				log.Printf("error in periodically saving metrics: %v", err)
			}
		case <-ms.stopChan:
			return
		}
	}
}

func (ms *MemStorage) Ping() error {
	if ms.db != nil {
		return ms.db.Ping()
	}
	return nil
}

func (ms *MemStorage) SetDB(db *sql.DB) {
	ms.db = db
}
