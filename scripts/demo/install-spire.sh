#!/bin/bash
set -e

# Configuration
SPIRE_VERSION="1.11.3"  # Pinned to latest stable version
INSTALL_DIR="/opt/spire"
SPIRE_URL="https://github.com/spiffe/spire/releases/download/v${SPIRE_VERSION}/spire-${SPIRE_VERSION}-linux-amd64-musl.tar.gz"

# Parse command line arguments
FORCE_INSTALL=false
while [[ $# -gt 0 ]]; do
    case $1 in
        --force)
            FORCE_INSTALL=true
            shift
            ;;
        --version)
            SPIRE_VERSION="$2"
            SPIRE_URL="https://github.com/spiffe/spire/releases/download/v${SPIRE_VERSION}/spire-${SPIRE_VERSION}-linux-amd64-musl.tar.gz"
            shift 2
            ;;
        *)
            echo "Unknown option: $1"
            echo "Usage: $0 [--force] [--version VERSION]"
            exit 1
            ;;
    esac
done

# Check if SPIRE is already installed
if [ -f "/usr/local/bin/spire-server" ] && [ "$FORCE_INSTALL" = false ]; then
    INSTALLED_VERSION=$(spire-server version 2>/dev/null | grep "Server" | awk '{print $3}' | cut -d'v' -f2 || echo "unknown")
    
    if [ "$INSTALLED_VERSION" = "$SPIRE_VERSION" ]; then
        echo "SPIRE version $SPIRE_VERSION is already installed."
        echo "Use --force to reinstall or --version to specify a different version."
        exit 0
    else
        echo "SPIRE version $INSTALLED_VERSION is installed, but version $SPIRE_VERSION is requested."
        echo "Upgrading SPIRE..."
    fi
fi

echo "Installing SPIRE $SPIRE_VERSION for Ubuntu 24..."


# Create installation directory
echo "Creating installation directory..."
sudo mkdir -p ${INSTALL_DIR}

# Download SPIRE
echo "Downloading SPIRE ${SPIRE_VERSION}..."
if ! wget -q --show-progress ${SPIRE_URL} -O /tmp/spire.tar.gz; then
    echo "❌ Failed to download SPIRE from ${SPIRE_URL}"
    exit 1
fi

# Verify download
if [ ! -f "/tmp/spire.tar.gz" ] || [ ! -s "/tmp/spire.tar.gz" ]; then
    echo "❌ SPIRE download failed or file is empty"
    exit 1
fi

# Extract SPIRE
echo "Extracting SPIRE..."
if ! sudo tar -xzf /tmp/spire.tar.gz -C ${INSTALL_DIR} --strip-components=1; then
    echo "❌ Failed to extract SPIRE"
    exit 1
fi

# Verify binaries were extracted
if [ ! -f "${INSTALL_DIR}/bin/spire-server" ] || [ ! -f "${INSTALL_DIR}/bin/spire-agent" ]; then
    echo "❌ SPIRE binaries not found in ${INSTALL_DIR}/bin/"
    ls -la ${INSTALL_DIR}/bin/ 2>/dev/null || echo "Directory does not exist"
    exit 1
fi

# Create symlinks
echo "Creating symlinks..."
sudo ln -sf ${INSTALL_DIR}/bin/spire-server /usr/local/bin/spire-server
sudo ln -sf ${INSTALL_DIR}/bin/spire-agent /usr/local/bin/spire-agent

# Create directories for SPIRE
echo "Creating SPIRE directories..."
sudo mkdir -p /opt/spire/data
sudo mkdir -p /opt/spire/conf
sudo mkdir -p /tmp/spire-server/private
sudo mkdir -p /tmp/spire-agent/public

# Create SPIRE server configuration
echo "Creating SPIRE server configuration..."
cat <<EOF | sudo tee ${INSTALL_DIR}/conf/server.conf
server {
    bind_address = "127.0.0.1"
    bind_port = "8081"
    socket_path = "/tmp/spire-server/private/api.sock"
    trust_domain = "example.org"
    data_dir = "/opt/spire/data"
    log_level = "INFO"
    ca_ttl = "24h"
    default_x509_svid_ttl = "1h"
}

plugins {
    DataStore "sql" {
        plugin_data {
            database_type = "sqlite3"
            connection_string = "/opt/spire/data/datastore.sqlite3"
        }
    }

    NodeAttestor "join_token" {
        plugin_data {}
    }

    KeyManager "disk" {
        plugin_data {
            keys_path = "/opt/spire/data/keys.json"
        }
    }

    UpstreamAuthority "disk" {
        plugin_data {
            key_file_path = "/opt/spire/data/upstream_ca.key"
            cert_file_path = "/opt/spire/data/upstream_ca.crt"
        }
    }
}
EOF

# Create SPIRE agent configuration
echo "Creating SPIRE agent configuration..."
cat <<EOF | sudo tee ${INSTALL_DIR}/conf/agent.conf
agent {
    data_dir = "/opt/spire/data"
    log_level = "INFO"
    server_address = "127.0.0.1"
    server_port = "8081"
    socket_path = "/tmp/spire-agent/public/api.sock"
    trust_domain = "example.org"
    insecure_bootstrap = true
}

plugins {
    NodeAttestor "join_token" {
        plugin_data {}
    }

    KeyManager "disk" {
        plugin_data {
            directory = "/opt/spire/data"
        }
    }

    WorkloadAttestor "unix" {
        plugin_data {}
    }
}
EOF

# Create systemd service for SPIRE server
echo "Creating systemd service for SPIRE server..."
cat <<EOF | sudo tee /etc/systemd/system/spire-server.service
[Unit]
Description=SPIRE Server
After=network.target

[Service]
Type=simple
User=root
ExecStart=/usr/local/bin/spire-server run -config /opt/spire/conf/server.conf
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF

# Create systemd service for SPIRE agent
echo "Creating systemd service for SPIRE agent..."
cat <<EOF | sudo tee /etc/systemd/system/spire-agent.service
[Unit]
Description=SPIRE Agent
After=spire-server.service
Requires=spire-server.service

[Service]
Type=simple
User=root
ExecStart=/usr/local/bin/spire-agent run -config /opt/spire/conf/agent.conf
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF

# Set permissions
echo "Setting permissions..."
sudo chmod 755 ${INSTALL_DIR}/bin/*
sudo chmod 600 ${INSTALL_DIR}/conf/*.conf

# Reload systemd
echo "Reloading systemd daemon..."
sudo systemctl daemon-reload

# Clean up
rm -f /tmp/spire.tar.gz

echo "SPIRE installation completed successfully!"
echo ""
echo "To start SPIRE services, run:"
echo "  ./start-spire.sh"
echo ""
echo "SPIRE binaries installed to: ${INSTALL_DIR}"
echo "SPIRE server socket: /tmp/spire-server/private/api.sock"
echo "SPIRE agent socket: /tmp/spire-agent/public/api.sock"