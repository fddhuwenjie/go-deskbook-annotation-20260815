#!/bin/sh
set -eu

image="${1:-deskbook:local}"
docker build --platform "${DOCKER_PLATFORM:-linux/amd64}" -t "$image" .
