# syntax=docker/dockerfile:1

FROM golang:1 AS build-stage

WORKDIR /app

COPY go.mod go.sum ./
COPY ./cmd/. ./cmd/
RUN go mod download

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /docker-pulltainer -v ./cmd/pulltainer/main.go

FROM build-stage AS run-test-stage

FROM gcr.io/distroless/base-debian12 AS build-release-stage

WORKDIR /

COPY --from=build-stage /docker-pulltainer /docker-pulltainer

USER nonroot:nonroot

ENTRYPOINT ["/docker-pulltainer"]
