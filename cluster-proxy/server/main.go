/*
 * ClusterProxy server
 *
 */

package main

import (
	pb "cluster-proxy/clusterproxy"
	rancher "cluster-proxy/rancher-go/client"
	"crypto/tls"
	"encoding/json"
	"os"

	"context"

	"flag"
	"fmt"
	"log"
	"net"

	"gopkg.in/resty.v1"
	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/golang/protobuf/ptypes/empty"
	"google.golang.org/grpc"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/clientcmd"
)

type ClusterProxyConf struct {
	config        *rest.Config
	clientset     *kubernetes.Clientset
	client        dynamic.Interface
	mapper        *restmapper.DeferredDiscoveryRESTMapper
	rancherClient rancher.Client
	debug         bool
	clusterId     string
	projectId     string
	projectName   string
}

type ResourceQuota struct {
	requestsCpu     int
	limitsCpu       int
	requestsMemory  int
	limitsMemory    int // in Mi
	requestsStorage int // in Gi
}

var proxy ClusterProxyConf

type server struct {
	pb.UnimplementedClusterProxyServer
}

func getNSNameForBatch(batchId string) string {
	return proxy.projectName + "-" + batchId
}

func getProjectID() string {
	return proxy.clusterId + ":" + proxy.projectId
}

func getVolumeName(moduleId string, idx int) string {
	return fmt.Sprintf("%s-%d", moduleId, idx)
}
func getServiceName(moduleId string) string {
	return fmt.Sprintf("%s", moduleId)
}

// PrepareWorkspace implementation
func (s *server) PrepareWorkspace(ctx context.Context, in *pb.XWorkspace) (*pb.ClusterStatusResponse, error) {
	quota := in.GetQuota()
	cpus := quota.GetCpus()
	memory := quota.GetMemory()
	storage := quota.GetStorage()
	log.Printf("PrepareWorkspace Received: %s", in.GetBatchId())
	if proxy.debug {
		json, err := json.MarshalIndent(in, "", "  ")
		if err == nil {
			fmt.Printf("message: %s\n", json)
		}
	}
	quotastr := fmt.Sprintf(
		"{\"limit\":{\"requestsCpu\":\"%dm\",\"requestsMemory\":\"%dMi\",\"requestsStorage\":\"%dGi\",\"limitsCpu\":\"%dm\",\"limitsMemory\":\"%dMi\",\"persistentVolumeClaims\":\"10\"}}",
		cpus, memory, storage, cpus, memory)
	ns := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: getNSNameForBatch(in.GetBatchId()),
			Annotations: map[string]string{
				"field.cattle.io/projectId":     getProjectID(),
				"field.cattle.io/resourceQuota": quotastr,
			},
		},
	}
	ret, err := proxy.clientset.CoreV1().Namespaces().Create(context.TODO(), ns, metav1.CreateOptions{})
	status := pb.ClusterStatusResponse_PENDING
	message := "Pending"
	if err != nil {
		log.Printf("error:", err)
		message = fmt.Sprintf("%s", err)
		status = pb.ClusterStatusResponse_ERROR
	} else if ret.Status.Phase == "Active" {
		status = pb.ClusterStatusResponse_ACTIVE
		message = "Active"
	}
	log.Printf("%+v", ret)
	return &pb.ClusterStatusResponse{Status: status, Message: message}, err

}

