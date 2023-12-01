#!/bin/bash
# Uninstall service mesh for the given argument
# Arguments:
# --mesh: service mesh name

showHelp() {
cat << EOF  
Usage: <script_name> -m <mesh-name>
Attach bpf programs for a specific service.

-h, -help,      --help        Display help
-m, -mesh,      --mesh        Service mesh name (istio/linkerd/nginx/plain)

EOF
}

MESH=""

options=$(getopt -l "help,mesh:" -o "hm:" -a -- "$@")

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
  --)
      shift
      break;;
  esac
  shift
done

: "${TESTBED:=$HOME}"

cd $TESTBED

# Uninstall NGINX Service Mesh before removing application
if [[ $MESH == "nginx" ]]; then
  # Uninstall NGINX
  ./nginx-meshctl remove -y
fi

# Perform the helm uninstall, in case.
helm uninstall social-network

# Remove hotel-reservation application
pushd DeathStarBench/hotelReservation
kubectl delete -Rf kubernetes/
popd

# Remove boutique application
pushd scripts/deployment/boutique
kubectl delete -f kubernetes-manifests.yaml
kubectl delete -f istio-manifests.yaml
popd

# Remove bookinfo application
pushd scripts/deployment/bookinfo
kubectl delete -f .
popd

# Remove the ingress - if any.
kubectl delete all --all -n ingress-nginx

# Wait for the pods to get deleted
sleep 1m

if [[ $MESH == "istio" ]]; then
  # Uninstall Istio
  pushd $TESTBED/istio-1.16.1
  ./bin/istioctl uninstall --purge -y
  popd
elif [[ $MESH == "linkerd" ]]; then
  # Uninstall Linkerd
  pushd $TESTBED/.linkerd2
  ./bin/linkerd uninstall | kubectl delete -f -
  popd  
fi

# # Reset kubeadm and reset kubernetes cluster
# sudo kubeadm reset -f
# sudo rm -rf ~/.kube

# # NOTE: Needed for some corner cases
# sleep 10

# sudo kubeadm init --pod-network-cidr=10.244.0.0/16

# mkdir -p $HOME/.kube
# sudo cp -i /etc/kubernetes/admin.conf $HOME/.kube/config
# sudo chown $(id -u):$(id -g) $HOME/.kube/config

# sudo kubectl apply -f https://raw.githubusercontent.com/coreos/flannel/master/Documentation/kube-flannel.yml

# # Mark this node for scheduling as well
# kubectl taint nodes --all node-role.kubernetes.io/control-plane-
