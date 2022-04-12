package boogeyman

import (
	"context"
	"flag"
	"fmt"
	"github.com/rs/xid"
	apps "k8s.io/api/apps/v1"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"math/rand"
	"path/filepath"
)

type Boogeyman struct {
	namespace      string
	client         dynamic.Interface
	clientset      *kubernetes.Clientset
	counter        int
	deployments    map[string]bool
	pvcs           map[string]bool
	pods           map[string]bool
	commandChannel chan command
}

const (
	createPVC              = iota
	deploy                 = iota
	deleteAllPods          = iota
	deleteRandomPod        = iota
	deleteAllDeployments   = iota
	deleteRandomDeployment = iota
	deploymentAdded        = iota
	deploymentDeleted      = iota
	pvcAdded               = iota
	pvcDeleted             = iota
	podAdded               = iota
	podDeleted             = iota
)

type command struct {
	cmdType        int
	name           string
	containerImage string
	instances      int
	pvcName        string
	pvcSize        int
	force          bool
}

func newCreatewPVCCommand(pvcName string, pvcSize int) command {
	return command{createPVC, "", "", 0, pvcName, pvcSize, false}
}

func newDeployCommand(name string, containerImage string, instances int, pvcName string, pvcSize int) command {
	return command{deploy, name, containerImage, instances, pvcName, pvcSize, false}
}

func newDeleteAllPodsCommand(force bool) command {
	return command{deleteAllPods, "", "", 0, "", 0, force}
}

func newDeleteRandomPodCommand(force bool, instances int) command {
	return command{deleteRandomPod, "", "", instances, "", 0, force}
}

func newDeleteAllDeploymentsCommand(force bool) command {
	return command{deleteAllDeployments, "", "", 0, "", 0, force}
}

func newDeleteRandomDeploymentCommand(force bool) command {
	return command{deleteRandomDeployment, "", "", 0, "", 0, force}
}

func newUpdateCommand(cmdType int, name string) command {
	return command{cmdType, name, "", 0, "", 0, false}
}

func NewBoogeyman(namespace string) (*Boogeyman, error) {
	boogeyman := &Boogeyman{}

	boogeyman.namespace = namespace
	boogeyman.deployments = make(map[string]bool)
	boogeyman.pvcs = make(map[string]bool)
	boogeyman.pods = make(map[string]bool)
	boogeyman.commandChannel = make(chan command)

	var err error
	boogeyman.client, boogeyman.clientset, err = boogeyman.setupClient()
	if err != nil {
		return nil, err
	}

	err = boogeyman.startDeploymentWatcher()
	if err != nil {
		return nil, err
	}

	err = boogeyman.startPVCWatcher()
	if err != nil {
		return nil, err
	}

	err = boogeyman.startPodWatcher()
	if err != nil {
		return nil, err
	}

	go boogeyman.masterWorker()

	return boogeyman, nil
}

func (boogeyman *Boogeyman) masterWorker() error {
	for {
		select {
		case cmd := <-boogeyman.commandChannel:
			switch cmd.cmdType {
			case createPVC:
				fmt.Println("-> create pvc <" + cmd.pvcName + ">")
				err := boogeyman.createPersistentVolumeClaim(cmd.pvcName, cmd.pvcSize)
				if err != nil {
					fmt.Println(err)
				}
			case deploy:
				fmt.Println("-> deploying <" + cmd.name + ">")
				if cmd.pvcName == "" {
					err := boogeyman.deployAndCreatePVC(cmd.name, cmd.containerImage, cmd.instances, cmd.pvcName, cmd.pvcSize)
					if err != nil {
						fmt.Println(err)
					}
				} else {
					err := boogeyman.deploy(cmd.name, cmd.containerImage, cmd.instances, cmd.pvcName)
					if err != nil {
						fmt.Println(err)
					}
				}
			case deleteAllPods:
				if cmd.force {
					fmt.Println("-> force deleting all pods")
				} else {
					fmt.Println("-> deleting all pods")
				}
				err := boogeyman.deleteAllPods(cmd.force)
				if err != nil {
					fmt.Println(err)
				}
			case deleteRandomPod:
				if cmd.force {
					fmt.Println("-> force deleting random pod")
				} else {
					fmt.Println("-> deleting random pod")
				}
				err := boogeyman.deleteRandomPod(cmd.force, cmd.instances)
				if err != nil {
					fmt.Println(err)
				}
			case deleteAllDeployments:
				if cmd.force {
					fmt.Println("-> force deleting all deployments")
				} else {
					fmt.Println("-> deleting all deployments")
				}
				err := boogeyman.deleteAllDeployments(cmd.force)
				if err != nil {
					fmt.Println(err)
				}
			case deleteRandomDeployment:
				if cmd.force {
					fmt.Println("-> force deleting random deployment")
				} else {
					fmt.Println("-> deleting random deployment")
				}
				err := boogeyman.deleteRandomDeployment(cmd.force)
				if err != nil {
					fmt.Println(err)
				}
			case deploymentAdded:
				fmt.Println("-> deployment added <" + cmd.name + ">")
				boogeyman.deployments[cmd.name] = true
			case deploymentDeleted:
				fmt.Println("-> deployment deleted <" + cmd.name + ">")
				delete(boogeyman.deployments, cmd.name)
			case pvcAdded:
				fmt.Println("-> pvc added <" + cmd.name + ">")
				boogeyman.pvcs[cmd.name] = true
			case pvcDeleted:
				fmt.Println("-> pvc deleted <" + cmd.name + ">")
				delete(boogeyman.pvcs, cmd.name)
			case podAdded:
				fmt.Println("-> pod added <" + cmd.name + ">")
				boogeyman.pods[cmd.name] = true
			case podDeleted:
				fmt.Println("-> pod deleted <" + cmd.name + ">")
				delete(boogeyman.pods, cmd.name)
			}
		}
	}
}

