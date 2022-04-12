#!/usr/bin/env bash

: ${1?'missing key directory'}

key_dir="$1"
namespace="$2"

chmod 0700 "$key_dir"
cd "$key_dir"

openssl req -nodes -new -x509 -keyout ca.key -out ca.crt -subj "/CN=RISE K8snsctrl namespace controller CA"
openssl genrsa -out k8snsctrl-server-tls.key 2048
openssl req -new -key k8snsctrl-server-tls.key -subj "/CN=k8snsctrl-server.${namespace}.svc" \
    | openssl x509 -req -CA ca.crt -CAkey ca.key -CAcreateserial -out k8snsctrl-server-tls.crt
