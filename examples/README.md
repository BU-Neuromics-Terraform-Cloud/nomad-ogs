# OGE Nomad Plugin Environment

This directory contains scripts and configuration files to set up and run Nomad with the OGE plugin in combined mode.

## Quick Start

To set up and run the complete environment:

```bash
./run_nomad_oge.sh
```

This script will:
1. Install Nomad (if not already installed)
2. Build the OGE plugin (if not already built)
3. Set up the OGE plugin for Nomad
4. Start Nomad in combined mode (both server and client in the same process)

## Running OGE Jobs

Once Nomad is running, you can submit OGE jobs using:

```bash
# Submit a basic OGE job
./run_oge_job.sh basic

# Submit an MPI job
./run_oge_job.sh mpi
```

## Configuration Files

- `nomad-combined-config.hcl`: Combined server and client configuration with OGE plugin settings

## Job Files

- `oge-job.nomad`: A simple OGE job that runs a bash script
- `oge-mpi-job.nomad`: An example MPI job for OGE

## Customization

You may need to customize the following:

1. In the configuration file, update the paths to OGE commands (`qsub_path`, `qstat_path`, `qdel_path`) to match your environment.
2. In the job files, update the OGE-specific configurations (`queue`, `pe`, `slots`, `resources`, `project`) to match your OGE cluster setup.

## Troubleshooting

- If Nomad fails to start, check the logs for errors.
- Make sure the OGE commands (qsub, qstat, qdel) are available in the specified paths.
- Verify that the OGE plugin is correctly built and copied to the plugins directory.
- Ensure that your OGE cluster is properly configured and accessible.

## Advanced Usage

### Nomad Commands

Once Nomad is running, you can use the Nomad CLI to interact with it:

```bash
# Check the status of Nomad
./nomad/nomad status

# List running jobs
./nomad/nomad job status

# View job details
./nomad/nomad job status oge-example

# View job logs
./nomad/nomad alloc logs -job oge-example

# Stop a job
./nomad/nomad job stop oge-example
```

### Accessing the Nomad UI

When Nomad is running, you can access the web UI at:

```
http://localhost:4646
``` 