// CheckWorkspaceStatus implementation
func (s *server) CheckWorkspaceStatus(ctx context.Context, in *pb.BatchId) (*pb.ClusterStatusResponse, error) {
	log.Printf("CheckWorkspaceStatus Received: %s", in.GetId())
	if proxy.debug {
		json, err := json.MarshalIndent(in, "", "  ")
		if err == nil {
			fmt.Printf("message: %s\n", json)
		}
	}
	ret, err := proxy.clientset.CoreV1().Namespaces().Get(context.TODO(), getNSNameForBatch(in.GetId()), metav1.GetOptions{})
	log.Printf("got: %+v", ret)
	status := pb.ClusterStatusResponse_PENDING
	message := "Pending"
	if err != nil {
		log.Printf("error:", err)
		status = pb.ClusterStatusResponse_ERROR
		message = fmt.Sprintf("%s", err)
	} else if ret.Status.Phase == "Active" {
		status = pb.ClusterStatusResponse_ACTIVE
		message = "Active"
		annotations := ret.ObjectMeta.GetAnnotations()
		if cattlestatus, ok := annotations["cattle.io/status"]; ok {
			type Status []map[string]string
			var parsedStatus map[string]Status
			json.Unmarshal([]byte(cattlestatus), &parsedStatus)
			if condarr, ok := parsedStatus["Conditions"]; ok {
				for _, condition := range condarr {
					if condition["Type"] == "ResourceQuotaValidated" && condition["Status"] == "False" {
						log.Printf("Quota exceeded: %s\n", condition["Message"])
						status = pb.ClusterStatusResponse_ERROR
						message = fmt.Sprintf("%s", condition["Message"])
						break
					}
				}
			}
		}
	}
	return &pb.ClusterStatusResponse{Status: status, Message: message}, err
}

// PurgeWorkspace implementation
func (s *server) PurgeWorkspace(ctx context.Context, in *pb.BatchId) (*pb.ClusterStatusResponse, error) {
	log.Printf("PurgeWorkspace Received")
	if proxy.debug {
		json, err := json.MarshalIndent(in, "", "  ")
		if err == nil {
			fmt.Printf("message: %s\n", json)
		}
	}
	err := proxy.clientset.CoreV1().Namespaces().Delete(context.TODO(), getNSNameForBatch(in.GetId()), metav1.DeleteOptions{})
	status := pb.ClusterStatusResponse_NOT_FOUND
	message := "Deleted"
	if err != nil {
		log.Printf("error:", err)
		status = pb.ClusterStatusResponse_ERROR
		message = fmt.Sprintf("%s", err)
	}
	return &pb.ClusterStatusResponse{Status: status, Message: message}, err
}

