#!/usr/bin/env bash

set -euo pipefail

namespace="icekube"
basedir="$(dirname "$0")/deployment"
keydir="$(dirname "$0")/keys"

kubectl delete namespaces ${namespace} 

ca_pem_b64="$(openssl base64 -A <"${keydir}/ca.crt")"
sed -e 's@${CA_PEM_B64}@'"$ca_pem_b64"'@g' -e 's@${NAMESPACE}@'"${namespace}"'@g' <"${basedir}/deployment.yaml.template" \
    | kubectl delete -f -


echo "-> Successfully undeployed k8snsctrl server"
