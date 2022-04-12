#!/bin/bash

source 'rancher.conf'

APITOKEN=`cat apitoken`
PROJECTID=`cat projectid`
USERID=`cat userid`
ADD_PROJECT_MEMBER_REQUEST_JSON='{"type":"projectRoleTemplateBinding","subjectKind":"User","userId":"","projectRoleTemplateId":"","projectId":"'$PROJECTID'","userPrincipalId":"local://'$USERID'","roleTemplateId":"project-member"}'

ADD_PROJECT_MEMBER_RESP=`curl -s $RANCHER_SERVER_URL'/v3/projectroletemplatebinding' -H 'content-type: application/json' -H "Authorization: Bearer $APITOKEN" --data-binary $ADD_PROJECT_MEMBER_REQUEST_JSON --insecure`
echo $ADD_PROJECT_MEMBER_RESP
