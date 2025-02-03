#!/bin/bash -eux

cwd=$(pwd)

pushd $cwd/dp-image-api
  make test-component
popd