# Introduction
This repo contains an Ansible playbook and scripts to create and setup a BalticLSC test or development cluster based on Rancher and Kubernetes. The playbook follows the instructions in the following youtube clip:
https://www.youtube.com/watch?v=e__ss4SA4hY&feature=youtu.be

Note that Rancher is installed as a single instance deployment with a self-signed certificate.

## Step 1 - Prerequisites
1. Install the jq tool

### MacOS
```
$ brew install jq
```

### Ubuntu 
```
$ apt install jq
```

2. Install ansible

### Mac
```
$ pip3 install --user ansible
```

### Ubuntu 
```
$ sudo apt update
$ sudo apt install software-properties-common
$ sudo apt-add-repository --yes --update ppa:ansible/ansible
$ sudo apt install ansible
```

Note that Ansible version 2.9.7 or higher is required.

## Step 2 - Create 3 ECC VMs
1. Login in to https://ecc.north.sics.se.
2. Create 4 with the following specs: VMs Large 6 vCPUs, 8 GB RAM, 24 GB Disk, Ubuntu 18.04 LTS. 
3. Name the VMs, rancher, node1, node2, and node3.

## Step 3 - Create an Ansible inventory
Edit the ecckube_dev_cluster.yml file and fill in correct information.
```
all:
  hosts:
    rancher:
      ansible_host: 109.225.89.161
      ansible_user: ubuntu
      ansible_port: 33979
      ansible_ssh_private_key_file: /Users/johan/.ssh/ecc.pem
  children:
      kubernetes_nodes:
        hosts:
          node1:
            ansible_host: 109.225.89.161
            ansible_user: ubuntu
            ansible_port: 33993
            ansible_ssh_private_key_file: /Users/johan/.ssh/ecc.pem
            etcd: true
            controlplane: true
            worker: true
          node2:
            ansible_host: 109.225.89.161
            ansible_user: ubuntu
            ansible_port: 33984
            ansible_ssh_private_key_file: /Users/johan/.ssh/ecc.pem
            etcd: true
            controlplane: true
            worker: true
          node3:
            ansible_host: 109.225.89.161
            ansible_user: ubuntu
            ansible_port: 33995
            ansible_ssh_private_key_file: /Users/johan/.ssh/ecc.pem
            etcd: true
            controlplane: true
            worker: true
```

## Step 5 - Try to ping all hosts to see that they can be reached
You might need to ssh into all hosts to add them to the ssh knownhost file.
```
$ ansible -i ecckube_dev_cluster.yml all -m ping
```

## Step 6 - Edit the rancher.conf file
Replace the URL with correct IPs obtained from Step 1.
```
RANCHER_SERVER_URL='https://109.225.89.130'
RANCHER_SERVER_LOCAL_URL='https://10.10.0.17'
RANCHER_PASSWORD='admin'
CLUSTER_NAME='mycluster'
USERNAME='myuser'
PASSWORD='mypassword'
POLICY_NAME='mycustom'
PROJECT_NAME='myproject4'
```

## Step 7 - Create the Rancher/Kubernetes cluster.
```
$ ansible-playbook  -i ecckube_dev_cluster.yml ecckube_playbook.yml
```
or simply:
```
$ ./install.sh
```

## Step 8 - Wait
This will take some time, may be 15-20 minutes.

## Uninstall everything
```
$ ansible-playbook  -i ecckube_dev_cluster.yml ecckube_playbook_uninstall.yml
```
or simply:
```
$ ./uninstall.sh
```
