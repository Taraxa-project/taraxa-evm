#!/bin/bash

docker build -t tmp - <"$(dirname "$0")"/Dockerfile && docker run -it -v "$1":/workspace tmp
