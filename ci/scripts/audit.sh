#!/bin/bash -eux

export cwd=$(pwd)

pushd $cwd/dp-image-api
  make audit
popd  