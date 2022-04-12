# Implementation specification
ClusterProxy is a stateless proxy used to translate ClusterProxyAPI to BalticLSC Platform API.
The northbound interface called ClusterProxyAPI is implemented using gRPC. The southbound interfaces communicate with the BalticLSC Platform APIs:
* Rancher REST API
* Kubernetes REST API


## __Global configuration__
```
conf {
  clusterId: string,
  projectId: string,
  kubeConf: yaml,
  metricsConf: yaml
}
```

## __Common types__
```protobuf
message Empty {}

message BatchId {
  string id = 1;
}

message Module {
  string batchId = 1;
  string moduleId = 2;
}

message SimpleResponse {
  enum StatusCode {
    ACTIVE = 0;
    PENDING = 1;
    ERROR = 2;
    NOT_FOUND = 3;
  }
  StatusCode status = 1;
  string message = 2;    // Forward for instance extra ERROR info here
}

```
### _RPC Functions_
```protobuf
  rpc PrepareWorkspace(Workspace) returns (SimpleResponse) {}
  rpc PurgeWorkspace(BatchId) returns (SimpleResponse) {}
  rpc CheckWorkspaceStatus(BatchId) returns (SimpleResponse) {}
  rpc RunBalticModule(ServiceDescription) returns (SimpleResponse) {}
  rpc DisposeBalticModule(Module) returns (SimpleResponse) {}
  rpc CheckBalticModuleStatus(Module) returns (SimpleResponse) {}
  rpc GetClusterDescription(Empty) returns (ClusterDescription) {}  
```


## __PrepareWorkspace(Workspace) returns (SimpleResponse) {}__

Create and prepare runtime environment with enough quotas to be able to hold planned workloads.

### _Type definition_
```protobuf
message Workspace {
  string batchId = 1;
  message WorkspaceQuota {
    int32 cpu = 1;        // mCPUs
    int32 memory = 2;     // Mi,
    int32 storage = 3;    // Gi
    int32 gpu = 4;        // Number of GPUs of any type
  }
  WorkspaceQuota quota = 2;
}
```

### _Implementation details_
Create namespace in BalticLSC Platform with specified resource quota:

Variables specified with ${variableName}

```yaml
apiVersion: v1
kind: Namespace
metadata:
  name: balticlsc-${BatchID}
  annotations:
    field.cattle.io/projectId: ${conf.ClusterID}:${conf.ProjectId}
    field.cattle.io/resourceQuota: '{"limit":{"requestsCpu":"${resourceQuota.cpu}","requestsMemory":"${resourceQuota.memory}Mi","requestsStorage":"${resourceQuota.storage}Gi","limitsCpu":"${resourceQuota.cpu}","limitsMemory":"${resourceQuota.memory}Mi"}}'

---
# Deny all outgoing traffic from namespace except dns queries within cluster
# Allow traffic to namespace  
# Allow traffic within namespace - i.e pod2pod communication
kind: NetworkPolicy
apiVersion: networking.k8s.io/v1
metadata:
  name: default-policy
  namespace: balticlsc-${BatchID}
spec:
  policyTypes:
  - Ingress
  - Egress
  podSelector: {}
  ingress:
  - from:
    - podSelector:
        matchLabels: {}
  egress:
  - to:
    - namespaceSelector: {}
    ports:
    - protocol: UDP
      port: 53
  - to:
    - podSelector:
        matchLabels: {}

```

### _Return_
Send back SimpleResponse. Note that function returns current state that is probably "PENDING".

## __CheckWorkspaceStatus(BatchId) returns (SimpleResponse) {}__
Ask BalticLSC Platform of namespace status of namespace with name "balticlsc-${BatchID}" and respond with SimpleResponse. 

kubectl example:
```cmd
kubectl get ns balticlsc-${BatchID}

if status == "not found" then return simpleresponse.status.NOT_FOUND
if status == "Active" then return simpleresponse.status.ACTIVE
if status == "Error" then return simpleresponse.status.ERROR + set error message
```
### _Response_
Send back SimpleResponse. Note that function returns current state that is probably "PENDING".

## __PurgeWorkspace(BatchId) returns (SimpleResponse) {}__
Delete the whole namespace:
```
kubectl delete ns balticlsc-${BatchID}
```

## __RunBalticModule(ServiceDescription) returns (SimpleResponse) {}__
Deploy BalticModule on BalticLSC Platform.

