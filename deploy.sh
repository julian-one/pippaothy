#!/bin/bash
set -e

echo "Building Docker image for ARM64..."
docker buildx build --platform linux/arm64 -t julianone/pippaothy:latest --load .

echo "Pushing image to registry..."
docker push julianone/pippaothy:latest

echo "Restarting Kubernetes deployment..."
kubectl rollout restart deployment pippaothy

echo "Deployment complete!"