func (boogeyman *Boogeyman) setupClient() (dynamic.Interface, *kubernetes.Clientset, error) {
	var kubeconfig *string
	if home := homedir.HomeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	flag.Parse()

	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		return nil, nil, err
	}

	client, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, nil, err
	}

	clientset, err := kubernetes.NewForConfig(config)

	return client, clientset, nil
}

func (boogeyman *Boogeyman) Cleanup() error {
	return boogeyman.deleteAllDeployments(true)
}

func (boogeyman *Boogeyman) NumberOfPods() int {
	return len(boogeyman.pods)
}

func (boogeyman *Boogeyman) NumberOfDeployments() int {
	return len(boogeyman.deployments)
}

func (boogeyman *Boogeyman) DeleteRandomPod(force bool, instances int) {
	boogeyman.commandChannel <- newDeleteRandomPodCommand(force, instances)
}

func (boogeyman *Boogeyman) deleteRandomPod(force bool, instances int) error {
	deploymentResDep := schema.GroupVersionResource{Version: "v1", Resource: "pods"}

	var deleteOptions metav1.DeleteOptions
	deletePolicy := metav1.DeletePropagationForeground
	if force {
		gracePeriod := int64(0)
		deleteOptions = metav1.DeleteOptions{
			GracePeriodSeconds: &gracePeriod,
			PropagationPolicy:  &deletePolicy,
		}
	} else {
		deleteOptions = metav1.DeleteOptions{
			PropagationPolicy: &deletePolicy,
		}
	}

	// Note that there is no guarantee that all pods will be deleted since they may be selected multiple times
	for i := 0; i < instances; i++ {
		numberOfPods := len(boogeyman.pods)
		var podList []string
		for podName, _ := range boogeyman.pods {
			podList = append(podList, podName)
		}

		//randomPodIndex := rand.Intn(numberOfPods - 1)
		randomPodIndex := rand.Intn(numberOfPods)
		randomPod := podList[randomPodIndex]

		if err := boogeyman.client.Resource(deploymentResDep).Namespace(boogeyman.namespace).Delete(context.TODO(), randomPod, deleteOptions); err != nil {
			fmt.Println(err)
			fmt.Println("Failed to delete pod: " + randomPod)
		}
	}

	return nil
}

func (boogeyman *Boogeyman) DeleteAllPods(force bool) {
	boogeyman.commandChannel <- newDeleteAllPodsCommand(force)
}

func (boogeyman *Boogeyman) deleteAllPods(force bool) error {
	deploymentResDep := schema.GroupVersionResource{Version: "v1", Resource: "pods"}

	var deleteOptions metav1.DeleteOptions
	deletePolicy := metav1.DeletePropagationForeground
	if force {
		gracePeriod := int64(0)
		deleteOptions = metav1.DeleteOptions{
			GracePeriodSeconds: &gracePeriod,
			PropagationPolicy:  &deletePolicy,
		}
	} else {
		deleteOptions = metav1.DeleteOptions{
			PropagationPolicy: &deletePolicy,
		}
	}

	for podName, _ := range boogeyman.pods {
		if err := boogeyman.client.Resource(deploymentResDep).Namespace(boogeyman.namespace).Delete(context.TODO(), podName, deleteOptions); err != nil {
			fmt.Println(err)
			fmt.Println("Failed to delete pod: " + podName)
		}
	}

	return nil
}

func (boogeyman *Boogeyman) DeleteAllDeployments(force bool) {
	boogeyman.commandChannel <- newDeleteAllDeploymentsCommand(force)
}

