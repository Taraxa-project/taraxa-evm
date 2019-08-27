#!/bin/bash

docker build -t tmp - <"$(dirname "$0")"/../dockerfiles/base.ubuntu.dockerfile &&
  docker run -it -v /workspace:/workspace tmp
