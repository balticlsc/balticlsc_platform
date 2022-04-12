#!/bin/bash
namespace="k8snsctrl"
if [ "$1" != "" ] ; then
    namespace=$1
fi
kubectl -n $namespace logs -l app=k8snsctrl-server --follow
