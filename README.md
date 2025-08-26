# dockman-exporter
Playing around with Go, Prometheus, Docker and Podman

(Dockman is a portmanteau of Docker and Podman.)

**DISCLAIMER: I'm not a developer. I play around with code to learn. This is most likely bad code and not up to standard.**

# Build
```bash
go build -o build/dockman-exporter src/dockman-exporter.go
```

# Run (non containerized)
```bash
#Move binary to primary directory of executable commands on the system
sudo mv build/dockman-exporter /usr/bin/dockman-exporter

#Create a systemd service /etc/systemd/system/dockman-exporter.service
sudo tee /etc/systemd/system/dockman-exporter.service <<EOF
[Unit]
Description="Docker/Podman image size metric exporter"

[Service]
ExecStart=/usr/bin/dockman-exporter

[Install]
WantedBy=multi-user.target
EOF

#Enable and run service
sudo systemctl daemon-reload
sudo systemctl enable dockman-exporter
sudo systemctl start dockman-exporter
```

# Run (containerized)

``` bash
# Docker
export DATE=`date --iso-8601`
#Or use latest as tag
docker build -t dockman-exporter:$DATE .
docker run -d --name dockman-exporter --publish "9910:9910" -v /var/run/docker.sock:/var/run/docker.sock dockman-exporter:$DATE


# Podman
export DATE=`date --iso-8601`
#Or use latest as tag
podman build -t dockman-exporter:$DATE .
podman run -d --name dockman-exporter --publish "9910:9910" -v /var/run/user/1000/podman/podman.sock:/var/run/podman.sock dockman-exporter:$DATE

# Use docker-compose file
docker compose up -d
podman compose up -d 
```

# Test functionality
```bash
# This request should return a 200
curl -Ssq -I localhost:9910/metrics
```
