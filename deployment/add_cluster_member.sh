#!/bin/bash

source 'rancher.conf'

APITOKEN=`cat apitoken`
PROJECTID=`cat projectid`
CLUSTERID=`cat clusterid`
USERID=`cat userid`
ADD_CLUSTER_MEMBER_REQUEST_JSON='{"type":"clusterRoleTemplateBinding","clusterId":"'$CLUSTERID'","userPrincipalId":"local://'$USERID'","roleTemplateId":"nodes-view"}'

ADD_CLUSTER_MEMBER_RESP=`curl -s $RANCHER_SERVER_URL'/v3/clusterroletemplatebinding' -H 'content-type: application/json' -H "Authorization: Bearer $APITOKEN" --data-binary $ADD_CLUSTER_MEMBER_REQUEST_JSON --insecure`
echo $ADD_CLUSTER_MEMBER_RESP