// RunBalticModule implementation
func (s *server) RunBalticModule(ctx context.Context, in *pb.XBalticModuleBuild) (*pb.ClusterStatusResponse, error) {
	log.Printf("RunBalticModule Received")
	if proxy.debug {
		json, err := json.MarshalIndent(in, "", "  ")
		if err == nil {
			fmt.Printf("message: %s\n", json)
		}
	}
	// Create PVCs
	namespace := getNSNameForBatch(in.GetBatchId())
	var volumeMounts []v1.VolumeMount
	var volumes []v1.Volume
	for idx, vol := range in.GetVolumes() {
		storageClass := vol.GetStorageClass()
		volumeMode := v1.PersistentVolumeFilesystem
		pvc := v1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: namespace,
				Name:      getVolumeName(in.GetModuleId(), idx),
			},
			Spec: v1.PersistentVolumeClaimSpec{
				AccessModes: []v1.PersistentVolumeAccessMode{v1.ReadWriteOnce},
				Resources: v1.ResourceRequirements{
					Requests: v1.ResourceList{
						v1.ResourceStorage: *resource.NewQuantity(int64(vol.GetSize()*1024), resource.DecimalSI),
					},
				},
				VolumeMode: &volumeMode,
			},
		}
		if storageClass != "" {
			pvc.Spec.StorageClassName = &storageClass
		}
		ret, err := proxy.clientset.CoreV1().PersistentVolumeClaims(namespace).Create(ctx, &pvc, metav1.CreateOptions{})
		if err != nil {
			log.Printf("error creating pvc:", err)
			return &pb.ClusterStatusResponse{Status: pb.ClusterStatusResponse_ERROR, Message: fmt.Sprintf("%s", err)}, err
		}
		log.Printf("PersistentVolumeClaim ret %+v", ret)
		volumeMounts = append(volumeMounts, v1.VolumeMount{
			Name:      getVolumeName(in.GetModuleId(), idx),
			MountPath: vol.GetMountPath(),
		})
		volumes = append(volumes, v1.Volume{
			Name: getVolumeName(in.GetModuleId(), idx),
			VolumeSource: v1.VolumeSource{
				PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
					ClaimName: getVolumeName(in.GetModuleId(), idx),
				},
			},
		})
	}
	// Create ConfigMap
	var data = make(map[string]string)
	defaultMode := int32(256)
	for idx, cnf := range in.GetConfigFiles() {
		data[fmt.Sprintf("file-%d", idx)] = cnf.GetData()
		volumeMounts = append(volumeMounts, v1.VolumeMount{
			Name:      in.GetModuleId(),
			MountPath: cnf.GetMountPath(),
			SubPath:   fmt.Sprintf("file-%d", idx),
		})
	}
	if len(data) > 0 {
		cmap := v1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      in.GetModuleId(),
				Namespace: namespace,
			},
			Data: data,
		}
		volumes = append(volumes, v1.Volume{
			Name: in.GetModuleId(),
			VolumeSource: v1.VolumeSource{
				ConfigMap: &v1.ConfigMapVolumeSource{
					DefaultMode: &defaultMode,
					LocalObjectReference: v1.LocalObjectReference{
						Name: in.GetModuleId(),
					},
				},
			},
		})
		ret, err := proxy.clientset.CoreV1().ConfigMaps(namespace).Create(context.TODO(), &cmap, metav1.CreateOptions{})
		if err != nil {
			// we should probably cleanup here
			log.Printf("error creating config-map:", err)
			return &pb.ClusterStatusResponse{Status: pb.ClusterStatusResponse_ERROR, Message: fmt.Sprintf("%s", err)}, err
		}
		log.Printf("Create configmap ret: %+v", ret)
	}
	// Create Service
	ports := in.GetPortMappings()

	var servicePorts []v1.ServicePort
	var containerPorts []v1.ContainerPort
	for idx, port := range ports {
		protocol := v1.ProtocolUDP
		if port.GetProtocol() == pb.XPortMapping_TCP {
			protocol = v1.ProtocolTCP
		}
		servicePorts = append(servicePorts, v1.ServicePort{
			Port: int32(port.GetContainerPort()),
			//TargetPort: int32(port.GetPublishedPort()),
			Protocol: protocol,
			Name:     fmt.Sprintf("port-%d", idx),
		})
		containerPorts = append(containerPorts, v1.ContainerPort{
			ContainerPort: int32(port.GetContainerPort()),
			Protocol:      protocol,
		})
	}
	service := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      getServiceName(in.GetModuleId()),
			Namespace: namespace,
		},
		Spec: v1.ServiceSpec{
			Ports: servicePorts,
			Selector: map[string]string{
				"balticlsc/workloadselector": in.GetModuleId(),
			},
		},
	}
	_, err := proxy.clientset.CoreV1().Services(namespace).Create(context.TODO(), service, metav1.CreateOptions{})
	if err != nil {
		// we should probably cleanup here
		log.Printf("error creating service:", err)
		return &pb.ClusterStatusResponse{Status: pb.ClusterStatusResponse_ERROR, Message: fmt.Sprintf("%s", err)}, err
	}
	// Create Deployment
	deploymentsClient := proxy.clientset.AppsV1().Deployments(namespace)

	affinity := &v1.Affinity{}
	resources := in.GetResources()
	gpu := resources.GetGpus()
	if gpu.GetQuantity() > 0 {

		nsTerm := &v1.NodeSelectorTerm{
			MatchExpressions: []v1.NodeSelectorRequirement{
				{
					Key:      "accelerator",
					Operator: v1.NodeSelectorOpIn,
					Values:   []string{gpu.GetType()},
				},
			},
		}
		nsTerms := make([]v1.NodeSelectorTerm, 1)
		nsTerms[0] = *nsTerm
		affinity = &v1.Affinity{
			NodeAffinity: &v1.NodeAffinity{
				RequiredDuringSchedulingIgnoredDuringExecution: &v1.NodeSelector{
					NodeSelectorTerms: nsTerms,
				},
			},
		}
	}
	replicas := int32(1)
	var envVars []v1.EnvVar
	for _, e := range in.GetEnvironmentVariables() {
		envVars = append(envVars, v1.EnvVar{
			Name:  e.Key,
			Value: e.Value,
		})
	}
	containers := []v1.Container{
		{
			Name:         in.GetModuleId(),
			Image:        in.GetImage(),
			Ports:        containerPorts,
			Env:          envVars,
			Args:         in.GetCommandArguments(),
			VolumeMounts: volumeMounts,
			Resources: v1.ResourceRequirements{
				Limits: v1.ResourceList{
					v1.ResourceCPU:    resource.MustParse(fmt.Sprintf("%dm", resources.GetCpus())),
					v1.ResourceMemory: resource.MustParse(fmt.Sprintf("%dMi", resources.GetMemory())),
					"nvidia.com/gpu":  *resource.NewQuantity(int64(gpu.GetQuantity()), resource.DecimalSI),
				},
				Requests: v1.ResourceList{
					v1.ResourceCPU:    resource.MustParse(fmt.Sprintf("%dm", resources.GetCpus())), // NewQuantity(int64(resources.GetCpus()), resource.DecimalSI),
					v1.ResourceMemory: resource.MustParse(fmt.Sprintf("%dMi", resources.GetMemory())),
				},
			},
		},
	}

	if in.GetCommand() != "" {
		containers[0].Command = []string{in.GetCommand()}
	}

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      in.GetModuleId(),
			Namespace: namespace,
			Labels: map[string]string{
				"balticlsc/workloadselector": in.GetModuleId(),
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"balticlsc/workloadselector": in.GetModuleId(),
				},
			},
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"balticlsc/workloadselector": in.GetModuleId(),
					},
				},
				Spec: v1.PodSpec{
					Affinity:   affinity,
					Containers: containers,
					Volumes:    volumes,
				},
			},
		},
	}

	// Create Deployment
	log.Println("Creating deployment...")
	result, err := deploymentsClient.Create(context.TODO(), deployment, metav1.CreateOptions{})
	message := "pending"
	if err != nil {
		message = fmt.Sprintf("%s", err)
		log.Printf("Error creating deployment: %v", err)
	}
	//log.Printf("res: %+v", result)
	log.Printf("Created deployment %q.\n", result.GetObjectMeta().GetName())

	return &pb.ClusterStatusResponse{Status: pb.ClusterStatusResponse_PENDING, Message: message}, err

}

