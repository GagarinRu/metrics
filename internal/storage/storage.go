package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/GagarinRu/metrics/internal/logger"
	"github.com/GagarinRu/metrics/internal/models"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"go.uber.org/zap"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"
)

type Storage interface {
	UpdateGauge(name string, value float64)
	UpdateCounter(name string, delta int64)
	UpdateBatch(metrics []models.Metrics) error
	GetGauge(name string) (float64, bool)
	GetCounter(name string) (int64, bool)
	GetAllGauges() map[string]float64
	GetAllCounters() map[string]int64
	Ping() error
	Close() error
}

type MemStorage struct {
	metrics    map[string]models.Metrics
	filePath   string
	interval   int
	stopChan   chan struct{}
	mu         sync.RWMutex
	db         *sql.DB
	useDB      bool
	memoryOnly bool
}

func NewMemStorage() *MemStorage {
	return NewMemStorageWithFile("", 0, false, "")
}

func NewMemStorageWithFile(filePath string, interval int, restore bool, dsn string) *MemStorage {
	ms := &MemStorage{
		metrics:    make(map[string]models.Metrics),
		filePath:   filePath,
		interval:   interval,
		stopChan:   make(chan struct{}),
		memoryOnly: true,
	}
	if dsn != "" {
		db, err := sql.Open("postgres", dsn)
		if err == nil {
			if err := ms.applyMigrations(db); err == nil {
				if err := db.Ping(); err == nil {
					ms.db = db
					ms.useDB = true
					ms.memoryOnly = false
					logger.Log.Info("Connected to PostgreSQL and applied migrations")
					if err := ms.loadFromDB(); err != nil {
						logger.Log.Error("Error loading metrics from DB", zap.Error(err))
					}
				} else {
					logger.Log.Error("Failed to ping database", zap.Error(err))
					db.Close()
				}
			} else {
				logger.Log.Error("Failed to apply migrations", zap.Error(err))
				db.Close()
			}
		} else {
			logger.Log.Error("Failed to open database", zap.Error(err))
		}
	}
	if ms.memoryOnly && filePath != "" {
		if restore {
			if err := ms.Load(); err != nil {
				logger.Log.Error("Error loading metrics from file", zap.Error(err))
			}
		}
		if interval > 0 {
			go ms.runSaver()
		}
	}
	return ms
}

func (ms *MemStorage) applyMigrations(db *sql.DB) error {
	wd, err := os.Getwd()
	if err != nil {
		return err
	}
	migrationsPath := filepath.ToSlash(wd) + "/migrations"
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return err
	}
	m, err := migrate.NewWithDatabaseInstance(
		"file://"+migrationsPath,
		"postgres",
		driver,
	)
	if err != nil {
		logger.Log.Error("Failed to create migrate instance", zap.Error(err))
		return err
	}
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		logger.Log.Error("Failed to apply migrations", zap.Error(err))
		return err
	}
	logger.Log.Info("Migrations applied successfully")
	return nil
}

func (ms *MemStorage) loadFromDB() error {
	if ms.db == nil {
		return errors.New("database not connected")
	}
	rows, err := ms.db.Query("SELECT id, type, value, delta FROM metrics")
	if err != nil {
		return err
	}
	defer rows.Close()
	ms.mu.Lock()
	defer ms.mu.Unlock()
	for rows.Next() {
		var m models.Metrics
		var value sql.NullFloat64
		var delta sql.NullInt64
		err := rows.Scan(&m.ID, &m.MType, &value, &delta)
		if err != nil {
			return err
		}
		if m.MType == "gauge" && value.Valid {
			v := value.Float64
			m.Value = &v
		} else if m.MType == "counter" && delta.Valid {
			m.Delta = &delta.Int64
		}
		ms.metrics[m.ID] = m
	}
	return rows.Err()
}

