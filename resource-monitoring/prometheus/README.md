# Installation

## Requirements
1. Install kubectl and download your kubeconf to ~/.kube/config
2. Have metering enabled in rancher for you cluster

## Install custom prometheus rules to your cluster
```
kubectl apply -f custom-metrics/rise-metrics-crd.yaml
```

## Resource usage API Documentation
Metering API documentation is found here: [api/README.md](api/README.md)