### _Type definition_
```protobuf
message ServiceDescription {
  string batchId = 1;    // Unique id of batch
  string moduleId = 2;       // Unique id within batch
  string hostname = 3;
  string image = 4; 
  message EnvironmentVariable {
    string key = 1;
    string value = 2;
  }
  repeated EnvironmentVariable environmentVariables = 5;
  message Command {
    string program = 1;
    repeated string args = 2;
  }
  Command cmd = 6;
  message PortMap {
    int32 port = 1;
    enum Protocol {
      TCP = 0;
      UDP = 1;
    }
    Protocol protocol = 2;
  }
  repeated PortMap portMapping = 7;
  message VolumeDescription {
    int32 size = 1;            // Gi
    string storageClass = 2;
    string mountPath = 3;
  }
  repeated VolumeDescription volumes = 8;
  
  message ResourceRequest {
    int32 cpu = 1;        // mCPUs
    int32 memory = 2;     // MiB
    message gpuRequest {
      string type = 1;    // ex: nvidia-gtx-2080ti
      int32 qty = 2;        // number of gpus
    }
    gpuRequest gpu = 3;
  }
  ResourceRequest resources = 9;
  message FileMountDescription {
    string data = 1; // json, yaml...
    string mountPath = 2;
  }
  repeated FileMountDescription configFiles = 10;
}
```

### _Create PersistentVolume_
For each *VolumeDescription* in serviceDescription.volumes create one of these. 
```yaml
# serviceDescription.volumes
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  annotations:
  name: ${serviceDescription.moduleId}-{idx} # append volume index 0,1,2...
  namespace: balticlsc-${BatchID}
spec:
  accessModes:
  - ReadWriteOnce # volume.access
  resources:
    requests:
      storage: 10Gi # volume.size
  storageClassName: rook-ceph-rbd # volume.storageClass
  volumeMode: Filesystem

```
### _Create config-map (configuration files)_
For *FileMountDescription* in serviceDescription.configFiles. 

```yaml
apiVersion: v1
data:
  file-0: '${serviceDescription.configFiles[0].data}'
  file-1: '${serviceDescription.configFiles[1].data}'
kind: ConfigMap
metadata:
  name: ${serviceDescription.moduleId}
  namespace: balticlsc-${BatchID}
```

### _Create Service_
By creating a service the jobModule endpoint is easily found by BatchManager
using DNS lookup of *${serviceDescription.moduleId}.balticlsc-${BatchID}*

```yaml
apiVersion: v1
kind: Service
metadata:
  name: ${serviceDescription.moduleId}
  namespace: balticlsc-${BatchID}
spec:
  ports:
  - name: default
    port: 42
    protocol: TCP
    targetPort: 42
  selector:
    balticlsc/workloadselector: deployment-balticlsc-${BatchID}-${serviceDescription.moduleId}
  sessionAffinity: None
  type: ClusterIP
```

### _Create Deployment_ 
To be able to reserve different types of GPUs, the nodes in Cluster needs to be labeled with the type of GPU that is installed. Note that only one type of GPUs will be supported per node. For example a node with Nvidia GTX 2080ti GPUs will have label accelerator=nvidia-gtx-2080ti, a node with Nvidia GTX 1080ti GPUs will have label accelerator=nvidia-gtx-1080ti set. The following example shows how to reserve two GPUs of type Nvidia GTX 1080ti.

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    balticlsc/workloadselector: deployment-balticlsc-${BatchID}-${serviceDescription.moduleId}
  name: ${serviceDescription.moduleId}
  namespace: balticlsc-${BatchID}
spec:
  replicas: 1
  selector:
    matchLabels:
      balticlsc/workloadselector: deployment-balticlsc-${BatchID}-${serviceDescription.moduleId}
  template:
    metadata:
      labels:
        balticlsc/workloadselector: deployment-balticlsc-${BatchID}-${serviceDescription.moduleId}
    spec:
      affinity:
        nodeAffinity:
          requiredDuringSchedulingIgnoredDuringExecution:
            nodeSelectorTerms:
            - matchExpressions:
              - key: accelerator
                operator: In
                values:
                - nvidia-gtx-1080ti
      containers:
      - name: ${serviceDescription.moduleId}
        command:
        - ${serviceDescription.cmd.program}
        args: [${serviceDescription.cmd.args}]
        env: [${serviceDescription.environmentVariables}] # [{key:value}, {key2,value2}]
        image: ${serviceDescription.image}
        imagePullPolicy: Always
        resources:
          requests:
            cpu: ${serviceDescription.resources.cpu}m
            memory: ${serviceDescription.resources.memory}Mi
          limits:
            cpu: ${serviceDescription.resources.cpu}m
            memory: ${serviceDescription.resources.memory}Mi
            nvidia.com/gpu: 2
        securityContext:
          allowPrivilegeEscalation: false
          capabilities: {}
          privileged: false
          readOnlyRootFilesystem: false
          runAsNonRoot: false
        stdin: true
        tty: true
        volumeMounts:
        - name: volume-0
          mountPath: /storage
        - name: conf
          subPath: file-0
          mountPath: ${serviceDescription.configFiles[0].mountPath}
        - name: conf
          subPath: file-1
          mountPath: ${serviceDescription.configFiles[1].mountPath}
      dnsPolicy: ClusterFirst
      restartPolicy: Always
      schedulerName: default-scheduler
      terminationGracePeriodSeconds: 30
      volumes:
      - name: volume-0
        persistentVolumeClaim:
          claimName: ${serviceDescription.moduleId}-0
      - name: conf
        configMap:
          defaultMode: 256
          name: ${serviceDescription.moduleId}
          optional: false
        
