#!/bin/bash

# Run tests with coverage
echo "Running tests..."
go test -v -race -coverprofile=coverage.out -covermode=atomic ./...

# Generate coverage report
echo "Generating coverage report..."
go tool cover -html=coverage.out -o coverage.html

echo "Coverage report generated: coverage.html"
echo "Opening coverage report..."
open coverage.html 2>/dev/null || xdg-open coverage.html 2>/dev/null || echo "Please open coverage.html manually"
