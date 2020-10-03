#!/bin/bash

VERSION=$(git rev-parse HEAD)
NAMESPACE=dtspotify
echo "$DOCKER_PASSWORD" | docker login -u "$DOCKER_USERNAME" --password-stdin
make docker-push


GCLOUD_CREDENTIALS="$PWD/client-secret.json"
echo "$GCLOUD_KEY" > "${GCLOUD_CREDENTIALS}"

gcloud auth activate-service-account --key-file "${GCLOUD_CREDENTIALS}"
gcloud container clusters get-credentials brews-k8s --zone europe-north1-a --project wearebrews

kubectl apply -n $NAMESPACE -f kubernetes/
kubectl set image -n $NAMESPACE deployment/dt-spotify dtspotify=wearebrews/dtspotify:$VERSION 

