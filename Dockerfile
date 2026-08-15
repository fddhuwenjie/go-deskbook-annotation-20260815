FROM golang:1.26.5-bookworm AS build

WORKDIR /src
COPY . .
ENV GOTOOLCHAIN=local CGO_ENABLED=0
RUN go test ./... -count=1 && go build -o /out/deskbook ./cmd/deskbook

FROM debian:bookworm-slim
COPY --from=build /out/deskbook /usr/local/bin/deskbook
ENTRYPOINT ["deskbook"]
