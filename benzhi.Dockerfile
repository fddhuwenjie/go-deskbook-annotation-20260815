FROM golang:1.26.5-bookworm

WORKDIR /app

# Resolve module dependencies while the image build is allowed network access.
COPY go.mod ./
RUN go mod download

COPY . .

# Warm the build cache and prove the complete repository compiles.
RUN go build ./...

CMD ["bash"]
