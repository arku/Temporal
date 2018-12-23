#! /bin/bash

# Used to install, and configure IPFS Cluster

NODE="initial_peer"
VERSION="v0.7.0"
IPFS_CLUSTER_PATH=/ipfs/ipfs-cluster
export IPFS_CLUSTER_PATH=/ipfs/ipfs-cluster

# initial peer is responsible for generating a cluster secret
if [[ "$NODE" == "initial_peer" ]]; then
    CLUSTER_SECRET=$(od  -vN 32 -An -tx1 /dev/urandom | tr -d ' \n')
    export CLUSTER_SECRET
elif [[ "$CLUSTER_SECRET" == "" ]]; then
    echo "[ERROR] Please set CLUSTER_SECRET environment variable"
    exit 1
fi

# download and install ipfs cluster service
cd ~ || exit
echo "[INFO] Downloading ipfs cluster"
wget "https://dist.ipfs.io/ipfs-cluster-service/${VERSION}/ipfs-cluster-service_${VERSION}_linux-amd64.tar.gz"
tar zxvf ipfs-cluster-service*.tar.gz
rm -- *gz
cd ipfs-cluster-service || exit
echo "[INFO] Initializing ipfs cluster configuration"
./ipfs-cluster-service init
sudo cp ipfs-cluster-service /usr/local/bin

# download and install ipfs cluster utility tool
cd ~ || exit
echo "[INFO] Downloading ipfs cluster ctl"
wget "https://dist.ipfs.io/ipfs-cluster-ctl/${VERSION}/ipfs-cluster-ctl_${VERSION}_linux-amd64.tar.gz"
tar zxvf ipfs-cluster-ctl*.tar.gz
rm -- *gz
cd ipfs-cluster-ctl || exit
sudo cp ipfs-cluster-ctl /usr/local/bin
echo "[INFO]  IPFS cluster service installed"