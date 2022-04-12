#!/bin/bash

source 'rancher.conf'

APITOKEN=`cat apitoken`
CLUSTERID=`cat clusterid`
ADD_PROJECT_REQUEST_JSON='{"enableProjectMonitoring":false,"type":"project","name":"'$PROJECT_NAME'","clusterId":"'$CLUSTERID'","labels":{},"podSecurityPolicyTemplateId":"'$POLICY_NAME'","resourceQuota":{"limit":{"limitsCpu":"100000m","limitsMemory":"256000Mi","requestsMemory":"256000Mi","requestsCpu":"100000m","persistentVolumeClaims":"50"}},"namespaceDefaultResourceQuota":{"limit":{"limitsCpu":"10000m","limitsMemory":"1024Mi","requestsMemory":"1024Mi","requestsCpu":"10000m","persistentVolumeClaims":"10"}},"containerDefaultResourceLimit":{"requestsCpu":"1000m","limitsCpu":"1000m","requestsMemory":"128Mi","limitsMemory":"128Mi"}}'

ADD_PROJECT_RESP=`curl -s $RANCHER_SERVER_URL'/v3/project' -H 'content-type: application/json' -H "Authorization: Bearer $APITOKEN" --data-binary $ADD_PROJECT_REQUEST_JSON --insecure`
echo $ADD_PROJECT_RESP

PROJECTID=`echo $ADD_PROJECT_RESP | jq -r .id`
printf "${PROJECTID}" > projectid
