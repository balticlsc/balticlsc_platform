#!/bin/bash

docker save k8snsctrl:latest > /tmp/img.tar && k3s ctr image import /tmp/img.tar
