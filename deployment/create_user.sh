#!/bin/bash

source 'rancher.conf'

APITOKEN=`cat apitoken`
USER_REQUEST_JSON='{"enabled":true,"me":false,"mustChangePassword":false,"type":"user","username":"'$USERNAME'","password":"'$PASSWORD'","name":"'$USERNAME'"}'
ADD_USER_RESP=`curl -s $RANCHER_SERVER_URL'/v3/user' -H 'content-type: application/json' -H "Authorization: Bearer $APITOKEN" --data-binary $USER_REQUEST_JSON --insecure`
echo $USERID

USERID=`echo $ADD_USER_RESP | jq -r .id`
printf "${USERID}" > userid

ROLE_REQUEST1_JSON='{"type":"globalRoleBinding","globalRoleId":"user-base","userId":"'$USERID'"}'
ADD_ROLE_RESP1=`curl -s $RANCHER_SERVER_URL'/v3/globalrolebinding' -H 'content-type: application/json' -H "Authorization: Bearer $APITOKEN" --data-binary $ROLE_REQUEST1_JSON --insecure`
ROLE_REQUEST2_JSON='{"type":"globalRoleBinding","globalRoleId":"catalogs-use","userId":"'$USERID'"}'
ADD_ROLE_RESP2=`curl -s $RANCHER_SERVER_URL'/v3/globalrolebinding' -H 'content-type: application/json' -H "Authorization: Bearer $APITOKEN" --data-binary $ROLE_REQUEST2_JSON --insecure`
