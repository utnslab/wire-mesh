#!/usr/bin/env bash
# Arguments:
# 1: Name of the experiment
# 2: Start node
# 3: End node

HOSTS=`./cloudlab/nodes.sh $1 $2 $3 --all`

echo "Configuring public keys for first node"
i=0
for host in $HOSTS; do
  echo $host
  if [ $i -eq 0 ] ; then
    echo "Test"
    ssh -o StrictHostKeyChecking=no $host "ssh-keygen"
    pkey=`ssh -o StrictHostKeyChecking=no $host "cat ~/.ssh/id_rsa.pub"`
    let i=$i+1
    continue
  fi
  ssh -o StrictHostKeyChecking=no $host "echo -e \"$pkey\" >> ~/.ssh/authorized_keys"
done

TARBALL=scripts.tar.gz
tar -czf $TARBALL scripts/

for host in $HOSTS; do
  echo "Pushing to $host ..."
  scp -rq -o StrictHostKeyChecking=no $TARBALL $host:~/ >/dev/null 2>&1 &
done
wait

for host in $HOSTS; do
  ssh -o StrictHostKeyChecking=no $host "mkdir -p scripts; tar -xzf $TARBALL 2>&1" &
done
wait

rm -f $TARBALL

# Before anything, update linux kernel
for host in $HOSTS; do
  echo "Updating kernel on $host ..."
  ssh -o StrictHostKeyChecking=no $host "./scripts/update_kernel.sh 2>&1" &
done
wait

# Wait for the nodes to reboot
sleep 1m
echo "Waiting for nodes to reboot ..."

# Check if the nodes are reachable via a SSH command every 1 minute
while [ 1 ]; do
  FLAG=0
  for host in $HOSTS; do
    HOSTNAME=$(echo $host | awk -F'@' '{print $2}')
    nc -zw 1 $HOSTNAME 22 > /dev/null
    OUT=$?
    if [ $OUT -eq 1 ] ; then
      echo "Waiting for $host to come up ..."
      FLAG=1
      sleep 1m
    fi
  done
  if [ $FLAG -eq 0 ]; then
    break
  fi
done

# Increase space on the nodes
for host in $HOSTS ; do
  echo "Configuring dependencies for $host"
  ssh -o StrictHostKeyChecking=no $host "tmux new-session -d -s config \"
    sudo mkdir -p /mydata &&
    sudo /usr/local/etc/emulab/mkextrafs.pl /mydata &&

    pushd /mydata/local &&
    sudo chmod 775 -R . &&
    popd\""

done

# Get the control node (first node in the first line of $HOSTS)
CONTROL_NODE=$(echo $HOSTS | head -1 | awk '{print $1}')

# Setup control node
echo "Building on control node ${CONTROL_NODE}"
ssh -o StrictHostKeyChecking=no ${CONTROL_NODE} "cd \$HOME; ./scripts/install_docker.sh --init --control > install_docker.log 2>&1"

# Get the join command
scp -rq -o StrictHostKeyChecking=no ${CONTROL_NODE}:~/command.txt command.txt >/dev/null 2>&1

# Setup worker nodes
for host in $HOSTS; do
  echo "Preparing $host ..."
  if [[ "$host" != "${CONTROL_NODE}" ]]; then
    scp -rq -o StrictHostKeyChecking=no command.txt $host:~/ >/dev/null 2>&1
    ssh -o StrictHostKeyChecking=no $host "cd \$HOME; sudo ./scripts/install_docker.sh --init > install_docker.log 2>&1" &
  fi
done
wait

rm command.txt

# After joining the nodes, make a rollout restart of coredns on control node.
ssh -o StrictHostKeyChecking=no ${CONTROL_NODE} "kubectl -n kube-system rollout restart deployment coredns"

for host in $HOSTS ; do
  echo "Configuring dependencies for $host"
  ssh -o StrictHostKeyChecking=no $host "tmux new-session -d -s config \"
    cd \$HOME &&
    sudo apt-get update &&
    sudo apt install -y clang llvm gcc-multilib libelf-dev libpcap-dev build-essential &&
    sudo apt install -y linux-tools-common linux-tools-generic linux-headers-generic &&
    sudo apt install -y linux-tools-\$(uname -r) linux-headers-\$(uname -r) &&
    sudo apt install -y tcpdump jq &&
    
    curl https://bootstrap.pypa.io/pip/3.6/get-pip.py -o get-pip.py &&
    python3 get-pip.py &&
    python3 -m pip install psutil asyncio aiohttp &&

    wget https://go.dev/dl/go1.20.7.linux-amd64.tar.gz &&
    sudo rm -rf /usr/local/go && sudo tar -C /usr/local -xzf go1.20.7.linux-amd64.tar.gz &&
    echo \"export PATH=\$PATH:/usr/local/go/bin\" >> ~/.bashrc &&
    rm go1.20.7.linux-amd64.tar.gz &&

    curl -LO https://storage.googleapis.com/minikube/releases/latest/minikube-linux-amd64 &&
    sudo install minikube-linux-amd64 /usr/local/bin/minikube &&
    mkdir -p \$HOME/out &&
    mkdir -p \$HOME/logs\""

done