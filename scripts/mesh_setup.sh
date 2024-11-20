#!/bin/bash
# Setup service mesh for the given argument
# Arguments:
# --mesh: service mesh name
# --ingress: whether to install an ingress controller

showHelp() {
cat << EOF  
Usage: <script_name> -m <mesh-name> [-i]
Attach bpf programs for a specific service.

-h, -help,      --help        Display help
-m, -mesh,      --mesh        Service mesh name (istio/linkerd/nginx/plain)
-i, -ingress,   --ingress     Whether to install an ingress controller

EOF
}

MESH=""
INGRESS=0

options=$(getopt -l "help,mesh:,ingress" -o "hm:i" -a -- "$@")

eval set -- "$options"

while true; do
  case "$1" in
  -h|--help) 
      showHelp
      exit 0
      ;;
  -m|--mesh)
      shift
      MESH=$1
      ;;
  -i|--ingress)
      INGRESS=1
      ;;
  --)
      shift
      break;;
  esac
  shift
done

sudo apt-get update

: "${TESTBED:=$HOME}"

pushd $TESTBED

# Write on a file to indicate that the mesh is installed
echo $MESH > mesh.txt

# Add and configure ingress if ingress is enabled
if [[ $INGRESS == 1 ]]; then
  kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/controller-v1.5.1/deploy/static/provider/cloud/deploy.yaml
  kubectl wait --namespace ingress-nginx \
    --for=condition=ready pod \
    --selector=app.kubernetes.io/component=controller \
    --timeout=120s
fi

if [[ $MESH == "istio" ]]; then
  # Install Istio
  curl -L https://istio.io/downloadIstio | ISTIO_VERSION=1.16.1 sh -
  pushd $TESTBED/istio-1.16.1
  ./bin/istioctl install --set profile=default -y
  popd

  kubectl label namespace default istio-injection=enabled --overwrite
elif [[ $MESH == "linkerd" ]]; then
  # Install Linkerd
  curl --proto '=https' -sSfL https://run.linkerd.io/install | sh  

  pushd $TESTBED/.linkerd2
  ./bin/linkerd install --crds | kubectl apply -f -
  ./bin/linkerd install | kubectl apply -f -
  popd
  
  # Wait for control plane to get running
  sleep 3m
elif [[ $MESH == "nginx" ]]; then
  # Install NGINX
  wget https://github.com/nginxinc/nginx-service-mesh/releases/download/v1.7.0/nginx-meshctl_1.7.0_linux_amd64.tar.gz
  tar -xvf nginx-meshctl_1.7.0_linux_amd64.tar.gz nginx-meshctl

  # Install the NGINX Service Mesh
  sudo chmod +x nginx-meshctl
  ./nginx-meshctl deploy

  # Wait for the service mesh to get running
  sleep 30

  rm nginx-meshctl_1.7.0_linux_amd64.tar.gz
elif [[ $MESH == "cilium" ]]; then
  cilium install --version 1.14.6 \
                 --set kubeProxyReplacement=true \
                 --set envoyConfig.enabled=true \
                 --set loadBalancer.l7.backend=envoy
  cilium status --wait
elif [[ $MESH == "hypo" ]]; then
  curl -L https://istio.io/downloadIstio | ISTIO_VERSION=1.16.1 sh -
  pushd $TESTBED/istio-1.16.1
  ./bin/istioctl install --set profile=default -y
  popd
  kubectl label namespace default istio-injection=disabled --overwrite

  # Wait for control plane to get running
  sleep 30
elif [[ $MESH == "wire" ]]; then
  cilium upgrade --version 1.14.6 \
                 --set kubeProxyReplacement=true \
                 --set envoyConfig.enabled=true \
                 --set loadBalancer.l7.backend=envoy
  cilium status --wait

  curl -L https://istio.io/downloadIstio | ISTIO_VERSION=1.16.1 sh -
  pushd $TESTBED/istio-1.16.1
  ./bin/istioctl install --set profile=default -y
  popd
  kubectl label namespace default istio-injection=disabled --overwrite

  # Wait for control plane to get running
  sleep 30

  # curl --proto '=https' -sSfL https://run.linkerd.io/install | sh
  # pushd $TESTBED/.linkerd2
  # ./bin/linkerd install --crds | kubectl apply -f -
  # ./bin/linkerd install | kubectl apply -f -
  # popd
  
  # # Wait for control plane to get running
  # sleep 30
fi

popd