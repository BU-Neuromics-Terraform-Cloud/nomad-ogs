job "oge-example" {
  datacenters = ["dc1"]
  type = "batch"

  group "example" {
    task "oge-task" {
      driver = "oge-task-driver"

      config {
        script = <<EOT
#!/bin/bash
#
# Simple OGE job example
#
echo "Starting job at $(date)"
echo "Running on host: $(hostname)"
echo "Current working directory: $(pwd)"

# Print environment
echo "Environment:"
env | sort

# Simulate some work
echo "Simulating work..."
for i in {1..10}; do
  echo "Step $i of 10"
  sleep 5
done

echo "Job completed at $(date)"
EOT

        # OGE specific configurations
        queue = "all.q"
        pe = "smp"
        slots = 4
        resources = "h_vmem=4G,h_rt=01:00:00"
        project = "myproject"
      }

      resources {
        cpu    = 400  # 4 cores
        memory = 4096 # 4GB
      }

      logs {
        max_files     = 10
        max_file_size = 10
      }
    }
  }
} 