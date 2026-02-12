package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// MetricsCollector interface
type MetricsCollector interface {
	IncrementCounter(name string, value int64, tags ...string)
	RecordGauge(name string, value float64, tags ...string)
	RecordDuration(name string, duration time.Duration, tags ...string)
	RecordHistogram(name string, value float64, tags ...string)
}

// PrometheusCollector implementation
type PrometheusCollector struct {
	counters   map[string]*prometheus.CounterVec
	gauges     map[string]*prometheus.GaugeVec
	histograms map[string]*prometheus.HistogramVec
}

func NewPrometheus() MetricsCollector {
	return &PrometheusCollector{
		counters:   make(map[string]*prometheus.CounterVec),
		gauges:     make(map[string]*prometheus.GaugeVec),
		histograms: make(map[string]*prometheus.HistogramVec),
	}
}

func (p *PrometheusCollector) IncrementCounter(name string, value int64, tags ...string) {
	counter := p.getOrCreateCounter(name, tags)
	labels := p.extractLabels(tags)
	counter.With(labels).Add(float64(value))
}

func (p *PrometheusCollector) RecordGauge(name string, value float64, tags ...string) {
	gauge := p.getOrCreateGauge(name, tags)
	labels := p.extractLabels(tags)
	gauge.With(labels).Set(value)
}

func (p *PrometheusCollector) RecordDuration(name string, duration time.Duration, tags ...string) {
	histogram := p.getOrCreateHistogram(name, tags)
	labels := p.extractLabels(tags)
	histogram.With(labels).Observe(duration.Seconds())
}

func (p *PrometheusCollector) RecordHistogram(name string, value float64, tags ...string) {
	histogram := p.getOrCreateHistogram(name, tags)
	labels := p.extractLabels(tags)
	histogram.With(labels).Observe(value)
}

func (p *PrometheusCollector) getOrCreateCounter(name string, tags []string) *prometheus.CounterVec {
	if counter, exists := p.counters[name]; exists {
		return counter
	}

	labelNames := p.extractLabelNames(tags)
	counter := promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "sendflix_" + name,
			Help: name,
		},
		labelNames,
	)

	p.counters[name] = counter
	return counter
}

func (p *PrometheusCollector) getOrCreateGauge(name string, tags []string) *prometheus.GaugeVec {
	if gauge, exists := p.gauges[name]; exists {
		return gauge
	}

	labelNames := p.extractLabelNames(tags)
	gauge := promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "sendflix_" + name,
			Help: name,
		},
		labelNames,
	)

	p.gauges[name] = gauge
	return gauge
}

func (p *PrometheusCollector) getOrCreateHistogram(name string, tags []string) *prometheus.HistogramVec {
	if histogram, exists := p.histograms[name]; exists {
		return histogram
	}

	labelNames := p.extractLabelNames(tags)
	histogram := promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "sendflix_" + name,
			Help:    name,
			Buckets: prometheus.DefBuckets,
		},
		labelNames,
	)

	p.histograms[name] = histogram
	return histogram
}

func (p *PrometheusCollector) extractLabels(tags []string) prometheus.Labels {
	labels := prometheus.Labels{}
	for i := 0; i < len(tags); i += 2 {
		if i+1 < len(tags) {
			labels[tags[i]] = tags[i+1]
		}
	}
	return labels
}

func (p *PrometheusCollector) extractLabelNames(tags []string) []string {
	names := []string{}
	for i := 0; i < len(tags); i += 2 {
		names = append(names, tags[i])
	}
	return names
}

// NoopCollector does nothing
type NoopCollector struct{}

func NewNoop() MetricsCollector { return &NoopCollector{} }

func (n *NoopCollector) IncrementCounter(name string, value int64, tags ...string)          {}
func (n *NoopCollector) RecordGauge(name string, value float64, tags ...string)             {}
func (n *NoopCollector) RecordDuration(name string, duration time.Duration, tags ...string) {}
func (n *NoopCollector) RecordHistogram(name string, value float64, tags ...string)         {}