package main

import (
	"encoding/json"
	"os"

	"github.com/thomasmitchell/demo-exporter/config"

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

	//TODO: Replace
	err = json.NewEncoder(os.Stdout).Encode(cfg)
	if err != nil {
		bailWith("%s", err)
	}
}

func bailWith(format string, args ...interface{}) {
	ansi.Fprintf(os.Stderr, "@R{FATAL:} "+format+"\n", args...)
	os.Exit(1)
}
