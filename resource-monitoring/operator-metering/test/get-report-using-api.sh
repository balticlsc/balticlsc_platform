#!/bin/bash

AUTH_TOKEN="XXXXX"
URL="https://ecckube.north.sics.se/k8s/clusters/c-vqbq2/api/v1/namespaces/metering/services/reporting-operator:api/proxy"

# All reports: 
# rise-by-project-gpu-request-daily
# rise-pod-cpu-request-hourly
# rise-pod-cpu-usage-hourly
# rise-pod-gpu-request-hourly
# rise-pod-memory-request-hourly
# rise-pod-memory-usage-hourly
# rise-pod-pvc-request-hourly
# rise-pod-pvc-usage-hourly
# rise-project-pod-gpu-request-hourly

function get_report() {
  curl -X GET --user $AUTH_TOKEN $URL"/api/v1/reports/get?name=$1&namespace=metering&format=tabular"
}

# GPU request
get_report $1

