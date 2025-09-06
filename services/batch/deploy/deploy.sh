#!/bin/bash
set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}Router Manager Batch Service Deployment Script${NC}"
echo "================================================"

# Check if running as root or with sudo
if [[ $EUID -ne 0 ]]; then
   echo -e "${RED}This script must be run as root or with sudo${NC}" 
   exit 1
fi

# Variables
BINARY_NAME="router-manager-batch"
BINARY_PATH="/usr/local/bin/${BINARY_NAME}"
CONFIG_DIR="/etc/router-manager-batch"
CONFIG_FILE="${CONFIG_DIR}/router-manager-batch.conf"
DOCKER_DIR="/opt/router-manager"
SYSTEMD_DIR="/etc/systemd/system"
SERVICE_USER="router-manager"

echo -e "${YELLOW}Step 1: Creating service user...${NC}"
if ! id "${SERVICE_USER}" &>/dev/null; then
    useradd -r -s /bin/false -d /nonexistent -c "Router Manager Service" ${SERVICE_USER}
    echo -e "${GREEN}User '${SERVICE_USER}' created${NC}"
else
    echo -e "${GREEN}User '${SERVICE_USER}' already exists${NC}"
fi

echo -e "${YELLOW}Step 2: Building Go binary...${NC}"
# Script is in services/batch/deploy/, so go up one level
cd "$(dirname "$0")/.."
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o ${BINARY_NAME} cmd/batch/main.go
echo -e "${GREEN}Binary built successfully${NC}"

echo -e "${YELLOW}Step 3: Installing binary...${NC}"
cp ${BINARY_NAME} ${BINARY_PATH}
chmod 755 ${BINARY_PATH}
chown root:root ${BINARY_PATH}
echo -e "${GREEN}Binary installed to ${BINARY_PATH}${NC}"

echo -e "${YELLOW}Step 4: Creating configuration directory...${NC}"
mkdir -p ${CONFIG_DIR}
if [ ! -f ${CONFIG_FILE} ]; then
    cp config/router-manager-batch.conf.example ${CONFIG_FILE}
    chmod 600 ${CONFIG_FILE}
    chown ${SERVICE_USER}:${SERVICE_USER} ${CONFIG_FILE}
    echo -e "${YELLOW}Configuration template copied to ${CONFIG_FILE}${NC}"
    echo -e "${RED}Please edit ${CONFIG_FILE} with your actual configuration${NC}"
else
    echo -e "${GREEN}Configuration file already exists at ${CONFIG_FILE}${NC}"
fi

echo -e "${YELLOW}Step 5: Setting up Docker environment...${NC}"
mkdir -p ${DOCKER_DIR}
cp ../../../docker-compose.production.yml ${DOCKER_DIR}/docker-compose.yml
chown -R ${SERVICE_USER}:${SERVICE_USER} ${DOCKER_DIR}
echo -e "${GREEN}Docker compose file installed to ${DOCKER_DIR}${NC}"

echo -e "${YELLOW}Step 6: Installing systemd units...${NC}"
cp systemd/router-manager-batch.service ${SYSTEMD_DIR}/
cp systemd/router-manager-batch.timer ${SYSTEMD_DIR}/
systemctl daemon-reload
echo -e "${GREEN}Systemd units installed${NC}"

echo -e "${YELLOW}Step 7: Setting up permissions...${NC}"
# Add service user to necessary groups
usermod -a -G docker ${SERVICE_USER} 2>/dev/null || true
# Grant capabilities for nftables management
setcap 'cap_net_admin,cap_net_raw+ep' ${BINARY_PATH}
echo -e "${GREEN}Permissions configured${NC}"

echo -e "${YELLOW}Step 8: Service status...${NC}"
echo "To enable and start the service:"
echo "  systemctl enable router-manager-batch.timer"
echo "  systemctl start router-manager-batch.timer"
echo ""
echo "To check status:"
echo "  systemctl status router-manager-batch.timer"
echo "  systemctl status router-manager-batch.service"
echo ""
echo "To manually run the batch:"
echo "  systemctl start router-manager-batch.service"
echo ""
echo "To view logs:"
echo "  journalctl -u router-manager-batch.service -f"

echo -e "${GREEN}Deployment completed!${NC}"
echo -e "${YELLOW}Remember to:${NC}"
echo "1. Edit ${CONFIG_FILE} with your configuration"
echo "2. Start PostgreSQL with: cd ${DOCKER_DIR} && docker-compose up -d"
echo "3. Enable and start the timer service"