// CheckBalticModuleStatus implementation
func (s *server) CheckBalticModuleStatus(ctx context.Context, in *pb.Module) (*pb.ClusterStatusResponse, error) {
	log.Println("CheckBalticModuleStatus received for %s", in.GetModuleId())
	if proxy.debug {
		json, err := json.MarshalIndent(in, "", "  ")
		if err == nil {
			fmt.Printf("message: %s\n", json)
		}
	}
	namespace := getNSNameForBatch(in.GetBatchId())
	deploymentsClient := proxy.clientset.AppsV1().Deployments(namespace)
	result, err := deploymentsClient.Get(ctx, in.GetModuleId(), metav1.GetOptions{})
	message := "pending"
	status := pb.ClusterStatusResponse_PENDING
	if err != nil {
		message = fmt.Sprintf("%s", err)
		log.Printf("Error getting deployment: %v", err)
		return &pb.ClusterStatusResponse{Status: pb.ClusterStatusResponse_ERROR, Message: fmt.Sprintf("%s", err)}, err
	}
	log.Printf("res: %+v", result.Spec.Template.Spec.Volumes)
	for _, vol := range result.Spec.Template.Spec.Volumes {
		if vol.VolumeSource.PersistentVolumeClaim != nil {
			fmt.Printf("Vol: %+v\n", vol)
		}
	}
	for _, cond := range result.Status.Conditions {
		if cond.Type == "Available" {
			message = cond.Message
			if cond.Status == "False" {
				status = pb.ClusterStatusResponse_ERROR
			} else {
				status = pb.ClusterStatusResponse_ACTIVE
			}
		}
	}
	fmt.Printf("CheckBalticModuleStatus return %s.\n", message)
	return &pb.ClusterStatusResponse{Status: status, Message: message}, nil

}

