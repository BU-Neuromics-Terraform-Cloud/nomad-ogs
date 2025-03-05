package main

import (
	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/nomad/plugins"
	"oge"
)

func main() {
	// Create a logger
	logger := hclog.New(&hclog.LoggerOptions{
		Level:      hclog.Trace,
		JSONFormat: true,
		Name:       "oge-nomad",
	})

	// Create the plugin
	plugins.Serve(map[string]plugins.PluginFactory{
		"oge-task-driver": func(logger hclog.Logger) interface{} {
			return oge.NewOGEDriver(logger)
		},
	})
} 