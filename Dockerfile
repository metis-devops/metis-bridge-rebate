# syntax=docker/dockerfile:1
FROM golang:1.23.6 AS builder
RUN apt update && apt install -y build-essential git
WORKDIR /app
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download
COPY . .
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build go install .

FROM gcr.io/distroless/base-debian12:latest
COPY --from=builder /go/bin/metis-bridge-rebate /usr/local/bin/
ENTRYPOINT [ "metis-bridge-rebate" ]
