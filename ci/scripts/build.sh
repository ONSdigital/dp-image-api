#!/bin/bash -eux

pushd dp-image-api
  make build
  cp build/dp-image-api Dockerfile.concourse ../build
popd
