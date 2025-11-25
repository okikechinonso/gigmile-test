#!/bin/bash

# Build the application
echo "Building the application..."
go build -o bin/api ./cmd/api

echo "Build completed successfully!"
