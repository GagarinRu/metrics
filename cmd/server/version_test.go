package main

import (
	"bytes"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPrintBuildInfo(t *testing.T) {
	buildVersion = "2.0.0"
	buildDate = "2026-06-26"
	buildCommit = "def456"
	defer func() {
		buildVersion = ""
		buildDate = ""
		buildCommit = ""
	}()

	old := os.Stdout
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout = w

	printBuildInfo()

	require.NoError(t, w.Close())
	os.Stdout = old

	var buf bytes.Buffer
	_, err = io.Copy(&buf, r)
	require.NoError(t, err)
	require.Contains(t, buf.String(), "Build version: 2.0.0")
	require.Contains(t, buf.String(), "Build date: 2026-06-26")
	require.Contains(t, buf.String(), "Build commit: def456")
}

func TestValueOrNA(t *testing.T) {
	require.Equal(t, "N/A", valueOrNA(""))
	require.Equal(t, "v2", valueOrNA("v2"))
}
