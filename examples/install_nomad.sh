#!/bin/bash

# Script to install HashiCorp Nomad in the examples directory
# This script will download the latest version of Nomad, extract it, and place it in the examples directory

set -e

# Define variables
EXAMPLES_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
INSTALL_DIR="${EXAMPLES_DIR}/nomad"
NOMAD_VERSION="1.9.6"  # You can change this to the version you want
OS="linux"
ARCH="amd64"

echo "Installing Nomad ${NOMAD_VERSION} in ${INSTALL_DIR}..."

# Create installation directory if it doesn't exist
mkdir -p "${INSTALL_DIR}"

# Download Nomad
DOWNLOAD_URL="https://releases.hashicorp.com/nomad/${NOMAD_VERSION}/nomad_${NOMAD_VERSION}_${OS}_${ARCH}.zip"
DOWNLOAD_PATH="${EXAMPLES_DIR}/nomad_${NOMAD_VERSION}_${OS}_${ARCH}.zip"

echo "Downloading Nomad from ${DOWNLOAD_URL}..."
curl -L -o "${DOWNLOAD_PATH}" "${DOWNLOAD_URL}"

# Extract Nomad
echo "Extracting Nomad..."
unzip -o "${DOWNLOAD_PATH}" -d "${INSTALL_DIR}"

# Clean up the zip file
echo "Cleaning up..."
rm "${DOWNLOAD_PATH}"

# Make Nomad executable
chmod +x "${INSTALL_DIR}/nomad"

# Create a basic Nomad configuration file
cat > "${INSTALL_DIR}/nomad.hcl" << EOF
# Basic Nomad configuration
data_dir = "${INSTALL_DIR}/data"
bind_addr = "0.0.0.0"

server {
  enabled = true
  bootstrap_expect = 1
}

client {
  enabled = true
}
EOF

# Create data directory
mkdir -p "${INSTALL_DIR}/data"

echo "Nomad ${NOMAD_VERSION} has been installed to ${INSTALL_DIR}"
echo "To start Nomad, run: ${INSTALL_DIR}/nomad agent -config=${INSTALL_DIR}/nomad.hcl"
echo "To verify the installation, run: ${INSTALL_DIR}/nomad version" 
