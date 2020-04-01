package exporter

import (
	"math/rand"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

func init() {
	rand.Seed(time.Now().Unix())
}

type Metric interface {
	UpdateMetric()
}

type CounterMetric struct {
	avgIncrease int
	maxJitter   int
	metric      prometheus.Counter
}

func NewCounterMetric(m prometheus.Counter, avg, max int) *CounterMetric {
	return &CounterMetric{
		avgIncrease: avg,
		maxJitter:   max,
		metric:      m,
	}
}

func (m *CounterMetric) UpdateMetric() {
	jitter := rand.Intn(m.maxJitter*2+1) - m.maxJitter
	increaseBy := m.avgIncrease + jitter
	if increaseBy < 0 {
		increaseBy = 0
	}

	m.metric.Add(float64(increaseBy))
}

type GaugeMetric struct {
	avg       int
	maxJitter int
	metric    prometheus.Gauge
}

func NewGaugeMetric(m prometheus.Gauge, avg, max int) *GaugeMetric {
	return &GaugeMetric{
		avg:       avg,
		maxJitter: max,
		metric:    m,
	}
}

func (m *GaugeMetric) UpdateMetric() {
	jitter := rand.Intn(m.maxJitter*2+1) - m.maxJitter
	newValue := m.avg + jitter
	m.metric.Set(float64(newValue))
}

type MetricModeSet struct {
	//slot 0 is treated as a default
	modes []Metric
}

func (m *MetricModeSet) AddMetric(mode int, metric Metric) {
	for len(m.modes) <= mode {
		m.modes = append(m.modes, nil)
	}

	m.modes[mode] = metric
}

func (m *MetricModeSet) UpdateMetric(mode int) {
	var metric Metric
	if len(m.modes) > mode {
		metric = m.modes[mode]
	}
	if metric == nil {
		metric = m.modes[0]
		if metric == nil {
			return
		}
	}

	metric.UpdateMetric()
}