// DisposeBalticModule implementation
func (s *server) DisposeBalticModule(ctx context.Context, in *pb.Module) (*pb.ClusterStatusResponse, error) {
	log.Printf("DisposeBalticModule Received for %s", in.GetModuleId())
	if proxy.debug {
		json, err := json.MarshalIndent(in, "", "  ")
		if err == nil {
			fmt.Printf("message: %s\n", json)
		}
	}
	message := "pending"
	status := pb.ClusterStatusResponse_PENDING
	namespace := getNSNameForBatch(in.GetBatchId())
	deployment, err := proxy.clientset.AppsV1().Deployments(namespace).Get(ctx, in.GetModuleId(), metav1.GetOptions{})
	retErr := err
	if err != nil {
		message = fmt.Sprintf("%s", err)
		status = pb.ClusterStatusResponse_ERROR
		retErr = err
		log.Printf("Error getting deployment: %s", err)
	}

	err = proxy.clientset.AppsV1().Deployments(namespace).Delete(ctx, in.GetModuleId(), metav1.DeleteOptions{})
	if err != nil {
		message = fmt.Sprintf("%s", err)
		status = pb.ClusterStatusResponse_ERROR
		retErr = err
		log.Printf("Error deleting deployment: %s", err)
	}

	err = proxy.clientset.CoreV1().Services(namespace).Delete(ctx, getServiceName(in.GetModuleId()), metav1.DeleteOptions{})
	if err != nil {
		message = fmt.Sprintf("%s", err)
		status = pb.ClusterStatusResponse_ERROR
		retErr = err
		log.Printf("error deleting service: %s", err)
	}
	for _, vol := range deployment.Spec.Template.Spec.Volumes {
		if vol.VolumeSource.PersistentVolumeClaim != nil {
			log.Printf("Deleting PersistentVolumeClaim: %s\n", vol.Name)
			err = proxy.clientset.CoreV1().PersistentVolumeClaims(namespace).Delete(ctx, vol.Name, metav1.DeleteOptions{})
			if err != nil {
				message = fmt.Sprintf("%s", err)
				status = pb.ClusterStatusResponse_ERROR
				retErr = err
				log.Printf("error deleting persistentvolumeclaim:", err)
			}
		} else if vol.VolumeSource.ConfigMap != nil {
			log.Printf("Deleting ConfigMap: %s\n", vol.Name)
			err = proxy.clientset.CoreV1().ConfigMaps(namespace).Delete(ctx, vol.Name, metav1.DeleteOptions{})
			if err != nil {
				message = fmt.Sprintf("%s", err)
				status = pb.ClusterStatusResponse_ERROR
				retErr = err
				log.Printf("error deleting configmap:", err)
			}

		}
	}
	return &pb.ClusterStatusResponse{Status: status, Message: message}, retErr

}

func cond2status(cond v1.NodeConditionType) pb.XClusterNode_NodeStatus {
	switch cond {
	case v1.NodeNetworkUnavailable:
		return pb.XClusterNode_NETWORK_UNAVAILABLE
	case v1.NodeMemoryPressure:
		return pb.XClusterNode_MEMORY_PRESSURE
	case v1.NodeDiskPressure:
		return pb.XClusterNode_DISK_PRESSURE
	case v1.NodePIDPressure:
		return pb.XClusterNode_PID_PRESSURE
	case v1.NodeReady:
		return pb.XClusterNode_READY
	}
	return pb.XClusterNode_READY
}

