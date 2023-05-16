#!/bin/bash

CGO_ENABLED=0 GOARCH=amd64 GOOS=linux go build
docker buildx build --platform linux/amd64 --tag elara6331/lure-api-server:amd64 --no-cache .

CGO_ENABLED=0 GOARCH=arm64 GOOS=linux go build
docker buildx build --platform linux/arm64/v8 --tag elara6331/lure-api-server:arm64 --no-cache .

docker login
docker push elara6331/lure-api-server -a

docker manifest rm elara6331/lure-api-server:latest
docker manifest create elara6331/lure-api-server:latest --amend elara6331/lure-api-server:arm64 --amend elara6331/lure-api-server:amd64
docker manifest push elara6331/lure-api-server:latest
