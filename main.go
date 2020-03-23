package main

import (
	"os"

	"github.com/thomasmitchell/demo-exporter/config"
	"github.com/thomasmitchell/demo-exporter/exporter"
	"github.com/thomasmitchell/demo-exporter/server"

	"github.com/jhunt/go-ansi"
	"github.com/jhunt/go-cli"
)

type Cmd struct {
	Config string `cli:"-c, --config"`
}

func main() {
	cmd := Cmd{
		Config: "./demo-exporter.yml",
	}
	_, _, err := cli.Parse(&cmd)
	if err != nil {
		bailWith("%s", err.Error())
	}

	cfg, err := config.Load(cmd.Config)
	if err != nil {
		bailWith("%s", err.Error())
	}

	exp := exporter.New(cfg.Prometheus.Namespace)
	for _, metric := range cfg.Prometheus.Metrics {
		err = exp.AddMetric(metric)
		if err != nil {
			bailWith("%s", err.Error())
		}
	}

	exp.StartScheduler()

	srv, err := server.New(cfg.Server)
	if err != nil {
		bailWith("%s", err)
	}

	err = srv.Listen()
	if err != nil {
		bailWith("%s", err)
	}
}

func bailWith(format string, args ...interface{}) {
	ansi.Fprintf(os.Stderr, "@R{FATAL:} "+format+"\n", args...)
	os.Exit(1)
}