func resource2capacity(r v1.ResourceList) *pb.XCapacity {
	cpuq := r[v1.ResourceCPU]
	cpus, _ := cpuq.AsInt64()
	memq := r[v1.ResourceMemory]
	mem, _ := memq.AsInt64()
	mem = mem / (1024 * 1024)
	storageq := r[v1.ResourceEphemeralStorage]
	storage, _ := storageq.AsInt64()
	storage = storage / (1024 * 1024)
	gpuq := r["nvidia.com/gpu"]
	gpus, _ := gpuq.AsInt64()
	podq := r["pods"]
	pods, _ := podq.AsInt64()
	return &pb.XCapacity{
		Cpus:             int32(cpus),
		Memory:           int32(mem),
		Gpus:             int32(gpus),
		Pods:             int32(pods),
		EphemeralStorage: int32(storage),
	}
}

func formatQuotas(rq rancher.Quotas) *pb.XResourceQuota {
	quota := &pb.XResourceQuota{}
	q, err := resource.ParseQuantity(rq["requestsCpu"])
	if err == nil {
		quota.CpuRequest = int32(q.Value()) * 1000
	}
	q, err = resource.ParseQuantity(rq["limitsCpu"])
	if err == nil {
		quota.CpuLimit = int32(q.Value()) * 1000
	}
	q, err = resource.ParseQuantity(rq["limitsMemory"])
	if err == nil {
		quota.MemoryLimit = int32(q.Value() / (1024 * 1024))
	}
	q, err = resource.ParseQuantity(rq["requestsMemory"])
	if err == nil {
		quota.MemoryRequest = int32(q.Value() / (1024 * 1024))
	}
	q, err = resource.ParseQuantity(rq["requestsStorage"])
	if err == nil {
		quota.StorageRequest = uint32(q.Value() / (1024 * 1024 * 1024))
	}
	q, err = resource.ParseQuantity(rq["persistentVolumeClaims"])
	if err == nil {
		quota.PersistentVolumeClaims = int32(q.Value())
	}
	return quota
}

// GetClusterDescription implementation
func (s *server) GetClusterDescription(ctx context.Context, in *empty.Empty) (*pb.XClusterDescription, error) {
	//log.Printf("%s Received", runtime.Caller)
	//ctx, _ = context.WithTimeout(ctx, 5*time.Second)
	cluster := &pb.XClusterDescription{
		ClusterId:   proxy.clusterId,
		ProjectId:   proxy.projectId,
		ProjectName: proxy.projectName,
		Status:      pb.XClusterDescription_ONLINE,
	}
	project, err := proxy.rancherClient.GetProjectQuotas(proxy.clusterId + ":" + proxy.projectId)
	if err != nil {
		log.Printf("error getting projectquota from Rancher: %s\n", err)
	} else {
		cluster.Limit = formatQuotas(project.Project)
		cluster.UsedLimit = formatQuotas(project.Used)
	}
	nodes, err := proxy.clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		log.Printf("error listing nodes:", err)
	}
	for _, node := range nodes.Items {
		log.Printf("getting node %s", node.ObjectMeta.Name)
		n, err := proxy.clientset.CoreV1().Nodes().Get(ctx, node.ObjectMeta.Name, metav1.GetOptions{})
		if err != nil {
			log.Printf("error getting node %s: %s", node.ObjectMeta.Name, err)
		}
		nstatus := pb.XClusterNode_READY
		for _, cond := range n.Status.Conditions {
			if cond.Status == v1.ConditionTrue {
				nstatus = cond2status(cond.Type)
				log.Printf("node status %v", nstatus)
			}
		}

		cluster.Nodes = append(cluster.Nodes, &pb.XClusterNode{
			Name:                node.ObjectMeta.Name,
			Os:                  node.ObjectMeta.Labels["kubernetes.io/os"],
			Architecture:        node.ObjectMeta.Labels["kubernetes.io/arch"],
			KernelVersion:       n.Status.NodeInfo.KernelVersion,
			OsImage:             n.Status.NodeInfo.OSImage,
			OrchestratorType:    "BalticLSC Platform",
			OrchestratorVersion: n.Status.NodeInfo.KubeletVersion,
			GpuType:             node.ObjectMeta.Labels["accelerator"],
			Unschedulable:       n.Spec.Unschedulable,
			Capacity:            resource2capacity(n.Status.Capacity),
			Allocatable:         resource2capacity(n.Status.Allocatable),
			Status:              nstatus,
		})
	}
	return cluster, nil
}

