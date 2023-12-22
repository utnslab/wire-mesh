#!/bin/bash
# Install Docker, Kubernetes, and other dependencies.

showHelp() {
cat << EOF  
Usage: <script_name> [-ic]
Install docker, kubernetes and helm.

-h, -help,      --help        Display help
-i, -init,      --init        Whether installing docker for the first time
-c, -control,   --control     Is the node a control node

EOF
}

INIT=0
IS_CONTROL_NODE=0

options=$(getopt -l "help,init,control" -o "hic" -a -- "$@")

eval set -- "$options"

while true; do
  case "$1" in
  -h|--help) 
      showHelp
      exit 0
      ;;
  -i|--init)
      INIT=1
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

# Check if init
if [[ $INIT -eq 1 ]]; then
  # <=========== Install Docker ===========>
  sudo apt --fix-broken install -y
  sudo apt-get update

  sudo apt-get install -y \
      ca-certificates \
      curl \
      gnupg \
      lsb-release \
      tmux

  sudo install -m 0755 -d /etc/apt/keyrings
  curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo gpg --batch --yes --dearmor -o /etc/apt/keyrings/docker.gpg
  sudo chmod a+r /etc/apt/keyrings/docker.gpg
  echo \
    "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/ubuntu \
    "$(. /etc/os-release && echo "$UBUNTU_CODENAME")" stable" | \
    sudo tee /etc/apt/sources.list.d/docker.list > /dev/null
  sudo apt-get update

  sudo apt-get install -y docker-ce=5:24.0.5-1~ubuntu.20.04~focal docker-ce-cli=5:24.0.5-1~ubuntu.20.04~focal containerd.io docker-compose-plugin

  # Change the data-root for docker
  mkdir -p /mydata/local/docker
  echo "{ \"data-root\": \"/mydata/local/docker\" }" | sudo tee /etc/docker/daemon.json
  sudo systemctl restart docker

  # Add user to docker group
  sudo groupadd docker
  sudo usermod -aG docker $USER

  # Adding a sleep so that kubernetes can be installed after docker
  sleep 30

  # <=========== Install Kubernetes ===========>
  # Install Kubectl
  sudo apt-get update
  sudo apt-get install -y apt-transport-https ca-certificates curl

  curl -fsSL https://packages.cloud.google.com/apt/doc/apt-key.gpg | sudo gpg --batch --yes --dearmor -o /etc/apt/keyrings/kubernetes-archive-keyring.gpg
  echo "deb [signed-by=/etc/apt/keyrings/kubernetes-archive-keyring.gpg] https://apt.kubernetes.io/ kubernetes-xenial main" | sudo tee /etc/apt/sources.list.d/kubernetes.list

  sudo apt-get update
  sudo apt-get install -y kubelet kubeadm kubectl

  # Initialize kubeadm cluster
  sudo swapoff -a

  sudo rm -f /etc/containerd/config.toml

  # Change the root directory for containerd
  sudo mkdir -p /mydata/local/containerd
  echo "root = \"/mydata/local/containerd\"" | sudo tee /etc/containerd/config.toml
  sudo systemctl restart containerd

  # <============== Install Helm ==============>
  curl https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash

  if [[ ${IS_CONTROL_NODE} -eq 1 ]]; then
    # Install CRI Tools
    wget https://github.com/kubernetes-sigs/cri-tools/releases/download/v1.26.0/crictl-v1.26.0-linux-amd64.tar.gz
    sudo tar zxvf crictl-v1.26.0-linux-amd64.tar.gz -C /usr/local/bin
    rm -f crictl-v1.26.0-linux-amd64.tar.gz
  fi
fi

# Reset kubadm, if already present.
sudo kubeadm reset -f
sudo rm -rf ~/.kube

if [[ ${IS_CONTROL_NODE} -eq 0 ]]; then
  CMD=$(cat command.txt)
  sudo $CMD
else
  echo "Setting up Control Node"
  sudo kubeadm init --pod-network-cidr=10.244.0.0/16

  mkdir -p $HOME/.kube
  sudo cp -i /etc/kubernetes/admin.conf $HOME/.kube/config
  sudo chown $(id -u):$(id -g) $HOME/.kube/config
  
  sudo kubectl apply -f https://raw.githubusercontent.com/coreos/flannel/master/Documentation/kube-flannel.yml
  
  sleep 10
  
  CMD=$(sudo kubeadm token create --print-join-command)
  echo $CMD > ./command.txt

  # Mark this node for scheduling as well
  kubectl taint nodes --all node-role.kubernetes.io/control-plane-
fi
