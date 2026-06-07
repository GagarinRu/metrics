package audit

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPublisher_NotifyFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "audit.log")
	p := NewPublisher(path, "")
	require.NotNil(t, p)
	defer p.Close()

	p.Notify([]string{"Alloc", "HeapAlloc"}, "127.0.0.1")

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	content := string(data)
	assert.Contains(t, content, "Alloc")
	assert.Contains(t, content, "127.0.0.1")
}

func TestPublisher_NotifyWithoutObservers(t *testing.T) {
	p := NewPublisher("", "")
	require.NotNil(t, p)
	assert.NotPanics(t, func() { p.Notify([]string{"Alloc"}, "127.0.0.1") })
}

func TestPublisher_Close(t *testing.T) {
	path := filepath.Join(t.TempDir(), "audit.log")
	p := NewPublisher(path, "")
	require.NotNil(t, p)

	p.Notify([]string{"Alloc"}, "127.0.0.1")
	assert.NoError(t, p.Close())

	p.Notify([]string{"HeapAlloc"}, "127.0.0.1")
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Contains(t, string(data), "Alloc")
	assert.NotContains(t, string(data), "HeapAlloc")
}

func TestClientIP(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	assert.Equal(t, "192.168.1.1", ClientIP(req))

	req.RemoteAddr = "invalid"
	assert.Equal(t, "invalid", ClientIP(req))
}

func TestPublisher_NotifyURL(t *testing.T) {
	var received string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf := make([]byte, 256)
		n, _ := r.Body.Read(buf)
		received = string(buf[:n])
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	p := NewPublisher("", srv.URL)
	require.NotNil(t, p)
	defer p.Close()
	p.Notify([]string{"PollCount"}, "10.0.0.1")
	assert.True(t, strings.Contains(received, "PollCount"))
}
