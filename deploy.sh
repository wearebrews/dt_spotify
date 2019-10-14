#!/bin/bash

VERSION=$(git rev-parse HEAD)
NAMESPACE=dtspotify
make docker-push
kubectl apply -n $NAMESPACE -f kubernetes/deployment.yaml
kubectl set image -n $NAMESPACE deployment/dt-spotify dtspotify=wearebrews/dtspotify:$VERSION 

