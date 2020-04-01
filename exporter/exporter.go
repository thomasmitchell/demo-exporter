package exporter

import (
	"fmt"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/thomasmitchell/demo-exporter/config"
)

const DefaultMode = 0

type Exporter struct {
	namespace string
	reg       *prometheus.Registry
	sched     scheduler
	modes     map[string]int
}

type scheduler struct {
	lock        sync.RWMutex
	groups      []timeGroup
	currentMode int
}

type timeGroup struct {
	every   time.Duration
	metrics []MetricModeSet
}

func New(namespace string, modes []string, reg *prometheus.Registry) *Exporter {
	retModes := map[string]int{}
	for i, mode := range modes {
		retModes[mode] = i + 1
	}
	return &Exporter{namespace: namespace, modes: retModes, reg: reg}
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
		modeSet := MetricModeSet{}

		metricToAdd, err := e.newMetric(col, metric.Type, metric.Properties, labels.orderedValues(instance.Labels))
		if err != nil {
			return err
		}

		modeSet.AddMetric(DefaultMode, metricToAdd)

		for _, mode := range instance.Modes {
			modeNum, err := e.modeNum(mode.Name)
			if err != nil {
				return err
			}

			metricToAdd, err := e.newMetric(col, metric.Type, mode.Properties, labels.orderedValues(instance.Labels))
			if err != nil {
				return err
			}

			modeSet.AddMetric(modeNum, metricToAdd)
		}

		e.getTimeGroup(metric.Interval).add(modeSet)
	}

	err := e.reg.Register(col)
	if err != nil {
		return err
	}

	return nil
}

func (e *Exporter) newMetric(
	col prometheus.Collector,
	typ config.MetricType,
	prop config.MetricProperties,
	labels []string,
) (Metric, error) {

	var ret Metric
	var err error
	switch typ {
	case config.MetricTypeCounter:
		ret = NewCounterMetric(
			col.(*prometheus.CounterVec).WithLabelValues(labels...),
			prop.Avg,
			prop.Jitter,
		)
	case config.MetricTypeGauge:
		ret = NewGaugeMetric(
			col.(*prometheus.GaugeVec).WithLabelValues(labels...),
			prop.Avg,
			prop.Jitter,
		)

	default:
		err = fmt.Errorf("Unknown metric type")
	}

	return ret, err
}

func (e *Exporter) modeNum(mode string) (int, error) {
	var err error
	ret, found := e.modes[mode]
	if !found {
		err = fmt.Errorf("Unknown mode `%s'", mode)
	}

	return ret, err
}

func (e *Exporter) StartScheduler() {
	e.sched.start()
}

func (e *Exporter) Gather() ([]*dto.MetricFamily, error) {
	return e.reg.Gather()
}

//Second return value is true if the current mode is the default
func (e *Exporter) GetMode() (string, bool) {
	modeInt := e.sched.getMode()
	if modeInt == 0 {
		return "default", true
	}

	for modeName, modeNum := range e.modes {
		if modeNum == modeInt {
			return modeName, false
		}
	}

	panic("Unknown mode found")
}

func (e *Exporter) SetMode(mode string) error {
	modeInt, err := e.modeNum(mode)
	if err != nil {
		return err
	}

	e.sched.lock.Lock()
	e.sched.currentMode = modeInt
	e.sched.lock.Unlock()
	return nil
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

func (t *timeGroup) add(m MetricModeSet) { t.metrics = append(t.metrics, m) }
func (t *timeGroup) performUpdates(mode int) {
	for i := range t.metrics {
		t.metrics[i].UpdateMetric(mode)
	}
}

func (s *scheduler) start() {
	//eventually should probably write a better scheduler for this, but we'll just run
	// goroutines for each group for now
	for _, group := range s.groups {
		go func(g timeGroup) {
			g.performUpdates(s.getMode())
			for range time.Tick(g.every) {
				g.performUpdates(s.getMode())
			}
		}(group)
	}
}

func (s *scheduler) getMode() int {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.currentMode
}
