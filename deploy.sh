#!/bin/bash

VERSION=$(git rev-parse HEAD)
NAMESPACE=dtspotify
echo "$DOCKER_PASSWORD" | docker login -u "$DOCKER_USERNAME" --password-stdin
make docker-push
kubectl apply -n $NAMESPACE -f kubernetes/deployment.yaml
kubectl set image -n $NAMESPACE deployment/dt-spotify dtspotify=wearebrews/dtspotify:$VERSION 

