package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v2"
)

const (
	//MetricTypeCounter denotes a Prometheus Counter Metric
	MetricTypeCounter MetricType = iota
	//MetricTypeGauge denotes a Prometheus Gauge Metric
	MetricTypeGauge
)

type Config struct {
	Namespace        string             `yaml:"namespace"`
	GlobalProperties []GlobalProperties `yaml:"global_properties"`
	Metrics          []Metric           `yaml:"metrics"`
	Modes            []ModeDefinition   `yaml:"modes"`
}

func Load(filename string) (*Config, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}

	yamlDecoder := yaml.NewDecoder(file)
	ret := &Config{}
	err = yamlDecoder.Decode(ret)
	return ret, err
}

type MetricType int

func (m *MetricType) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var rawString string
	err := unmarshal(&rawString)
	if err != nil {
		return err
	}

	rawString = strings.ToLower(rawString)
	switch rawString {
	case "counter":
		*m = MetricTypeCounter
	case "gauge":
		*m = MetricTypeGauge
	default:
		return fmt.Errorf("unsupported metric type `%s'", rawString)
	}

	return nil
}

type Metric struct {
	Name           string           `yaml:"name"`
	Type           MetricType       `yaml:"type"` //actually a string in config
	BaseProperties MetricProperties `yaml:"default_properties"`
	Modes          []MetricMode     `yaml:"modes"`
}

type MetricProperties struct {
	//Both
	// Interval defines how often it goes up for counter and
	// how often it jitters for gauge
	Interval time.Duration `yaml:"interval"`

	//For Counter
	IncreaseAvg    int `yaml:"increase_avg"`
	IncreaseJitter int `yaml:"increase_jitter"`

	//For Gauge
	Avg    int `yaml:"avg"`
	Jitter int `yaml:"jitter"`
}

type ModeDefinition struct {
	Name string `yaml:"name"`
}

type MetricMode struct {
	Name       string           `yaml:"name"`
	Disable    bool             `yaml:"disable"`
	Properties MetricProperties `yaml:"properties"`
}

type GlobalProperties struct {
	Type       string           `yaml:"type"`
	Properties MetricProperties `yaml:"properties"`
}
