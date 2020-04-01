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
	Prometheus Prometheus `yaml:"prometheus"`
	Server     Server     `yaml:"server"`
}

type Prometheus struct {
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
	if err != nil {
		return nil, err
	}

	ret.mergeProperties()
	err = ret.convertAllRawProperties()
	if err != nil {
		return nil, err
	}
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

func (m MetricType) String() string {
	switch m {
	case MetricTypeCounter:
		return "counter"
	case MetricTypeGauge:
		return "gauge"
	default:
		return "unknown"
	}
}

type Metric struct {
	Name          string                 `yaml:"name"`
	Description   string                 `yaml:"description"`
	Type          MetricType             `yaml:"type"` //actually a string in config
	Interval      time.Duration          `yaml:"interval"`
	Properties    MetricProperties       `yaml:"-"`
	RawProperties map[string]interface{} `yaml:"properties"`
	Modes         []MetricMode           `yaml:"modes"`
	Instances     []MetricInstance       `yaml:"instances"`
}

type rawMetric struct {
	Name          string                 `yaml:"name"`
	Description   string                 `yaml:"description"`
	Type          MetricType             `yaml:"type"` //actually a string in config
	Interval      int                    `yaml:"interval"`
	Properties    MetricProperties       `yaml:"-"`
	RawProperties map[string]interface{} `yaml:"properties"`
	Modes         []MetricMode           `yaml:"modes"`
	Instances     []MetricInstance       `yaml:"instances"`
}

func (m *Metric) UnmarshalYAML(unmarshal func(interface{}) error) error {
	raw := rawMetric{}
	err := unmarshal(&raw)
	if err != nil {
		return err
	}

	*m = Metric{
		Name:          raw.Name,
		Description:   raw.Description,
		Type:          raw.Type,
		Interval:      time.Duration(raw.Interval) * time.Second,
		Properties:    raw.Properties,
		RawProperties: raw.RawProperties,
		Modes:         raw.Modes,
		Instances:     raw.Instances,
	}

	return nil
}

type MetricProperties struct {
	// For Counter, this defines how much the average increase is
	// For Gauge, this defines the average value
	Avg    int `yaml:"avg"`
	Jitter int `yaml:"jitter"`
}

type ModeDefinition struct {
	Name string `yaml:"name"`
}

type MetricMode struct {
	Name          string                 `yaml:"name"`
	Properties    MetricProperties       `yaml:"-"`
	RawProperties map[string]interface{} `yaml:"properties"`
}

type GlobalProperties struct {
	Type          MetricType             `yaml:"type"`
	Properties    MetricProperties       `yaml:"-"`
	RawProperties map[string]interface{} `yaml:"properties"`
}

type MetricInstance struct {
	Labels        map[string]string      `yaml:"labels"`
	Properties    MetricProperties       `yaml:"-"`
	RawProperties map[string]interface{} `yaml:"properties"`
	Modes         []MetricMode
}

func (c *Config) mergeProperties() {
	for _, global := range c.Prometheus.GlobalProperties {
		for j, metric := range c.Prometheus.Metrics {
			if global.Type == metric.Type {
				for key, value := range global.RawProperties {
					if _, found := metric.RawProperties[key]; !found {
						c.Prometheus.Metrics[j].RawProperties[key] = value
					}
				}
			}

			for k, instance := range metric.Instances {
				for key, value := range metric.RawProperties {
					if _, found := instance.RawProperties[key]; !found {
						c.Prometheus.Metrics[j].Instances[k].RawProperties[key] = value
					}
				}

				for l, mode := range instance.Modes {
					for key, value := range instance.RawProperties {
						if _, found := mode.RawProperties[key]; !found {
							c.Prometheus.Metrics[j].Instances[k].Modes[l].RawProperties[key] = value
						}
					}
				}
			}
		}
	}
}

func (c *Config) mapToMetricProperties(raw map[string]interface{}) (MetricProperties, error) {
	ret := MetricProperties{}
	if value, found := raw["avg"]; found {
		valueInt, isInt := value.(int)
		if !isInt {
			return ret, fmt.Errorf("avg must be type int")
		}

		ret.Avg = valueInt
	}

	if value, found := raw["jitter"]; found {
		valueInt, isInt := value.(int)
		if !isInt {
			return ret, fmt.Errorf("jitter must be type int")
		}

		ret.Jitter = valueInt
	}

	return ret, nil
}

func (c *Config) convertAllRawProperties() error {
	var err error
	for i, prop := range c.Prometheus.GlobalProperties {
		c.Prometheus.GlobalProperties[i].Properties, err = c.mapToMetricProperties(prop.RawProperties)
		if err != nil {
			return fmt.Errorf("Error parsing global properties for type `%s': %s", prop.Type, err)
		}
	}

	for i, metric := range c.Prometheus.Metrics {
		c.Prometheus.Metrics[i].Properties, err = c.mapToMetricProperties(metric.RawProperties)
		if err != nil {
			return fmt.Errorf("Error parsing metric properties for metric `%s': %s", metric.Name, err)
		}

		for j, instance := range metric.Instances {
			c.Prometheus.Metrics[i].Instances[j].Properties, err = c.mapToMetricProperties(instance.RawProperties)
			if err != nil {
				return fmt.Errorf("Error parsing metric instance properties for instance `%d' of metric `%s': %s", j, metric.Name, err)
			}

			for k, mode := range instance.Modes {
				c.Prometheus.Metrics[i].Instances[j].Modes[k].Properties, err = c.mapToMetricProperties(mode.RawProperties)
				if err != nil {
					return fmt.Errorf("Error parsing metric mode properties for mode `%d' of instance `%d' of metric `%s': %s", k, j, metric.Name, err)
				}
			}
		}
	}

	return nil
}

type Server struct {
	Port uint16
}
