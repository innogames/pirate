package pirate

import (
	"strconv"
	"time"
)

type Message struct {
	Header  map[string][]byte
	Metrics []*Metric
}

type Metric struct {
	Name      []byte
	Value     []byte
	Timestamp []byte
}

func NewMetric(name string, value float32, timestamp time.Time) *Metric {
	return &Metric{
		[]byte(name),
		strconv.AppendFloat(nil, float64(value), 'g', -1, 32),
		strconv.AppendInt(nil, timestamp.Unix(), 10),
	}
}
