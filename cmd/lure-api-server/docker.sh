#!/bin/bash

CGO_ENABLED=0 GOARCH=amd64 GOOS=linux go build
docker buildx build --platform linux/amd64 --tag arsen6331/lure-api-server:amd64 .

CGO_ENABLED=0 GOARCH=arm64 GOOS=linux go build
docker buildx build --platform linux/arm64/v8 --tag arsen6331/lure-api-server:arm64 .

docker login
docker push arsen6331/lure-api-server -a

docker manifest rm arsen6331/lure-api-server:latest
docker manifest create arsen6331/lure-api-server:latest --amend arsen6331/lure-api-server:arm64 --amend arsen6331/lure-api-server:amd64
docker manifest push arsen6331/lure-api-server:latest
