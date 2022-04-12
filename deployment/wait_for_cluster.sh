#!/bin/bash

source 'rancher.conf'

APITOKEN=`cat apitoken`
CLUSTERID=`cat clusterid`

while true; do
    GET_STATUS_RESP=`curl $RANCHER_SERVER_URL'/v3/clusters/'$CLUSTERID -H "Authorization: Bearer $APITOKEN" --insecure`
    STATE=`echo $GET_STATUS_RESP | jq -r .state`
    echo $STATE
    if [ $STATE = "active" ]; then 
      break
    fi
    sleep 1
done
