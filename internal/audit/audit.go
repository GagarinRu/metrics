package audit

import (
	"bytes"
	"encoding/json"
	"net"
	"net/http"
	"os"
	"sync"
	"time"
)

type Event struct {
	TS        int64    `json:"ts"`
	Metrics   []string `json:"metrics"`
	IPAddress string   `json:"ip_address"`
}

type Observer interface {
	Update(event Event)
	ID() string
}

type Publisher struct {
	observers map[string]Observer
}

func NewPublisher(auditFile, auditURL string) *Publisher {
	p := &Publisher{observers: make(map[string]Observer)}
	if auditFile != "" {
		p.register(&fileObserver{path: auditFile})
	}
	if auditURL != "" {
		p.register(&urlObserver{url: auditURL})
	}
	if len(p.observers) == 0 {
		return nil
	}
	return p
}

func (p *Publisher) register(o Observer) {
	p.observers[o.ID()] = o
}

func (p *Publisher) Notify(metrics []string, ipAddress string) {
	event := Event{
		TS:        time.Now().Unix(),
		Metrics:   metrics,
		IPAddress: ipAddress,
	}
	for _, o := range p.observers {
		o.Update(event)
	}
}

type fileObserver struct {
	path string
	mu   sync.Mutex
}

func (f *fileObserver) ID() string {
	return "file:" + f.path
}

func (f *fileObserver) Update(event Event) {
	data, err := json.Marshal(event)
	if err != nil {
		return
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	file, err := os.OpenFile(f.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		return
	}
	defer file.Close()
	file.Write(data)
	file.Write([]byte("\n"))
}

type urlObserver struct {
	url string
}

func (u *urlObserver) ID() string {
	return "url:" + u.url
}

func (u *urlObserver) Update(event Event) {
	data, err := json.Marshal(event)
	if err != nil {
		return
	}
	http.Post(u.url, "application/json", bytes.NewReader(data))
}

func ClientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
