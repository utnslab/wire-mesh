#!/bin/bash
# Update the kernel to the latest version

# wget https://raw.githubusercontent.com/pimlie/ubuntu-mainline-kernel.sh/master/ubuntu-mainline-kernel.sh
# chmod +x ubuntu-mainline-kernel.sh

# sudo ./ubuntu-mainline-kernel.sh --yes -i 5.14.0

# Install the latest kernel
sudo apt-get update
sudo apt install -y linux-oem-20.04b

# Reboot the machine
sudo reboot