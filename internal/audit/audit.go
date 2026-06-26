// Package audit publishes audit events to a file and over HTTP.
package audit

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"os"
	"sync"
	"time"
)

const urlObserverTimeout = 5 * time.Second

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
		if o, err := newFileObserver(auditFile); err == nil {
			p.register(o)
		}
	}
	if auditURL != "" {
		p.register(newURLObserver(auditURL))
	}
	return p
}

func (p *Publisher) register(o Observer) {
	p.observers[o.ID()] = o
}

func (p *Publisher) Notify(metrics []string, ipAddress string) {
	if len(p.observers) == 0 {
		return
	}
	event := Event{
		TS:        time.Now().Unix(),
		Metrics:   metrics,
		IPAddress: ipAddress,
	}
	for _, o := range p.observers {
		o.Update(event)
	}
}

func (p *Publisher) Close() error {
	var err error
	for _, o := range p.observers {
		if c, ok := o.(io.Closer); ok {
			err = errors.Join(err, c.Close())
		}
	}
	return err
}

type fileObserver struct {
	path string
	file *os.File
	mu   sync.Mutex
}

func newFileObserver(path string) (*fileObserver, error) {
	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		return nil, err
	}
	return &fileObserver{path: path, file: file}, nil
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
	if f.file == nil {
		return
	}
	_, _ = f.file.Write(data)
	_, _ = f.file.Write([]byte("\n"))
}

func (f *fileObserver) Close() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.file == nil {
		return nil
	}
	err := f.file.Close()
	f.file = nil
	return err
}

type urlObserver struct {
	url    string
	client *http.Client
}

func newURLObserver(url string) *urlObserver {
	return &urlObserver{
		url: url,
		client: &http.Client{
			Timeout: urlObserverTimeout,
		},
	}
}

func (u *urlObserver) ID() string {
	return "url:" + u.url
}

func (u *urlObserver) Update(event Event) {
	data, err := json.Marshal(event)
	if err != nil {
		return
	}
	resp, err := u.client.Post(u.url, "application/json", bytes.NewReader(data))
	if err != nil {
		return
	}
	defer func() { _ = resp.Body.Close() }()
}

func ClientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
