#!/bin/bash
# Setup Cilium on a cluster. NOTE: It must not have a CNI already installed.

# <============== Install Cilium ==============>
CILIUM_CLI_VERSION=v0.15.20
CLI_ARCH=amd64
if [ "$(uname -m)" = "aarch64" ]; then CLI_ARCH=arm64; fi
curl -L --fail --remote-name-all https://github.com/cilium/cilium-cli/releases/download/${CILIUM_CLI_VERSION}/cilium-linux-${CLI_ARCH}.tar.gz{,.sha256sum}
sha256sum --check cilium-linux-${CLI_ARCH}.tar.gz.sha256sum
sudo tar xzvfC cilium-linux-${CLI_ARCH}.tar.gz /usr/local/bin
rm cilium-linux-${CLI_ARCH}.tar.gz{,.sha256sum}

helm repo add cilium https://helm.cilium.io/
helm install cilium cilium/cilium --version 1.14.6 --namespace kube-system \
                                  --set loadBalancer.l7.backend=envoy \
                                  --set envoyConfig.enabled=true \
                                  --set kubeProxyReplacement=true \
                                  --set ingressController.enabled=true \
                                  --set ingressController.loadbalancerMode=dedicated

sleep 10

# Wait for cilium pods to be ready
cilium status --wait
