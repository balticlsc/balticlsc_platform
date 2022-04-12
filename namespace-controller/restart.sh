#!/bin/bash

make
kubectl -n k8snsctrl delete pods --all
