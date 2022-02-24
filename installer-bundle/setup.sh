#!/bin/sh

SERVICE_NAME=fake-opsverse-agent
SERVICE_FILE=/etc/systemd/system/${SERVICE_NAME}.service

# move executable and config to appropriate directories
mkdir -p /usr/local/bin/ /etc/opsverse
cp -f ./fake-opsverse-telemetry-agent /usr/local/bin/ 
cp -f ./fake-agent-config.yaml /etc/opsverse/
chmod +x /usr/local/bin/fake-opsverse-telemetry-agent

if [ -f ${SERVICE_FILE} ]; then
  systemctl stop ${SERVICE_NAME}.service
  systemctl disable ${SERVICE_NAME}.service
  cp -f ./${SERVICE_NAME}.service ${SERVICE_FILE}
else
  cp -f ./${SERVICE_NAME}.service ${SERVICE_FILE}
fi
 
systemctl enable ${SERVICE_NAME}.service
systemctl start ${SERVICE_NAME}.service
