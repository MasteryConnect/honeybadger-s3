#!/bin/bash

minimal() {
  # Minimal docker container build
  # The -ldflags '-s' removes debug information making the binary smaller
  CGO_ENABLED=0 GOOS=linux go build -a -ldflags '-s' -installsuffix cgo -o ./bin/honeybadger-s3 .

  sudo docker build -t masteryconnect/honeybadger-s3:1.0 -f Dockerfile .
}

echo "Running minimal docker build"
minimal
