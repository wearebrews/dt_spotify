#!/bin/bash

VERSION = $(git rev-parse HEAD)
echo "$DOCKER_PASSWORD" | docker login -u "$DOCKER_USERNAME" --password-stdin
make docker-push

kubectl set image -n dtspotify deployment/dt-spotify dtspotify=wearebrews/dtspotify:$VERSION 

