package agent

import (
	"errors"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestIsRetryableError(t *testing.T) {
	assert.False(t, isRetryableError(errors.New("permanent")))
	assert.True(t, isRetryableError(&net.OpError{Err: timeoutError{}}))
}

type timeoutError struct{}

func (timeoutError) Error() string   { return "timeout" }
func (timeoutError) Timeout() bool   { return true }
func (timeoutError) Temporary() bool { return false }

func TestCollectAllMetrics(t *testing.T) {
	a := NewAgent(Config{
		ServerAddr:     "http://localhost",
		PollInterval:   time.Second,
		ReportInterval: time.Second,
	})
	a.metrics.UpdateRuntimeMetrics()
	metrics := a.collectAllMetrics()
	assert.NotEmpty(t, metrics)
}

func TestSendBatchEmpty(t *testing.T) {
	a := NewAgent(Config{ServerAddr: "http://localhost"})
	assert.NoError(t, a.sendBatch(nil))
}

func TestCalculateHash(t *testing.T) {
	hash := calculateHash([]byte(`{"id":"x"}`), "secret")
	assert.NotEmpty(t, hash)
}
