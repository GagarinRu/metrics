package main

import (
    "flag"
    "fmt"
    "time"
    "github.com/GagarinRu/metrics/internal/agent"
)

func main() {
    var (
        pollInterval   int
        reportInterval int
        serverAddr     string
    )
    flag.IntVar(&pollInterval, "p", 2, "Poll interval in seconds")
    flag.IntVar(&reportInterval, "r", 10, "Report interval in seconds")
    flag.StringVar(&serverAddr, "a", "http://localhost:8080", "Server address")
    flag.Parse()
    cfg := agent.Config{
        PollInterval:   time.Duration(pollInterval) * time.Second,
        ReportInterval: time.Duration(reportInterval) * time.Second,
        ServerAddr:     serverAddr,
    }
    a := agent.NewAgent(cfg)
    fmt.Printf("Starting agent with poll interval %v and report interval %v\n", 
        cfg.PollInterval, cfg.ReportInterval)
    fmt.Printf("Sending metrics to %s\n", cfg.ServerAddr)
    if err := a.Run(); err != nil {
        panic(err)
    }
}
