package server

import (
	"fmt"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/thomasmitchell/demo-exporter/config"
)

type Server struct {
	port     uint16
	gatherer prometheus.Gatherer
}

func New(conf config.Server, gatherer prometheus.Gatherer) (*Server, error) {
	http.Handle("/metrics", promhttp.HandlerFor(gatherer, promhttp.HandlerOpts{
		Timeout: 30 * time.Second,
	}))
	return &Server{port: conf.Port}, nil
}

func (s *Server) Listen() error {
	return http.ListenAndServe(fmt.Sprintf(":%d", s.port), nil)
}
