#!/bin/bash

# Production optimized startup script for VM
echo "Starting production optimized webapp for VM..."

# Set environment variables for maximum performance
export GOMAXPROCS=4
export GOGC=100
export GOMEMLIMIT=2GiB

# Set MySQL optimization variables
export MYSQL_OPTIMIZATION=1

# Disable debug logging for production performance
export DEBUG=false

# Disable tracing for production performance
export TRACE_ENABLED=false

# Set Go runtime optimizations
export GOGC=100
export GOMEMLIMIT=2GiB

# Optimize system settings
echo "Optimizing system settings..."
echo 'net.core.somaxconn = 65535' >> /etc/sysctl.conf
echo 'net.ipv4.tcp_max_syn_backlog = 65535' >> /etc/sysctl.conf
echo 'net.core.netdev_max_backlog = 5000' >> /etc/sysctl.conf
sysctl -p

# Start the application with optimized settings
echo "Starting optimized production application..."
docker-compose up --build -d

echo "Production optimized webapp started successfully!"
echo "Backend: https://$(hostname).ftt2508.dabaas.net/api/health"
echo "Frontend: https://$(hostname).ftt2508.dabaas.net/"
echo "Jaeger: https://$(hostname).ftt2508.dabaas.net/jaeger/"