```
### _Another GPU example_
if 4 GPUs of type Nvidia GTX 2080ti was to be reserved, then following configuration needs to be changed in above Deployment specification:

```yaml
    spec:
      affinity:
        nodeAffinity:
          requiredDuringSchedulingIgnoredDuringExecution:
            nodeSelectorTerms:
            - matchExpressions:
              - key: accelerator
                operator: In
                values:
                - nvidia-gtx-2080ti

            nvidia.com/gpu: 4

```
## __CheckBalticModuleStatus(Module) returns (SimpleResponse) {}__
Check status of all resources (service, persistenStorage, configMap, deployment) and return error if any resource in error state.

Example: check status of deployment and report back SimpleResponse.
```
if status.readyReplicas > 0 then return status=ACTIVE

if status.unavailableReplicas > 0 then return status=ERROR and set message to message from most recent condition message
```

## __DisposeBalticModule(Module) returns (SimpleResponse) {}__
Run delete on all created by RunBalticModule:
```cmd
kubectl -n balticlsc-${BatchID} delete service ${moduleID}
kubectl -n balticlsc-${BatchID} delete deployment ${moduleID}
kubectl -n balticlsc-${BatchID} delete configmap ${moduleID}
# per volume idx run
kubectl -n balticlsc-${BatchID} delete volume ${moduleID}-{idx}
```

## __GetClusterDescription(Empty) returns (ClusterDescription) {}__
Return description and state of all machines in the cluster

### Type definition
```protobuf
message ClusterDescription {
  string clusterId = 1;          // Rancher ClusterID
  string projectId = 2;          // Rancher ProjectID
  message ClusterNode {
    string name = 1;
    string os = 2;             // linux
    string osImage = 3;        // Ubuntu 18.04.5 LTS
    string arch = 4;           // amd64
    string kernel = 5;         // linux kernel version
    string orchestratorType = 6; // BalticLSC Platform / Docker swarm
    string orchestratorVersion = 7; 
    string accelerator = 8;    // gpu type (ex nvidia-gtx-2080ti or "")
    message Capacity {
      int32 cpu = 1;           // num cpu cores
      int32 ephemeralStorage = 2; // Ki
      int32 memory = 3;        // Ki
      int32 gpus = 4;
      int32 pods = 5;
    }
    Capacity capacity = 9;     // Total capacity of node
    Capacity allocatable = 10;  // Available capacity of node
    bool unschedulable = 11;   // True if not accepting workloads
    enum NodeStatus {
      READY = 1;
      DISK_PRESSURE = 2;
      MEMORY_PRESSURE = 3;
      PID_PRESSURE = 4;
      NETWORK_UNAVAILABLE = 5;
    }
    NodeStatus status = 12;
  }
  repeated ClusterNode nodes = 3;
  enum ClusterStatus {
    ONLINE = 1;
    UNREACHABLE = 2;
  }
  ClusterStatus status = 4;
  message ResourceQuota {
    int32 cpuRequest = 1;         // mCPUs
    int32 cpuLimit = 2;           // mCPUs
    int32 memoryRequest = 3;      // MiB
    int32 memoryLimit = 4;        // MiB
    int32 persistentVolumeClaims = 5; 
    uint32 storageRequest = 6;    // GB
    int32 gpuRequest = 7;         // Num GPUs (-1 unlimited)
  }
  ResourceQuota limit = 5;  
  ResourceQuota usedLimit = 6;  
}

```

### Implementation

Example using kubectl:
```
kubectl get nodes -o yaml
```
Query ResourceQuota from RancherAPI:
```
Action: GET
Path: /v3/projects/${clusterId}:${projectId}
```

## __GetWorkspaceStatistic(BatchID: string): BLSCWorkspaceStatistics__
TBD

# Documentation

## Installing batch-manager

Login into https://ice.ri.se or directly to https://k8s.ice.ri.se

Install batchmanager using wizard in namespace "balticlsc-default". When using wizard, remember to click "advanced options" and add CPU and Memory reservations in "Security & Host Config".

## Connecting to cluster-proxy

Addr: cluster-proxy
Port: 50051

Note: "cluster-proxy" DNS name will resolve to correct internal address

Example using telnet from busybox pod inside "balticlsc-default" namespace:

```
/ # telnet cluster-proxy 50051
Connected to cluster-proxy
```

## Connecting to BalticLSC modules from within Kubernetes

Namespace name is:
```go
func getNSNameForBatch(batchId string) string {
return proxy.projectName + "-" + batchId
}
```

Service name:
```go
func getServiceName(moduleId string) string {
return fmt.Sprintf("bltc-%s", moduleId)
}
```

Settings:
```
projectName: balticlsc
batchId: 1111
ModuleId: 1234
```
To connect to module with id 1234 from same namespace you use: bltc-1234
To connect to same module from other namespace you use: bltc-1234.balticlsc-1111

Note that you can always use the second alternative, i.e. from within same namespace

