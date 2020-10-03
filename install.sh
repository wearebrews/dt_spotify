#!/bin/bash

#Install kubectl
curl -LO https://storage.googleapis.com/kubernetes-release/release/$(curl -s https://storage.googleapis.com/kubernetes-release/release/stable.txt)/bin/linux/amd64/kubectl
chmod +x ./kubectl
sudo mv ./kubectl /usr/local/bin/kubectl

mkdir ${HOME}/.kube

if [ ! -f "$GCLOUD_PATH_APPLY" ]; then
  echo Installing gcloud SDK
  rm -rf "$GCLOUD_HOME"
  export CLOUDSDK_CORE_DISABLE_PROMPTS=1 # SDK installation is interactive, thus prompts must be disabled
  curl "https://sdk.cloud.google.com" | bash > /dev/null
fi





