package server

import (
	"crypto/tls"
	rancher "icekube/admission-controller/rancher-go/client"
	"io/ioutil"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"net/http"
	"os"
)

func SetupRancher() rancher.Client {
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	return rancher.NewClient(rancherURL, token)
}

func SetupRancherWithToken(token string) rancher.Client {
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	return rancher.NewClient(rancherURL, token)
}

func CreateProject(rancherClient rancher.Client, projectName string) (string, error) {
	return rancherClient.CreateProject(clusterID, rancher.Project{Name: projectName})
}

func CreateUser(rancherClient rancher.Client, username string, password string) (string, error) {
	return rancherClient.AddUser(username, password)
}

func AddUserAsMember(rancherClient rancher.Client, projectID string, userID string) error {
	return rancherClient.AddProjectMember(projectID, rancher.Member{UserID: userID, RoleTemplateID: "project-member", Type: rancher.MemberTypeUser})
}

func AddUserAsOwner(rancherClient rancher.Client, projectID string, userID string) error {
	return rancherClient.AddProjectMember(projectID, rancher.Member{UserID: userID, RoleTemplateID: "project-owner", Type: rancher.MemberTypeUser})
}

func RemoveUser(rancherClient rancher.Client, userID string) error {
	if err := rancherClient.DeleteUser(userID); err != nil {
		return err
	}

	return nil
}

func RemoveProject(rancherClient rancher.Client, projectID string) error {
	if err := rancherClient.DeleteProject(projectID); err != nil {
		return err
	}

	return nil
}

func SetupClientGo(rancherClient rancher.Client, username string, password string) (*kubernetes.Clientset, error) {
	token, err := rancherClient.GetUserToken(username, password)
	if err != nil {
		return nil, err
	}

	return SetupClientGoWithToken(rancherClient, token)
}

func SetupClientGoWithToken(rancherClient rancher.Client, token string) (*kubernetes.Clientset, error) {
	rancherClientNewUser := SetupRancherWithToken(token)

	// Get Kubeconfig for the new user
	kubeconfig, err := rancherClientNewUser.GetKubeConfig(clusterID)
	if err != nil {
		return nil, err
	}

	tmpFile, err := ioutil.TempFile(os.TempDir(), "kubeconfig")
	if err != nil {
		return nil, err
	}

	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.Write([]byte(kubeconfig))
	if err != nil {
		return nil, err
	}

	// Setup Kubernetes client-go
	config, err := clientcmd.BuildConfigFromFlags("", tmpFile.Name())
	if err != nil {
		return nil, err
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return clientset, nil
}
