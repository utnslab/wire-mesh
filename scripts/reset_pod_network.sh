#!/bin/bash
# Restart pod network -- problem with flannel.

showHelp() {
cat << EOF  
Usage: <script_name> [-c]

-h, -help,      --help        Display help
-c, -control,   --control     Is the node a control node

EOF
}

IS_CONTROL_NODE=0

options=$(getopt -l "help,control" -o "hc" -a -- "$@")

eval set -- "$options"

while true; do
  case "$1" in
  -h|--help) 
      showHelp
      exit 0
      ;;
  -c|--control)
      IS_CONTROL_NODE=1
      ;;
  --)
      shift
      break;;
  esac
  shift
done

if [[ ${IS_CONTROL_NODE} -eq 1 ]]; then
  # Delete flannel
  kubectl delete -f https://raw.githubusercontent.com/coreos/flannel/master/Documentation/kube-flannel.yml
fi

sudo ip link set cni0 down && sudo ip link set flannel.1 down
sudo ip link delete cni0 && sudo ip link delete flannel.1
sudo systemctl restart docker && sudo systemctl restart kubelet

if [[ ${IS_CONTROL_NODE} -eq 1 ]]; then
  # Re-install flannel
  kubectl apply -f https://raw.githubusercontent.com/coreos/flannel/master/Documentation/kube-flannel.yml

  # Restart coredns
  sleep 5
  kubectl -n kube-system rollout restart deployment coredns
fi