#!/bin/bash

source 'rancher.conf'

APITOKEN=`cat apitoken`
ADD_POLICY_REQUEST_JSON='{"hostIPC":false,"hostNetwork":false,"hostPID":false,"privileged":false,"readOnlyRootFilesystem":false,"type":"podSecurityPolicyTemplate","name":"'$POLICY_NAME'","allowPrivilegeEscalation":false,"defaultAllowPrivilegeEscalation":false,"allowedCapabilities":["CHOWN","NET_BIND_SERVICE","NET_BROADCAST","NET_RAW","SETGID","SETUID"],"defaultAddCapabilities":["CHOWN","NET_BIND_SERVICE","NET_BROADCAST","NET_RAW","SETGID","SETUID"],"requiredDropCapabilities":["AUDIT_CONTROL"],"volumes":["emptyDir","secret","persistentVolumeClaim","configMap"],"allowedHostPaths":[],"fsGroup":{"type":"fsGroupStrategyOptions","rule":"RunAsAny"},"hostPorts":[],"runAsUser":{"type":"runAsUserStrategyOptions","rule":"RunAsAny"},"seLinux":{"type":"seLinuxStrategyOptions","rule":"RunAsAny"},"supplementalGroups":{"type":"supplementalGroupsStrategyOptions","rule":"RunAsAny"},"labels":{}}'

ADD_POLICY_RESP=`curl -s $RANCHER_SERVER_URL'/v3/podsecuritypolicytemplate' -H 'content-type: application/json' -H "Authorization: Bearer $APITOKEN" --data-binary $ADD_POLICY_REQUEST_JSON --insecure`

echo $ADD_POLICY_RESP
