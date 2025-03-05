module oge-nomad

go 1.24.0

require (
	github.com/hashicorp/go-hclog v1.6.3
	github.com/hashicorp/nomad v1.9.6
	github.com/stretchr/testify v1.10.0
)

// Handle the module path conflict
replace github.com/armon/go-metrics => github.com/hashicorp/go-metrics v0.5.3