func (ms *MemStorage) UpdateGauge(name string, value float64) {
	ms.mu.Lock()
	ms.metrics[name] = models.Metrics{
		ID:    name,
		MType: "gauge",
		Value: &value,
	}
	ms.mu.Unlock()
	if ms.useDB && ms.db != nil {
		ms.saveGaugeToDB(name, value)
	} else if ms.interval == 0 && ms.filePath != "" && !ms.memoryOnly {
		if err := ms.Save(); err != nil {
			logger.Log.Error("error saving metrics", zap.Error(err))
		}
	}
}

func (ms *MemStorage) saveGaugeToDB(name string, value float64) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := ms.db.ExecContext(ctx,
		`INSERT INTO metrics (id, type, value, delta) VALUES ($1, 'gauge', $2, NULL)
		 ON CONFLICT (id) DO UPDATE SET value = $2, type = 'gauge'`,
		name, value)
	if err != nil {
		logger.Log.Error("Error saving gauge to DB", zap.Error(err))
	}
}

func (ms *MemStorage) saveCounterToDB(name string, delta int64) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := ms.db.ExecContext(ctx,
		`INSERT INTO metrics (id, type, value, delta) VALUES ($1, 'counter', NULL, $2)
		 ON CONFLICT (id) DO UPDATE SET delta = metrics.delta + $2, type = 'counter'`,
		name, delta)
	if err != nil {
		logger.Log.Error("Error saving counter to DB", zap.Error(err))
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
	if ms.useDB && ms.db != nil {
		ms.saveCounterToDB(name, delta)
	} else if ms.interval == 0 && ms.filePath != "" && !ms.memoryOnly {
		if err := ms.Save(); err != nil {
			logger.Log.Error("error saving metrics", zap.Error(err))
		}
	}
}

func (ms *MemStorage) UpdateBatch(metrics []models.Metrics) error {
	if len(metrics) == 0 {
		return nil
	}
	ms.mu.Lock()
	defer ms.mu.Unlock()
	if ms.useDB && ms.db != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		tx, err := ms.db.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		defer tx.Rollback()
		for _, m := range metrics {
			if m.MType == "gauge" && m.Value != nil {
				_, err := tx.ExecContext(ctx,
					`INSERT INTO metrics (id, type, value, delta) VALUES ($1, 'gauge', $2, NULL)
					 ON CONFLICT (id) DO UPDATE SET value = $2, type = 'gauge'`,
					m.ID, *m.Value)
				if err != nil {
					return err
				}
				ms.metrics[m.ID] = models.Metrics{
					ID:    m.ID,
					MType: "gauge",
					Value: m.Value,
				}
			} else if m.MType == "counter" && m.Delta != nil {
				_, err := tx.ExecContext(ctx,
					`INSERT INTO metrics (id, type, value, delta) VALUES ($1, 'counter', NULL, $2)
					 ON CONFLICT (id) DO UPDATE SET delta = metrics.delta + $2, type = 'counter'`,
					m.ID, *m.Delta)
				if err != nil {
					return err
				}
				existing, ok := ms.metrics[m.ID]
				if ok && existing.MType == "counter" && existing.Delta != nil {
					*existing.Delta += *m.Delta
					ms.metrics[m.ID] = existing
				} else {
					d := *m.Delta
					ms.metrics[m.ID] = models.Metrics{
						ID:    m.ID,
						MType: "counter",
						Delta: &d,
					}
				}
			}
		}
		return tx.Commit()
	}
	for _, m := range metrics {
		ms.metrics[m.ID] = m
	}
	if ms.interval == 0 && ms.filePath != "" && !ms.memoryOnly {
		return ms.Save()
	}
	return nil
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
	if !ms.useDB && ms.filePath != "" {
		if err := ms.Save(); err != nil {
			logger.Log.Error("error saving metrics when stopping", zap.Error(err))
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
				logger.Log.Error("error in periodically saving metrics", zap.Error(err))
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
	return errors.New("database not connected")
}

func (ms *MemStorage) Close() error {
	if ms.db != nil {
		return ms.db.Close()
	}
	return nil
}