func main() {
	// Get configuration from environment
	log.Println("Starting cluster-proxy...")
	proxy.clusterId = os.Getenv("RANCHER_CLUSTER_ID") //"c-tmfxj"
	proxy.projectId = os.Getenv("RANCHER_PROJECT_ID") // "p-vxl6k"
	rancherURL := os.Getenv("RANCHER_URL")            //"https://k8s.ice.ri.se"
	token := os.Getenv("RANCHER_TOKEN")               // "token-k9lqc:mbfmedchkl9v5epqb5tn4wxkg7n9zjmlm2c6r2z8h7bn4gk5dsjbsh"
	listenPort := ":" + os.Getenv("LISTEN_PORT")      // 50051
	kubeConfig := os.Getenv("KUBECONFIG")
	debug := os.Getenv("DEBUG")
	insecure := os.Getenv("TLS_INSECURE")

	if proxy.clusterId == "" {
		panic("env RANCHER_CLUSTER_ID not set!")
	} else if proxy.projectId == "" {
		panic("env RANCHER_PROJECT_ID not set!")
	} else if rancherURL == "" {
		panic("env RANCHER_URL not set!")
	} else if token == "" {
		panic("env RANCHER_TOKEN not set!")
	} else if listenPort == ":" {
		listenPort = ":50051"
	}
	if kubeConfig == "" {
		kubeConfig = "/config"
	}
	if debug == "" {
		proxy.debug = false
	} else {
		proxy.debug = true
	}

	var kubeconfig *string

	//if home := homedir.HomeDir(); home != "" {
	//	kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	//} else {
	kubeconfig = flag.String("kubeconfig", kubeConfig, "(optional) absolute path to the kubeconfig file")
	//}
	flag.Parse()

	// use the current context in kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		panic(err.Error())
	}
	if insecure == "true" {
		config.TLSClientConfig.Insecure = true
	} else {
		config.TLSClientConfig.Insecure = false
	}
	// create the client
	proxy.clientset, err = kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}
	proxy.client, err = dynamic.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}
	dc, err := discovery.NewDiscoveryClientForConfig(config)
	if err != nil {
		panic(err.Error())
	}
	proxy.rancherClient = rancher.NewClient(rancherURL, token)
	resty.SetTLSClientConfig(&tls.Config{InsecureSkipVerify: true})

	// Get project name
	project, err := proxy.rancherClient.GetProjectDetail(proxy.clusterId + ":" + proxy.projectId)
	if err != nil {
		panic(err.Error())
	}
	log.Printf("Rancher project name: %s\n", project.Name)
	proxy.projectName = project.Name

	// Prepare a RESTMapper to find GVR
	proxy.mapper = restmapper.NewDeferredDiscoveryRESTMapper(memory.NewMemCacheClient(dc))
	lis, err := net.Listen("tcp", listenPort)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	log.Printf("Listening on %s\n", listenPort)
	s := grpc.NewServer()

	pb.RegisterClusterProxyServer(s, &server{})
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}

}
