# dockman-exporter
Playing around with Go, Prometheus, Docker and Podman

**DISCLAIMER: I'm not a developer. I play around with code to learn. This is most likely bad code and not up to standards.**

# Build
```bash
go build -o build/dockman-exporter src/dockman-exporter.go
```

# Usage (non containerized)
```bash
sudo mv build/dockman-exporter /usr/bin/dockman-exporter
#Create a systemd service /etc/systemd/system/dockman-exporter.service

[Unit]
Description="Docker/Podman image size metric exporter"

[Service]
ExecStart=/usr/bin/dockman-exporter

[Install]
WantedBy=multi-user.target

sudo systemctl daemon-reload
sudo systemctl enable dockman-exporter
sudo systemctl start dockman-exporter
```

# Usage (containerized)

``` bash
# Docker
export DATE=`date --iso-8601`
docker build -t dockman-exporter:$DATE .
docker run -d --name dockman-exporter --publish "9910:9910" -v /var/run/docker.sock:/var/run/docker.sock dockman-exporter


# Podman
export DATE=`date --iso-8601`
podman build -t dockman-exporter:$DATE .
podman run -d --name dockman-exporter --publish "9910:9910" -v /var/run/user/1000/podman/podman.sock:/var/run/podman.sock dockman-exporter:$DATE
```
