#!/bin/bash

# Optimized startup script for the webapp
echo "Starting optimized webapp..."

# Set environment variables for performance
export GOMAXPROCS=4
export GOGC=100
export GOMEMLIMIT=2GiB

# Set MySQL optimization variables
export MYSQL_OPTIMIZATION=1

# Disable debug logging for production performance
export DEBUG=false

# Set Go runtime optimizations
export GOGC=100
export GOMEMLIMIT=2GiB

# Start the application with optimized settings
cd backend && go run cmd/main.go
