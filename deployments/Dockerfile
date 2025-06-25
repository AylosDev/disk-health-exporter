# Build stage
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o disk-health-exporter ./cmd/disk-health-exporter

# Final stage
FROM ubuntu:22.04

# Install runtime dependencies
RUN apt-get update && \
  apt-get install -y \
  smartmontools \
  wget \
  ca-certificates \
  && rm -rf /var/lib/apt/lists/*

# Install MegaCLI
RUN wget -O /tmp/megacli.deb https://docs.broadcom.com/docs-and-downloads/raid-controllers/raid-controllers-common-files/8-07-14_MegaCLI.zip || true
# Note: You may need to manually download and install MegaCLI from Broadcom's website
# For now, we'll create a symbolic link that might work if megacli is available
RUN ln -sf /usr/sbin/megacli /usr/local/bin/megacli || true
RUN ln -sf /opt/MegaRAID/MegaCli/MegaCli64 /usr/local/bin/MegaCli64 || true

# Create non-root user
RUN groupadd -r exporter && useradd -r -g exporter exporter

# Copy the binary from builder stage
COPY --from=builder /app/disk-health-exporter /usr/local/bin/disk-health-exporter

# Make binary executable
RUN chmod +x /usr/local/bin/disk-health-exporter

# Expose port
EXPOSE 9100

# Set environment variables
ENV PORT=9100

# Health check
HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
  CMD curl -f http://localhost:9100/metrics || exit 1

# Note: This container needs to run with --privileged flag to access disk information
# or with specific capabilities and device mounts

# Switch to non-root user (commented out as it may need privileged access)
# USER exporter

# Run the exporter
CMD ["/usr/local/bin/disk-health-exporter"]
