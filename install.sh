#!/bin/bash

#Install kubectl
curl -LO https://storage.googleapis.com/kubernetes-release/release/$(curl -s https://storage.googleapis.com/kubernetes-release/release/stable.txt)/bin/linux/amd64/kubectl
chmod +x ./kubectl
sudo mv ./kubectl /usr/local/bin/kubectl

mkdir ${HOME}/.kube

#Install doctl
DOCTL_VERSION=1.17.0
curl -sL https://github.com/digitalocean/doctl/releases/download/v$DOCTL_VERSION/doctl-$DOCTL_VERSION-linux-amd64.tar.gz | tar -xzv
sudo mv doctl /usr/local/bin

#Requires a valid access token in env
doctl kubernetes cluster kubeconfig show brews-k8s > ${HOME}/.kube/config






