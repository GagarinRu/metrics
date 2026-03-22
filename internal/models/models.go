package models

import (
    "encoding/json"
    "fmt"
)

type Metrics struct {
    ID    string   `json:"id"`
    MType string   `json:"type"`
    Delta *int64   `json:"delta,omitempty"`
    Value *float64 `json:"value,omitempty"`
}

func (m Metrics) MarshalJSON() ([]byte, error) {
    switch m.MType {
    case "gauge":
        type alias struct {
            ID    string  `json:"id"`
            Type  string  `json:"type"`
            Value *float64 `json:"value,omitempty"`
        }
        return json.Marshal(alias{
            ID:    m.ID,
            Type:  "gauge",
            Value: m.Value,
        })
    case "counter":
        type alias struct {
            ID    string  `json:"id"`
            Type  string  `json:"type"`
            Delta *int64 `json:"delta,omitempty"`
        }
        return json.Marshal(alias{
            ID:    m.ID,
            Type:  "counter",
            Delta: m.Delta,
        })
    default:
        return nil, fmt.Errorf("unknown type of metric: %s", m.MType)
    }
}
