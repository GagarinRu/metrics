package main

import (
	"compress/gzip"
	"io"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestShouldCompress(t *testing.T) {
	require.True(t, shouldCompress("application/json"))
	require.True(t, shouldCompress("text/html"))
	require.False(t, shouldCompress(""))
	require.False(t, shouldCompress("image/png"))
}

func TestCompressWriterRoundTrip(t *testing.T) {
	rec := httptest.NewRecorder()
	cw := newCompressWriter(rec)
	cw.Header().Set("Content-Type", "text/plain")
	_, err := cw.Write([]byte("metrics"))
	require.NoError(t, err)
	require.NoError(t, cw.Close())

	zr, err := gzip.NewReader(rec.Body)
	require.NoError(t, err)
	defer func() { _ = zr.Close() }()
	body, err := io.ReadAll(zr)
	require.NoError(t, err)
	require.Equal(t, "metrics", string(body))
}
