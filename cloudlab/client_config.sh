#!/usr/bin/env bash
# Arguments:
# 1: Name of the experiment
# 2: Client node

HOSTS=`./cloudlab/nodes.sh $1 $2 $2 --all`

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

for host in $HOSTS ; do
  echo "Configuring dependencies for $host"
  ssh -o StrictHostKeyChecking=no $host "tmux new-session -d -s config \"
    cd \$HOME &&
    sudo apt-get update &&
    sudo apt install -y clang llvm gcc-multilib libelf-dev libpcap-dev build-essential &&
    
    curl https://bootstrap.pypa.io/pip/3.6/get-pip.py -o get-pip.py &&
    python3 get-pip.py &&
    python3 -m pip install psutil asyncio aiohttp &&

    sudo apt install -y luarocks &&
    sudo luarocks install luasocket &&

    git clone https://github.com/DivyanshuSaxena/DeathStarBench.git &&
    pushd DeathStarBench/wrk2 &&
    make -j 4 &&
    popd &&

    mkdir -p \$HOME/out &&
    mkdir -p \$HOME/logs\""

done
