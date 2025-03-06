# Nomad configuration with both server and client in the same process

# Data directory
data_dir = "/usr3/graduate/labadorf/development/terraform/nomad-oge/examples/data"

# Log level
log_level = "INFO"

# Server configuration
server {
  enabled = true
  bootstrap_expect = 1
}

# Client configuration
client {
  enabled = true
  
  # Server connection (for client mode)
  servers = ["127.0.0.1:4647"]
}

# Plugin configuration for OGE
plugin "oge-task-driver" {
  config {
    # Paths to OGE commands
    qsub_path = "/usr/local/bin/qsub"
    qstat_path = "/usr/local/bin/qstat"
    qdel_path = "/usr/local/bin/qdel"
    
    # Default queue to use if not specified in the job
    default_queue = "all.q"
  }
}

# Plugin directory where the OGE plugin binary should be placed
plugin_dir = "/usr3/graduate/labadorf/development/terraform/nomad-oge/examples/plugins"

# Enable the UI
ui {
  enabled = true
}

# Network configuration
ports {
  http = 4646
  rpc = 4647
  serf = 4648
}

# Advertise configuration
advertise {
  # Defaults to the first private IP address.
  http = "127.0.0.1"
  rpc = "127.0.0.1"
  serf = "127.0.0.1"
} 