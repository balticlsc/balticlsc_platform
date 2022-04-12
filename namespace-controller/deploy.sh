#!/usr/bin/env bash

#make bin/k8snsctrl
#make docker-image
#make push-image

set -euo pipefail

namespace="icekube"

echo "-> Creating k8snsctrl namespace"
kubectl create namespace ${namespace}

basedir="$(dirname "$0")/deployment"
mkdir -p keys
keydir="$(dirname "$0")/keys"

echo "-> Generating TLS keys ..."
"${basedir}/generate-keys.sh" "$keydir" "${namespace}"

echo "-> Creating TLS secret"
kubectl -n ${namespace} create secret tls k8snsctrl-server-tls \
    --cert "${keydir}/k8snsctrl-server-tls.crt" \
    --key "${keydir}/k8snsctrl-server-tls.key"

echo "-> Successfully created TLS keys and secret"

echo "-> Generating deployment.yaml file"
ca_pem_b64="$(openssl base64 -A <"${keydir}/ca.crt")"
sed -e 's@${CA_PEM_B64}@'"$ca_pem_b64"'@g' -e 's@${NAMESPACE}@'"${namespace}"'@g' <"${basedir}/deployment.yaml.template" \
    | kubectl create -f -

echo "-> Successfully deployed k8snsctrl server"
