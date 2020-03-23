package exporter

import (
	"fmt"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/thomasmitchell/demo-exporter/config"
)

type Exporter struct {
	namespace string
	sched     scheduler
}

type scheduler struct {
	groups []timeGroup
}

type timeGroup struct {
	every   time.Duration
	metrics []Metric
}

func New(namespace string) *Exporter {
	return &Exporter{namespace: namespace}
}

func (e *Exporter) AddMetric(metric config.Metric) error {
	labels := e.getAllLabels(metric)
	var col prometheus.Collector
	promOpts := prometheus.Opts{
		Namespace: e.namespace,
		Name:      metric.Name,
		Help:      metric.Description,
	}
	switch metric.Type {
	case config.MetricTypeCounter:
		col = prometheus.NewCounterVec(prometheus.CounterOpts(promOpts), labels)
	case config.MetricTypeGauge:
		col = prometheus.NewGaugeVec(prometheus.GaugeOpts(promOpts), labels)
	}

	for _, instance := range metric.Instances {
		var metricToAdd Metric
		switch metric.Type {
		case config.MetricTypeCounter:
			metricToAdd = NewCounterMetric(
				col.(*prometheus.CounterVec).WithLabelValues(labels.orderedValues(instance.Labels)...),
				metric.Properties.Avg,
				metric.Properties.Jitter,
			)
		case config.MetricTypeGauge:
			metricToAdd = NewGaugeMetric(
				col.(*prometheus.GaugeVec).WithLabelValues(labels.orderedValues(instance.Labels)...),
				metric.Properties.Avg,
				metric.Properties.Jitter,
			)

		default:
			return fmt.Errorf("Unknown metric type")
		}

		e.getTimeGroup(metric.Properties.Interval).add(metricToAdd)
	}

	err := prometheus.Register(col)
	if err != nil {
		return err
	}

	return nil
}

func (e *Exporter) StartScheduler() {
	e.sched.start()
}

func (e *Exporter) getTimeGroup(t time.Duration) *timeGroup {
	for _, group := range e.sched.groups {
		if group.every == t {
			return &group
		}
	}

	return e.addTimeGroup(t)
}

func (e *Exporter) addTimeGroup(t time.Duration) *timeGroup {
	e.sched.groups = append(e.sched.groups, timeGroup{every: t})
	return &e.sched.groups[len(e.sched.groups)-1]
}

type labelList []string

func (l labelList) orderedValues(labels map[string]string) []string {
	ret := make([]string, len(l))
	if len(labels) > 0 {
		for i, label := range l {
			ret[i] = labels[label]
		}
	}

	return ret
}

func (e *Exporter) getAllLabels(metric config.Metric) labelList {
	allLabelNames := map[string]bool{}
	for _, instance := range metric.Instances {
		for k := range instance.Labels {
			allLabelNames[k] = true
		}
	}

	ret := []string{}
	for k := range allLabelNames {
		ret = append(ret, k)
	}

	return ret
}

func (t *timeGroup) add(m Metric) { t.metrics = append(t.metrics, m) }
func (t *timeGroup) performUpdates() {
	for i := range t.metrics {
		t.metrics[i].UpdateMetric()
	}
}

func (s *scheduler) start() {
	//eventually should probably write a better scheduler for this, but we'll just run
	// goroutines for each group for now
	for _, group := range s.groups {
		go func(g timeGroup) {
			g.performUpdates()
			for range time.Tick(g.every) {
				g.performUpdates()
			}
		}(group)
	}
}
