package handler_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/GagarinRu/metrics/internal/handler"
	"github.com/GagarinRu/metrics/internal/models"
	"github.com/GagarinRu/metrics/internal/storage"
	"github.com/go-chi/chi/v5"
)

func exampleServer() *httptest.Server {
	store := storage.NewMemStorage()
	h := handler.NewHandler(store, "", nil)
	r := chi.NewRouter()
	r.Get("/", h.GetAllMetrics)
	r.Get("/value/{metricType}/{metricName}", h.GetMetric)
	r.Post("/update/{metricType}/{metricName}/{metricValue}", h.UpdateMetrics)
	r.Post("/update", h.UpdateMetricsJSON)
	r.Post("/updates", h.UpdateMetricsBatch)
	r.Post("/value", h.GetMetricJSON)
	return httptest.NewServer(r)
}

func ExampleHandler_UpdateMetrics() {
	srv := exampleServer()
	defer srv.Close()

	resp, err := http.Post(srv.URL+"/update/gauge/Alloc/42.5", "text/plain", nil)
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	defer func() { _ = resp.Body.Close() }()
	fmt.Println(resp.StatusCode)
}

func ExampleHandler_GetMetric() {
	srv := exampleServer()
	defer srv.Close()

	if postResp, _ := http.Post(srv.URL+"/update/gauge/Alloc/42.5", "text/plain", nil); postResp != nil {
		_ = postResp.Body.Close()
	}

	resp, err := http.Get(srv.URL + "/value/gauge/Alloc")
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	defer func() { _ = resp.Body.Close() }()
	body, _ := io.ReadAll(resp.Body)
	fmt.Println(string(body))
}

func ExampleHandler_UpdateMetricsJSON() {
	srv := exampleServer()
	defer srv.Close()

	payload := `{"id":"RandomValue","type":"gauge","value":0.75}`
	resp, err := http.Post(srv.URL+"/update", "application/json", strings.NewReader(payload))
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	defer func() { _ = resp.Body.Close() }()
	fmt.Println(resp.StatusCode)
}

func ExampleHandler_GetMetricJSON() {
	srv := exampleServer()
	defer srv.Close()

	if postResp, _ := http.Post(srv.URL+"/update/gauge/HeapAlloc/512", "text/plain", nil); postResp != nil {
		_ = postResp.Body.Close()
	}

	payload := `{"id":"HeapAlloc","type":"gauge"}`
	resp, err := http.Post(srv.URL+"/value", "application/json", strings.NewReader(payload))
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	defer func() { _ = resp.Body.Close() }()

	var m models.Metrics
	if err := json.NewDecoder(resp.Body).Decode(&m); err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Println(m.ID, *m.Value)
}

func ExampleHandler_UpdateMetricsBatch() {
	srv := exampleServer()
	defer srv.Close()

	gv := 1.1
	dl := int64(2)
	body, _ := json.Marshal([]models.Metrics{
		{ID: "g", MType: "gauge", Value: &gv},
		{ID: "c", MType: "counter", Delta: &dl},
	})
	resp, err := http.Post(srv.URL+"/updates", "application/json", bytes.NewReader(body))
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	defer func() { _ = resp.Body.Close() }()
	fmt.Println(resp.StatusCode)
}

func ExampleHandler_GetAllMetrics() {
	srv := exampleServer()
	defer srv.Close()

	if postResp, _ := http.Post(srv.URL+"/update/gauge/Alloc/100", "text/plain", nil); postResp != nil {
		_ = postResp.Body.Close()
	}

	resp, err := http.Get(srv.URL + "/")
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	defer func() { _ = resp.Body.Close() }()
	fmt.Println(resp.StatusCode)
}