func (boogeyman *Boogeyman) deleteAllDeployments(force bool) error {
	deploymentResDep := schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"}
	deploymentResPVC := schema.GroupVersionResource{Version: "v1", Resource: "persistentvolumeclaims"}

	var deleteOptions metav1.DeleteOptions
	deletePolicy := metav1.DeletePropagationForeground
	if force {
		gracePeriod := int64(0)
		deleteOptions = metav1.DeleteOptions{
			GracePeriodSeconds: &gracePeriod,
			PropagationPolicy:  &deletePolicy,
		}
	} else {
		deleteOptions = metav1.DeleteOptions{
			PropagationPolicy: &deletePolicy,
		}
	}

	for deploymentName, _ := range boogeyman.deployments {
		if err := boogeyman.client.Resource(deploymentResDep).Namespace(boogeyman.namespace).Delete(context.TODO(), deploymentName, deleteOptions); err != nil {
			fmt.Println("Failed to delete deployment: " + deploymentName)
		}
	}

	for pvcName, _ := range boogeyman.pvcs {
		if err := boogeyman.client.Resource(deploymentResPVC).Namespace(boogeyman.namespace).Delete(context.TODO(), pvcName, deleteOptions); err != nil {
			fmt.Println("Failed to delete pvc: " + pvcName)
		}
	}

	return nil
}

func (boogeyman *Boogeyman) DeleteRandomDeployment(force bool) {
	boogeyman.commandChannel <- newDeleteRandomDeploymentCommand(force)
}

func (boogeyman *Boogeyman) deleteRandomDeployment(force bool) error {
	deploymentResDep := schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"}

	var deleteOptions metav1.DeleteOptions
	deletePolicy := metav1.DeletePropagationForeground
	if force {
		gracePeriod := int64(0)
		deleteOptions = metav1.DeleteOptions{
			GracePeriodSeconds: &gracePeriod,
			PropagationPolicy:  &deletePolicy,
		}
	} else {
		deleteOptions = metav1.DeleteOptions{
			PropagationPolicy: &deletePolicy,
		}
	}

	numberOfDeployments := len(boogeyman.deployments)
	var deploymentList []string
	for deploymentName, _ := range boogeyman.deployments {
		deploymentList = append(deploymentList, deploymentName)
	}

	randomDeploymentIndex := rand.Intn(numberOfDeployments)
	randomDeployment := deploymentList[randomDeploymentIndex]

	if err := boogeyman.client.Resource(deploymentResDep).Namespace(boogeyman.namespace).Delete(context.TODO(), randomDeployment, deleteOptions); err != nil {
		fmt.Println("Failed to delete deployment: " + randomDeployment)
	}

	return nil
}

func (boogeyman *Boogeyman) Deploy(containerImage string, instances int, pvcSize int) {
	guid := xid.New()
	name := "boogeyman" + "-" + guid.String()
	boogeyman.commandChannel <- newDeployCommand(name, containerImage, instances, "", pvcSize)
}

func (boogeyman *Boogeyman) DeploySharedPVC(containerImage string, instances int, pvcName string) {
	guid := xid.New()
	name := "boogeyman" + "-" + guid.String()
	boogeyman.commandChannel <- newDeployCommand(name, containerImage, instances, pvcName, 0)
}

func (boogeyman *Boogeyman) deploy(name string, containerImage string, instances int, pvcName string) error {
	err := boogeyman.createDeployment(name, containerImage, instances, pvcName)
	if err != nil {
		return err
	}

	return nil
}

func (boogeyman *Boogeyman) deployAndCreatePVC(name string, containerImage string, instances int, pvcName string, pvcSize int) error {
	uniquePVCName := name + "-claim"

	err := boogeyman.createPersistentVolumeClaim(uniquePVCName, pvcSize)
	if err != nil {
		fmt.Println("-> failed to create pvc (may be it already exists)")
	}

	err = boogeyman.createDeployment(name, containerImage, instances, uniquePVCName)
	if err != nil {
		return err
	}

	return nil
}

func (boogeyman *Boogeyman) CreatePersistentVolumeClaim(pvcName string, pvcSize int) {
	boogeyman.commandChannel <- newCreatewPVCCommand(pvcName, pvcSize)
}

