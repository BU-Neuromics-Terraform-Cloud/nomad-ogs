#!/bin/bash

# Main script to set up and run Nomad with OGE plugin in combined mode
# This script handles installation, setup, and running of Nomad with the OGE plugin

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
NOMAD_BIN="${SCRIPT_DIR}/nomad/nomad"
CONFIG_FILE="${SCRIPT_DIR}/nomad-combined-config.hcl"
PLUGIN_DIR="${SCRIPT_DIR}/plugins"
DATA_DIR="${SCRIPT_DIR}/data"

echo "=== OGE Nomad Plugin Environment ==="
echo "This script will set up and run Nomad with the OGE plugin in combined mode."
echo ""

# Step 1: Install Nomad if needed
if [ ! -f "${NOMAD_BIN}" ]; then
  echo "=== Step 1: Installing Nomad ==="
  bash "${SCRIPT_DIR}/install_nomad.sh"
else
  echo "=== Step 1: Nomad already installed ==="
fi

# Step 2: Build and set up the OGE plugin
echo ""
echo "=== Step 2: Setting up the OGE plugin ==="
if [ ! -f "${REPO_ROOT}/nomad-oge-driver" ]; then
  echo "OGE plugin binary not found at ${REPO_ROOT}/nomad-oge-driver"
  echo "Building the OGE plugin..."
  cd "${REPO_ROOT}"
  go build -o nomad-oge-driver
  cd "${SCRIPT_DIR}"
fi

# Set up the OGE plugin
echo "Setting up the OGE plugin..."
mkdir -p "${PLUGIN_DIR}"
cp "${REPO_ROOT}/nomad-oge-driver" "${PLUGIN_DIR}/nomad-oge-driver"
chmod +x "${PLUGIN_DIR}/nomad-oge-driver"

# Step 3: Create data directory if it doesn't exist
mkdir -p "${DATA_DIR}"

# Step 4: Start Nomad in combined mode
echo ""
echo "=== Step 3: Starting Nomad (server+client) ==="
echo "Configuration file: ${CONFIG_FILE}"
echo "Plugin directory: ${PLUGIN_DIR}"
echo "Data directory: ${DATA_DIR}"
echo "Access the Nomad UI at http://localhost:4646"
echo "Press Ctrl+C to stop Nomad"

# Run Nomad in the foreground
${NOMAD_BIN} agent -config="${CONFIG_FILE}" 