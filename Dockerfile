# syntax=docker/dockerfile:1

# Build the application from source
FROM golang:1.24 AS build-stage

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY assets/ ./assets/
COPY config/ ./config/
COPY database/ ./database/
COPY server/ ./server/
COPY *.go ./

RUN CGO_ENABLED=0 GOOS=linux go build -o /serve

# Deploy the application binary into a lean image
FROM gcr.io/distroless/base-debian12 AS build-release

WORKDIR /

COPY --from=build-stage /serve /serve

EXPOSE 8000

USER nonroot:nonroot

ENTRYPOINT ["/serve"]
