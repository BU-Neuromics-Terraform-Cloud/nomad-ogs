job "oge-mpi-example" {
  datacenters = ["dc1"]
  type = "batch"

  group "mpi-example" {
    task "mpi-task" {
      driver = "oge-task-driver"

      config {
        script = <<EOT
#!/bin/bash
#
# MPI job example for OGE
#
echo "Starting MPI job at $(date)"
echo "Running on host: $(hostname)"

# Load MPI module if needed
module load openmpi

# Get the machine file from OGE
MACHINEFILE=$TMPDIR/machines
cat $PE_HOSTFILE | awk '{print $1":"$2}' > $MACHINEFILE

echo "Machine file contents:"
cat $MACHINEFILE

# Run MPI program
echo "Running MPI program with $(wc -l < $MACHINEFILE) processes"
mpirun -np $NSLOTS -machinefile $MACHINEFILE mpi_hello_world

echo "MPI job completed at $(date)"
EOT

        # OGE specific configurations for MPI
        queue = "all.q"
        pe = "mpi"  # Use the MPI parallel environment
        slots = 16  # Request 16 slots
        resources = "h_vmem=2G,h_rt=02:00:00"
        project = "hpc_project"
      }

      resources {
        cpu    = 1600  # 16 cores
        memory = 32768 # 32GB
      }

      logs {
        max_files     = 10
        max_file_size = 10
      }
    }
  }
} 