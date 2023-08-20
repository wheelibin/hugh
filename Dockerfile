# syntax=docker/dockerfile:1

FROM golang:latest

ENV TZ=Europe/London

ENV DEBIAN_FRONTEND=noninteractive
RUN apt-get update && apt-get install -y tzdata


# Set destination for COPY
WORKDIR /app

# Download Go modules
COPY go.mod go.sum ./
RUN go mod download

# Copy the code
COPY . ./
COPY ./cmd/hugh/config.yaml ./

# Build
RUN CGO_ENABLED=0 GOOS=linux go build -C cmd/hugh -o /hugh

# Run
CMD ["/hugh"]