func (boogeyman *Boogeyman) createPersistentVolumeClaim(pvcName string, pvcSize int) error {
	pvcSizeStr := fmt.Sprintf("%dGi", pvcSize)
	deploymentRes := schema.GroupVersionResource{Version: "v1", Resource: "persistentvolumeclaims"}

	accessModes := []string{}
	//	accessModes = append(accessModes, "ReadWriteOnce")
	accessModes = append(accessModes, "ReadWriteMany")

	deployment := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "PersistentVolumeClaim",
			"metadata": map[string]interface{}{
				"name": pvcName,
			},
			"spec": map[string]interface{}{
				"storageClassName": "rook-ceph-fs",
				"accessModes":      accessModes,
				"resources": map[string]interface{}{
					"requests": map[string]interface{}{
						"storage": pvcSizeStr,
					},
				},
			},
		},
	}

	_, err := boogeyman.client.Resource(deploymentRes).Namespace(boogeyman.namespace).Create(context.TODO(), deployment, metav1.CreateOptions{})
	if err != nil {
		return err
	}

	return nil
}

func (boogeyman *Boogeyman) createDeployment(name string, containerImage string, instances int, pvcClaimName string) error {
	deploymentRes := schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"}

	deployment := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "apps/v1",
			"kind":       "Deployment",
			"metadata": map[string]interface{}{
				"name": name + "-deployment",
			},
			"spec": map[string]interface{}{
				"replicas": instances,
				"selector": map[string]interface{}{
					"matchLabels": map[string]interface{}{
						"app": name,
					},
				},
				"template": map[string]interface{}{
					"metadata": map[string]interface{}{
						"labels": map[string]interface{}{
							"app": name,
						},
					},

					"spec": map[string]interface{}{
						"volumes": []map[string]interface{}{
							{
								"name": "storage",
								"persistentVolumeClaim": map[string]interface{}{
									"claimName": pvcClaimName,
								},
							},
						},
						"containers": []map[string]interface{}{
							{
								"name":  name,
								"image": containerImage,
								"volumeMounts": []map[string]interface{}{
									{
										"name":      "storage",
										"mountPath": "/storage",
									},
								},
								"env": []map[string]interface{}{
									{
										"name":  "STORAGE_PATH",
										"value": "/storage",
									},
									{
										"name":  "FILE_SIZE",
										"value": "1", // 1 GiB
									},
								},
							},
						},
					},
				},
			},
		},
	}

	_, err := boogeyman.client.Resource(deploymentRes).Namespace(boogeyman.namespace).Create(context.TODO(), deployment, metav1.CreateOptions{})
	if err != nil {
		return err
	}

	return nil
}

func (boogeyman *Boogeyman) startDeploymentWatcher() error {
	listOptions := metav1.ListOptions{
		LabelSelector: "",
		FieldSelector: "",
	}
	deploymentWatcher, err := boogeyman.clientset.AppsV1().Deployments(boogeyman.namespace).Watch(context.TODO(), listOptions)
	if err != nil {
		return err
	}

	ch := deploymentWatcher.ResultChan()
	go func() {
		for event := range ch {
			dep, _ := event.Object.(*apps.Deployment)
			switch event.Type {
			case watch.Added:
				boogeyman.commandChannel <- newUpdateCommand(deploymentAdded, dep.Name)
			case watch.Deleted:
				boogeyman.commandChannel <- newUpdateCommand(deploymentDeleted, dep.Name)
			}
		}
	}()

	return nil
}

func (boogeyman *Boogeyman) startPVCWatcher() error {
	api := boogeyman.clientset.CoreV1()
	listOptions := metav1.ListOptions{
		LabelSelector: "",
		FieldSelector: "",
	}

	pvcWatcher, err := api.PersistentVolumeClaims(boogeyman.namespace).Watch(context.TODO(), listOptions)
	if err != nil {
		return err
	}

	ch := pvcWatcher.ResultChan()
	go func() {
		for event := range ch {
			pvc, _ := event.Object.(*v1.PersistentVolumeClaim)
			switch event.Type {
			case watch.Added:
				boogeyman.commandChannel <- newUpdateCommand(pvcAdded, pvc.Name)
			case watch.Deleted:
				boogeyman.commandChannel <- newUpdateCommand(pvcDeleted, pvc.Name)
			}
		}
	}()

	return nil
}

func (boogeyman *Boogeyman) startPodWatcher() error {
	api := boogeyman.clientset.CoreV1()
	listOptions := metav1.ListOptions{
		LabelSelector: "",
		FieldSelector: "",
	}

	pvcWatcher, err := api.Pods(boogeyman.namespace).Watch(context.TODO(), listOptions)
	if err != nil {
		return err
	}

	ch := pvcWatcher.ResultChan()
	go func() {
		for event := range ch {
			pod, _ := event.Object.(*v1.Pod)
			switch event.Type {
			case watch.Added:
				boogeyman.commandChannel <- newUpdateCommand(podAdded, pod.Name)
			case watch.Deleted:
				boogeyman.commandChannel <- newUpdateCommand(podDeleted, pod.Name)
			}
		}
	}()

	return nil
}
