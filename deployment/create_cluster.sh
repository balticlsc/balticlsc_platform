#!/bin/bash

source 'rancher.conf'

APITOKEN=`cat apitoken`

REQUEST_JSON='{"dockerRootDir":"/var/lib/docker","enableClusterAlerting":false,"enableClusterMonitoring":false,"enableNetworkPolicy":true,"windowsPreferedCluster":false,"type":"cluster","name":"'$CLUSTER_NAME'","rancherKubernetesEngineConfig":{"addonJobTimeout":30,"ignoreDockerVersion":true,"sshAgentAuth":false,"type":"rancherKubernetesEngineConfig","kubernetesVersion":"v1.17.5-rancher1-1","authentication":{"strategy":"x509","type":"authnConfig"},"dns":{"type":"dnsConfig","nodelocal":{"type":"nodelocal","ip_address":"","node_selector":null,"update_strategy":{}}},"network":{"mtu":0,"plugin":"canal","type":"networkConfig","options":{"flannel_backend_type":"vxlan"}},"ingress":{"provider":"nginx","type":"ingressConfig"},"monitoring":{"provider":"metrics-server","replicas":1,"type":"monitoringConfig"},"services":{"type":"rkeConfigServices","kubeApi":{"alwaysPullImages":false,"podSecurityPolicy":true,"serviceNodePortRange":"30000-32767","type":"kubeAPIService"},"etcd":{"creation":"12h","extraArgs":{"heartbeat-interval":500,"election-timeout":5000},"gid":0,"retention":"72h","snapshot":false,"uid":0,"type":"etcdService","backupConfig":{"enabled":false,"intervalHours":12,"retention":6,"safeTimestamp":false,"type":"backupConfig"}}},"upgradeStrategy":{"maxUnavailableControlplane":"1","maxUnavailableWorker":"10%","drain":"false","nodeDrainInput":{"deleteLocalData":"false","force":false,"gracePeriod":-1,"ignoreDaemonSets":true,"timeout":120,"type":"nodeDrainInput"},"maxUnavailableUnit":"percentage"}},"localClusterAuthEndpoint":{"enabled":false,"type":"localClusterAuthEndpoint"},"labels":{},"defaultPodSecurityPolicyTemplateId":"unrestricted","scheduledClusterScan":{"enabled":false,"scheduleConfig":null,"scanConfig":null}}'

CLUSTERRESPONSE=`curl -s $RANCHER_SERVER_URL'/v3/cluster' -H 'content-type: application/json' -H "Authorization: Bearer $APITOKEN" --data-binary $REQUEST_JSON --insecure`
CLUSTERID=`echo $CLUSTERRESPONSE | jq -r .id`
printf "${CLUSTERID}" > clusterid

ROLEFLAGS="--etcd"
AGENTCOMMAND=`curl -s $RANCHER_SERVER_URL'/v3/clusterregistrationtoken' -H 'content-type: application/json' -H "Authorization: Bearer $APITOKEN" --data-binary '{"type":"clusterRegistrationToken","clusterId":"'$CLUSTERID'"}' --insecure | jq -r .nodeCommand`
printf "#!/bin/bash\n\n${AGENTCOMMAND} ${ROLEFLAGS}" > node_cmd_etcd.sh
chmod +x node_cmd_etcd.sh

ROLEFLAGS="--controlplane"
AGENTCOMMAND=`curl -s $RANCHER_SERVER_URL'/v3/clusterregistrationtoken' -H 'content-type: application/json' -H "Authorization: Bearer $APITOKEN" --data-binary '{"type":"clusterRegistrationToken","clusterId":"'$CLUSTERID'"}' --insecure | jq -r .nodeCommand`
printf "#!/bin/bash\n\n${AGENTCOMMAND} ${ROLEFLAGS}" > node_cmd_controlplane.sh
chmod +x node_cmd_controlplane.sh

ROLEFLAGS="--worker"
AGENTCOMMAND=`curl -s $RANCHER_SERVER_URL'/v3/clusterregistrationtoken' -H 'content-type: application/json' -H "Authorization: Bearer $APITOKEN" --data-binary '{"type":"clusterRegistrationToken","clusterId":"'$CLUSTERID'"}' --insecure | jq -r .nodeCommand`
printf "#!/bin/bash\n\n${AGENTCOMMAND} ${ROLEFLAGS}" > node_cmd_worker.sh
chmod +x node_cmd_worker.sh

ROLEFLAGS="--etcd --controlplane"
AGENTCOMMAND=`curl -s $RANCHER_SERVER_URL'/v3/clusterregistrationtoken' -H 'content-type: application/json' -H "Authorization: Bearer $APITOKEN" --data-binary '{"type":"clusterRegistrationToken","clusterId":"'$CLUSTERID'"}' --insecure | jq -r .nodeCommand`
printf "#!/bin/bash\n\n${AGENTCOMMAND} ${ROLEFLAGS}" > node_cmd_etcd_controlplane.sh
chmod +x node_cmd_etcd_controlplane.sh

ROLEFLAGS="--etcd --worker"
AGENTCOMMAND=`curl -s $RANCHER_SERVER_URL'/v3/clusterregistrationtoken' -H 'content-type: application/json' -H "Authorization: Bearer $APITOKEN" --data-binary '{"type":"clusterRegistrationToken","clusterId":"'$CLUSTERID'"}' --insecure | jq -r .nodeCommand`
printf "#!/bin/bash\n\n${AGENTCOMMAND} ${ROLEFLAGS}" > node_cmd_etcd_worker.sh
chmod +x node_cmd_etcd_worker.sh

ROLEFLAGS="--controlplane --worker"
AGENTCOMMAND=`curl -s $RANCHER_SERVER_URL'/v3/clusterregistrationtoken' -H 'content-type: application/json' -H "Authorization: Bearer $APITOKEN" --data-binary '{"type":"clusterRegistrationToken","clusterId":"'$CLUSTERID'"}' --insecure | jq -r .nodeCommand`
printf "#!/bin/bash\n\n${AGENTCOMMAND} ${ROLEFLAGS}" > node_cmd_controlplane_worker.sh
chmod +x node_cmd_controlplane_worker.sh

ROLEFLAGS="--etcd --controlplane --worker"
AGENTCOMMAND=`curl -s $RANCHER_SERVER_URL'/v3/clusterregistrationtoken' -H 'content-type: application/json' -H "Authorization: Bearer $APITOKEN" --data-binary '{"type":"clusterRegistrationToken","clusterId":"'$CLUSTERID'"}' --insecure | jq -r .nodeCommand`
printf "#!/bin/bash\n\n${AGENTCOMMAND} ${ROLEFLAGS}" > node_cmd_etcd_controlplane_worker.sh
chmod +x node_cmd_etcd_controlplane_worker.sh
