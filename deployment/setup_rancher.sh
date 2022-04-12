#!/bin/bash

source 'rancher.conf'

while ! curl -k $RANCHER_SERVER_URL/ping; do sleep 3; done

LOGINRESPONSE=`curl -s $RANCHER_SERVER_URL'/v3-public/localProviders/local?action=login' -H 'content-type: application/json' --data-binary '{"username":"admin","password":"admin"}' --insecure`
LOGINTOKEN=`echo $LOGINRESPONSE | jq -r .token`

curl -s $RANCHER_SERVER_URL'/v3/users?action=changepassword' -H 'content-type: application/json' -H "Authorization: Bearer $LOGINTOKEN" --data-binary '{"currentPassword":"admin","newPassword":"'$RANCHER_PASSWORD'"}' --insecure

APIRESPONSE=`curl -s $RANCHER_SERVER_URL'/v3/token' -H 'content-type: application/json' -H "Authorization: Bearer $LOGINTOKEN" --data-binary '{"type":"token","description":"automation"}' --insecure`
APITOKEN=`echo $APIRESPONSE | jq -r .token`
printf "${APITOKEN}" > apitoken

curl -s $RANCHER_SERVER_URL'/v3/settings/server-url' -H 'content-type: application/json' -H "Authorization: Bearer $APITOKEN" -X PUT --data-binary '{"name":"server-url","value":"'$RANCHER_SERVER_LOCAL_URL'"}' --insecure
