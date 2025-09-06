#!/bin/bash
set -e

echo "Building Docker image for ARM64 with caching..."
# Use buildx with cache mount for faster builds
docker buildx build \
    --platform linux/arm64 \
    --cache-from type=registry,ref=julianone/pippaothy:buildcache \
    --cache-to type=registry,ref=julianone/pippaothy:buildcache,mode=max \
    -t julianone/pippaothy:latest \
    --push \
    .

echo "Restarting Kubernetes deployment..."
kubectl rollout restart deployment pippaothy

echo "Waiting for rollout to complete..."
kubectl rollout status deployment pippaothy --timeout=300s

echo "Deployment complete!"