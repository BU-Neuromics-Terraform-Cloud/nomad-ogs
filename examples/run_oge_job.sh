#!/bin/bash

# Script to run OGE jobs using Nomad
# This script can be used to submit either the basic OGE job or the MPI job

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
NOMAD_BIN="${SCRIPT_DIR}/nomad/nomad"
BASIC_JOB_FILE="${SCRIPT_DIR}/oge-job.nomad"
MPI_JOB_FILE="${SCRIPT_DIR}/oge-mpi-job.nomad"

# Check if Nomad is installed
if [ ! -f "${NOMAD_BIN}" ]; then
  echo "Nomad binary not found at ${NOMAD_BIN}"
  echo "Please run ./run_nomad_oge.sh first to install and start Nomad"
  exit 1
fi

# Function to display usage
usage() {
  echo "Usage: $0 [basic|mpi]"
  echo "  basic - Submit the basic OGE job (default)"
  echo "  mpi   - Submit the MPI job"
  exit 1
}

# Parse command line arguments
JOB_TYPE="basic"
if [ $# -gt 0 ]; then
  case "$1" in
    basic) JOB_TYPE="basic" ;;
    mpi) JOB_TYPE="mpi" ;;
    *) usage ;;
  esac
fi

# Set the job file based on the job type
if [ "$JOB_TYPE" == "basic" ]; then
  JOB_FILE="${BASIC_JOB_FILE}"
  JOB_NAME="oge-example"
else
  JOB_FILE="${MPI_JOB_FILE}"
  JOB_NAME="oge-mpi-example"
fi

# Check if the job file exists
if [ ! -f "${JOB_FILE}" ]; then
  echo "Job file not found at ${JOB_FILE}"
  exit 1
fi

echo "Submitting OGE ${JOB_TYPE} job to Nomad..."
echo "Job file: ${JOB_FILE}"

# Run the job
${NOMAD_BIN} job run "${JOB_FILE}"

echo "Job submitted successfully"
echo "To check job status, run: ${NOMAD_BIN} job status ${JOB_NAME}"
echo "To see job logs, run: ${NOMAD_BIN} alloc logs -job ${JOB_NAME}"
echo "To stop the job, run: ${NOMAD_BIN} job stop ${JOB_NAME}" 