# Nomad client configuration with OGE plugin

# Client configuration
client {
  enabled = true
  
  # Configure the OGE plugin
  plugin "oge-task-driver" {
    config {
      # Paths to OGE commands
      qsub_path = "/opt/sge/bin/lx-amd64/qsub"
      qstat_path = "/opt/sge/bin/lx-amd64/qstat"
      qdel_path = "/opt/sge/bin/lx-amd64/qdel"
      
      # Default queue to use if not specified in the job
      default_queue = "all.q"
    }
  }
}

# Plugin directory where the OGE plugin binary should be placed
plugin_dir = "/opt/nomad/plugins"

# Server configuration (for reference)
server {
  enabled = true
  bootstrap_expect = 1
}

# Data directory
data_dir = "/opt/nomad/data"

# Log level
log_level = "INFO"

# Enable the UI
ui {
  enabled = true
}

# Advertise address
advertise {
  http = "{{ GetPrivateIP }}"
  rpc  = "{{ GetPrivateIP }}"
  serf = "{{ GetPrivateIP }}"
} 