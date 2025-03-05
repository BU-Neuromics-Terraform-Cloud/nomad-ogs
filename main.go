package main

import (
	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/nomad/plugins"
	"github.com/BU-Neuromics-Terraform-Cloud/nomad-ogs/oge"
)

func main() {
	// Create the plugin
	plugins.Serve(func(logger hclog.Logger) interface{} {
		return oge.NewOGEDriver(logger)
	})
} 