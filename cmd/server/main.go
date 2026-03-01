package main

import (
    "flag"
    "fmt"
    "net/http"
    "github.com/go-chi/chi/v5"
    "github.com/GagarinRu/metrics/internal/handler"
    "github.com/GagarinRu/metrics/internal/storage"
)

func main() {
    var addr string
    flag.StringVar(&addr, "a", ":8080", "server address")
    flag.Parse()
    store := storage.NewMemStorage()
    h := handler.NewHandler(store)
    r := chi.NewRouter()
    r.Get("/", h.GetAllMetrics)
    r.Get("/value/{metricType}/{metricName}", h.GetMetric)
    r.Post("/update/{metricType}/{metricName}/{metricValue}", h.UpdateMetrics)
    fmt.Printf("Server started on %s\n", addr)
    if err := http.ListenAndServe(addr, r); err != nil {
        panic(err)
    }
}
