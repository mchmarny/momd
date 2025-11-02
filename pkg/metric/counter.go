package metric

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type IncrementalCounter interface {
	Increment(val ...string)
}

type Counter struct {
	Name string
	Help string

	vec *prometheus.CounterVec
}

func (c *Counter) Increment(val ...string) {
	c.vec.WithLabelValues(val...).Inc()
}

func NewCounter(name, help string, labels ...string) IncrementalCounter {
	return NewCounterWithRegistry(prometheus.DefaultRegisterer, name, help, labels...)
}

func NewCounterWithRegistry(reg prometheus.Registerer, name, help string, labels ...string) IncrementalCounter {
	counter := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: name,
		Help: help,
	}, labels)

	reg.MustRegister(counter)

	return &Counter{
		Name: name,
		Help: help,
		vec:  counter,
	}
}

// GetHandler returns an HTTP handler for serving Prometheus metrics.
func GetHandler() http.Handler {
	return promhttp.Handler()
}

// GetHandlerForRegistry returns an HTTP handler for serving Prometheus metrics from a custom registry.
func GetHandlerForRegistry(reg prometheus.Gatherer) http.Handler {
	return promhttp.HandlerFor(reg, promhttp.HandlerOpts{})
}
