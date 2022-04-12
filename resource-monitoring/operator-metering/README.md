# Installation

## Install script dependency faq binary into /usr/local/bin
```
LATEST_RELEASE=$(curl -s https://api.github.com/repos/jzelinskie/faq/releases | cat | head -n 10 | grep "tag_name" | cut -d\" -f4)
sudo curl -Lo /usr/local/bin/faq https://github.com/jzelinskie/faq/releases/download/$LATEST_RELEASE/faq-linux-amd64
sudo chmod +x /usr/local/bin/faq
```

## Clone operator-metering repo:
operator-metering exists as an submodule to this project. Run following to clone submodules:

 git submodule update --init

You find the code in [repo]/upstream/operator-metering

Edit manifests/deploy/upstream/metering-ansible-operator/meteringconfig.yaml:
```
apiVersion: metering.openshift.io/v1
kind: MeteringConfig
metadata:
  name: operator-metering
spec:
  storage:
    type: "hive"
    hive:
      type: "sharedPVC"
      sharedPVC:
        claimName: "metering-nfs"
  reporting-operator:
    spec:
      config:
        prometheus:
          url: "http://access-prometheus.cattle-prometheus"
```

## Create namespace "metering" in project "system"

In metering namespace create two persistent storage volumes:
1. hive-metastore-db-data: accessMode ReadWriteOnce, can be simle volume like ceph block
2. metering-nfs: need accessModes ReadWriteMany like cephfs or nfs

Install operator-metering
```
cd hack
./upstream-install.sh
```

