# OGE Nomad Plugin

This is a Nomad task driver plugin for Open Grid Engine (OGE) that allows Nomad to manage and monitor jobs on an OGE high-performance computing cluster.

## Features

- Submit jobs to OGE from Nomad
- Monitor job status
- Terminate jobs
- Support for OGE-specific parameters:
  - Queue selection
  - Parallel environments
  - Resource requests
  - Project assignment

## Building the Plugin

To build the plugin, you need Go installed:

```bash
go build -o nomad-oge-driver
```

## Installation

Copy the built plugin to the Nomad plugins directory:

```bash
cp nomad-oge-driver /path/to/nomad/plugins/
```

## Configuration

### Plugin Configuration

In your Nomad client configuration:

```hcl
plugin "oge-task-driver" {
  config {
    qsub_path = "/path/to/qsub"
    qstat_path = "/path/to/qstat"
    qdel_path = "/path/to/qdel"
    default_queue = "all.q"
  }
}
```

### Job Configuration

Example job specification:

```hcl
job "oge-example" {
  datacenters = ["dc1"]
  type = "batch"

  group "example" {
    task "oge-task" {
      driver = "oge-task-driver"

      config {
        script = <<EOT
#!/bin/bash
echo "Hello from OGE!"
sleep 60
echo "Done!"
EOT
        queue = "all.q"
        pe = "smp"
        slots = 4
        resources = "h_vmem=4G"
        project = "myproject"
      }

      resources {
        cpu    = 100
        memory = 256
      }
    }
  }
}
```

## Task Configuration Parameters

| Parameter | Description | Required |
|-----------|-------------|----------|
| `script` | The script content to be executed | Yes |
| `args` | Arguments to pass to the script | No |
| `queue` | OGE queue to submit the job to | No (uses default_queue if not specified) |
| `pe` | Parallel environment to use | No |
| `slots` | Number of slots to request (for parallel jobs) | No (defaults to 1) |
| `resources` | Resource requests in OGE format | No |
| `project` | Project to associate the job with | No |

## Development

### Prerequisites

- Go 1.12 or later
- Nomad 1.0.0 or later
- Access to an OGE cluster

### Testing

To test the plugin:

1. Build the plugin
2. Configure Nomad to use the plugin
3. Submit a job using the plugin
4. Monitor the job status

## License

MIT 