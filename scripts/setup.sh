cd $HOME &&

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
echo 'export PATH=\$PATH:/usr/local/go/bin' >> ~/.bashrc &&
rm go1.20.7.linux-amd64.tar.gz &&

curl -LO https://storage.googleapis.com/minikube/releases/latest/minikube-linux-amd64 &&
sudo install minikube-linux-amd64 /usr/local/bin/minikube &&
mkdir -p $HOME/out &&
mkdir -p $HOME/logs