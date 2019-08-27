#!/bin/bash

docker build -t tmp - <"$(dirname "$0")"/Dockerfile && docker run -it -v /worskpace:/workspace tmp
