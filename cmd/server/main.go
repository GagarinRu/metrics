package main

import (
    "fmt"
    "net/http"
    "github.com/GagarinRu/metrics/internal/handler" 
    "github.com/GagarinRu/metrics/internal/storage"
)

func main() {
    store := storage.NewMemStorage()
    handler := handler.NewHandler(store)
    mux := http.NewServeMux()
    mux.HandleFunc("/update/", handler.UpdateMetrics)
    addr := ":8080"
    fmt.Printf("Server started on %s\n", addr)
    if err := http.ListenAndServe(addr, mux); err != nil {
        panic(err)
    }
}
