package server

import (
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
	"time"
)

func TestAdmissionController(t *testing.T) {
	username := "username"
	password := "password"

	// Create a user
	rancherClient := SetupRancher()
	userID, err := CreateUser(rancherClient, username, password)
	assert.Nil(t, err)
	time.Sleep(delay)

	projectID, err := CreateProject(rancherClient, username)
	assert.Nil(t, err)

	err = AddUserAsMember(rancherClient, projectID, userID)
	assert.Nil(t, err)

	// Setup Kubernetes
	clientset, err := SetupClientGo(rancherClient, username, password)

	// Now create a namespace
	nsSpec := &v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "testnamespace"}}
	_, err = clientset.CoreV1().Namespaces().Create(nsSpec)
	assert.Nil(t, err)

	clientsetAdmin, err := SetupClientGoWithToken(rancherClient, token)

	// Fetch the namespace
	namespace, err := clientsetAdmin.CoreV1().Namespaces().Get("testnamespace", metav1.GetOptions{})
	assert.Nil(t, err)

	// We expect field.cattle.io/projectId:CLUSTER_UD:USER_ID to be set
	foundFieldCattleIOKey := false
	value := ""
	for k, v := range namespace.Annotations {
		if k == "field.cattle.io/projectId" {
			foundFieldCattleIOKey = true
			value = v
		}
	}

	assert.True(t, foundFieldCattleIOKey)
	assert.True(t, value == projectID)

	// Delete the namespace
	err = clientsetAdmin.CoreV1().Namespaces().Delete("testnamespace", nil)
	assert.Nil(t, err)

	// Cleanup
	time.Sleep(delay)

	RemoveUser(rancherClient, userID)
	RemoveProject(rancherClient, projectID)
}

func TestAdmissionControllerMultipleProject(t *testing.T) {
	username := "username2"
	password := "password2"

	// Create a user
	rancherClient := SetupRancher()
	userID, err := CreateUser(rancherClient, username, password)
	assert.Nil(t, err)
	time.Sleep(delay)

	projectID1, err := CreateProject(rancherClient, username)
	assert.Nil(t, err)

	err = AddUserAsMember(rancherClient, projectID1, userID)
	assert.Nil(t, err)

	projectID2, err := CreateProject(rancherClient, username+"_2")
	assert.Nil(t, err)

	err = AddUserAsMember(rancherClient, projectID2, userID)
	assert.Nil(t, err)

	// Setup Kubernetes
	clientset, err := SetupClientGo(rancherClient, username, password)

	// Now create a namespace
	nsSpec := &v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "testnamespace2"}}
	_, err = clientset.CoreV1().Namespaces().Create(nsSpec)
	assert.Nil(t, err)

	clientsetAdmin, err := SetupClientGoWithToken(rancherClient, token)

	// Fetch the namespace
	namespace, err := clientsetAdmin.CoreV1().Namespaces().Get("testnamespace2", metav1.GetOptions{})
	assert.Nil(t, err)

	// We expect field.cattle.io/projectId:CLUSTER_UD:USER_ID to be set
	foundFieldCattleIOKey := false
	for k, _ := range namespace.Annotations {
		if k == "field.cattle.io/projectId" {
			foundFieldCattleIOKey = true
		}
	}

	assert.False(t, foundFieldCattleIOKey)

	// Delete the namespace
	err = clientsetAdmin.CoreV1().Namespaces().Delete("testnamespace2", nil)
	assert.Nil(t, err)

	// Cleanup
	time.Sleep(delay)

	RemoveUser(rancherClient, userID)
	RemoveProject(rancherClient, projectID1)
	RemoveProject(rancherClient, projectID2)
}
