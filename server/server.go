package server

import (
	"fmt"
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/thomasmitchell/demo-exporter/config"
)

type Server struct {
	port uint16
}

func New(conf config.Server) (*Server, error) {
	http.Handle("/metrics", promhttp.Handler())
	return &Server{port: conf.Port}, nil
}

func (s *Server) Listen() error {
	return http.ListenAndServe(fmt.Sprintf(":%d", s.port), nil)
}
