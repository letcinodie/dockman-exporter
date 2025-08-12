FROM ubuntu:latest
COPY build/dockman-exporter /usr/bin/dockman-exporter 
RUN chmod +x /usr/bin/dockman-exporter 
EXPOSE 9910
ENTRYPOINT ["/usr/bin/dockman-exporter"]
