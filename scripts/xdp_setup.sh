#!/bin/bash
# Install utilities for XDP

# git clone https://github.com/xdp-project/xdp-tutorial.git
# pushd xdp-tutorial
# git submodule update --init
# popd

sudo apt-get update
sudo apt install -y clang llvm libelf-dev libpcap-dev gcc-multilib build-essential
sudo apt install -y linux-tools-$(uname -r)
sudo apt install linux-headers-$(uname -r)

sudo apt install -y linux-tools-common linux-tools-generic
sudo apt install -y tcpdump

# If on ARM system, create symbolic link to asm
ARCH=$(arch)
if [[ $ARCH == "arm" ]]; then
    sudo ln -s /usr/include/aarch64-linux-gnu/asm/ /usr/include/asm
fi

# Install Nmap
sudo apt-get install -y nmap