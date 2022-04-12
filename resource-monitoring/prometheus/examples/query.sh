#!/bin/bash

# API token
AUTHTOKEN="token-p6drl:rwzd54rvnjtfrg7nmzdhqxllbk69cmrfzlvrlhj9w29nqrvshscgjf"

# Request payload
JSON='{"filters":{"resourceType":"pod","projectId":"c-tmfxj:p-vxl6k"},"metricParams":{"podName":"balticlsc-jlab:ssdl-jupyterlab-lab-766fcb86c5-r6ttr"},"interval":"5s","isDetails":true,"from":"now-5m","to":"now"}'

curl -u ${AUTHTOKEN} -d ${JSON} -X POST https://k8s.ice.ri.se/v3/projectmonitorgraphs?action=query
