#!/bin/bash
set -e

echo "Installing SPIRE for Ubuntu 24..."

# Configuration
SPIRE_VERSION="1.8.7"
INSTALL_DIR="/opt/spire"
SPIRE_URL="https://github.com/spiffe/spire/releases/download/v${SPIRE_VERSION}/spire-${SPIRE_VERSION}-linux-amd64-musl.tar.gz"


# Create installation directory
echo "Creating installation directory..."
sudo mkdir -p ${INSTALL_DIR}

# Download SPIRE
echo "Downloading SPIRE ${SPIRE_VERSION}..."
wget -q --show-progress ${SPIRE_URL} -O /tmp/spire.tar.gz

# Extract SPIRE
echo "Extracting SPIRE..."
sudo tar -xzf /tmp/spire.tar.gz -C ${INSTALL_DIR} --strip-components=1

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
    trust_bundle_path = "/opt/spire/data/bundle.crt"
    trust_domain = "example.org"
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