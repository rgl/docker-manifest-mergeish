FROM golang:1.17.1-bullseye AS build
WORKDIR /build
COPY go.* ./
RUN go mod download
COPY *.go ./
RUN CGO_ENABLED=0 go build -ldflags="-s"

FROM debian:bullseye-slim
RUN apt-get update && \
    apt-get install -y --no-install-recommends ca-certificates && \
    rm -rf /var/lib/apt/lists/*
COPY --from=build /build/docker-manifest-mergeish /app/
ENTRYPOINT ["/app/docker-manifest-mergeish"]
