#!/bin/bash -eux

cwd=$(pwd)

pushd $cwd/dp-image-api
  go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.63.4
  make lint
popd
