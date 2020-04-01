package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/thomasmitchell/demo-exporter/config"
	"github.com/thomasmitchell/demo-exporter/exporter"
)

type Server struct {
	port uint16
}

func New(conf config.Server, exp *exporter.Exporter) (*Server, error) {
	http.Handle("/metrics", promhttp.HandlerFor(exp, promhttp.HandlerOpts{
		Timeout: 30 * time.Second,
	}))
	http.Handle("/mode", &modeHandler{exp: exp})

	return &Server{port: conf.Port}, nil
}

func (s *Server) Listen() error {
	return http.ListenAndServe(fmt.Sprintf(":%d", s.port), nil)
}

type apiError struct {
	Error string `json:"error"`
}

func respondError(w http.ResponseWriter, code int, f string, args ...interface{}) {
	respond(w, code, apiError{Error: fmt.Sprintf(f, args...)})
}

func respond(w http.ResponseWriter, code int, body interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	b, _ := json.Marshal(&body)
	w.Write(b)
}

type modeHandler struct {
	exp *exporter.Exporter
}

type modeRequest struct {
	Mode string `json:"mode"`
}

type getModeResponse struct {
	ModeName  string `json:"mode_name"`
	IsDefault bool   `json:"is_default"`
}

type setModeResponse struct {
	Message string `json:"message"`
}

func (h *modeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		h.getMode(w, r)
	case "POST", "PUT":
		h.setMode(w, r)
	default:
		http.NotFound(w, r)
	}
}

func (h *modeHandler) getMode(w http.ResponseWriter, r *http.Request) {
	mode, isDefault := h.exp.GetMode()
	respond(w, 200, &getModeResponse{ModeName: mode, IsDefault: isDefault})
}

func (h *modeHandler) setMode(w http.ResponseWriter, r *http.Request) {
	dec := json.NewDecoder(r.Body)
	bodyStruct := modeRequest{}
	dec.Decode(&bodyStruct)
	if bodyStruct.Mode == "" {
		respondError(w, 400, "`mode' field must be provided")
		return
	}

	err := h.exp.SetMode(bodyStruct.Mode)
	if err != nil {
		respondError(w, 400, err.Error())
		return
	}

	respond(w, 200, setModeResponse{Message: "success"})
}
