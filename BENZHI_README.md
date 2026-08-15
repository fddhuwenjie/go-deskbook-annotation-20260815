# Deskbook Go Evaluation Environment

This repository contains a small in-memory desk reservation service. The
evaluation image keeps the complete Go toolchain so that diagnosis,
reproduction, editing, compilation, and testing can all run in the container.

## Build

Build one image for each required architecture:

```bash
./build_benzhi_docker.sh deskbook-benzhi:amd64 linux/amd64
./build_benzhi_docker.sh deskbook-benzhi:arm64 linux/arm64
```

The build uses `benzhi.Dockerfile`, downloads module dependencies before the
source is copied, and runs `go build ./...` while creating the image.

## Run and verify

Open an interactive shell:

```bash
docker run --rm -it --platform linux/amd64 deskbook-benzhi:amd64
```

Run the standard verification commands inside the container:

```bash
docker run --rm --platform linux/amd64 deskbook-benzhi:amd64 bash -c \
  'go version && go test ./... -count=1 && go test -race ./... -count=1 && go vet ./... && go build ./...'
```

Run the same checks for arm64:

```bash
docker run --rm --platform linux/arm64 deskbook-benzhi:arm64 bash -c \
  'go version && go test ./... -count=1 && go test -race ./... -count=1 && go vet ./... && go build ./...'
```

After the image is built, `go build ./...` can also be checked with container
networking disabled:

```bash
docker run --rm --network none --platform linux/amd64 \
  deskbook-benzhi:amd64 go build ./...
```
