package boogeyman

import (
	"fmt"
	"github.com/rs/xid"
	"testing"
	"time"
)

// Test scenario:
// --------------
// Purpose: This is just a simple test to make sure the producer and consumer is working.
//
func TestSmoke1(t *testing.T) {
	fmt.Println("Starting Smoke Test 1")

	namespace := "storage-test"

	producerImage := "johan/boogeyman_producer"
	consumerImage := "johan/boogeyman_consumer"

	consumerInstances := 1
	producerInstances := 1

	guid := xid.New()
	sharedPVCName := "sharedpvctest-" + guid.String()
	sharedPVCSize := 100 // 100 GiB

	boogeyman, err := NewBoogeyman(namespace)
	CheckError(err)

	HandleCtrlC(boogeyman) // Delete all created resources when Ctrl+C is pressed

	// Create a PVC
	boogeyman.CreatePersistentVolumeClaim(sharedPVCName, sharedPVCSize)

	// Create Producers and Consumers
	boogeyman.DeploySharedPVC(producerImage, producerInstances, sharedPVCName)
	boogeyman.DeploySharedPVC(consumerImage, consumerInstances, sharedPVCName)

	done := make(chan bool, 1)
	<-done
}

// Test scenario:
// --------------
// Purpose: Similar setup as SmokeTest1, but with more instances and a ChaosMonkey.
//
func TestSmoke2(t *testing.T) {
	fmt.Println("Starting Smoke Test 2")

	namespace := "storage-test"

	producerImage := "johan/boogeyman_producer"
	consumerImage := "johan/boogeyman_consumer"

	consumerInstances := 40
	producerInstances := 20

	guid := xid.New()
	sharedPVCName := "sharedpvctest-" + guid.String()
	sharedPVCSize := 2000 // 1000 GiB

	boogeyman, err := NewBoogeyman(namespace)
	CheckError(err)

	HandleCtrlC(boogeyman) // Delete all created resources when Ctrl+C is pressed

	// Start the Chaos Monkey after 80 s, which then tries to kill 5% of all pods.
	StartPodChaosMonkey(boogeyman, 60, 0.1)

	// Create a PVC
	boogeyman.CreatePersistentVolumeClaim(sharedPVCName, sharedPVCSize)

	// Create Producers and Consumers
	boogeyman.DeploySharedPVC(producerImage, producerInstances, sharedPVCName)
	boogeyman.DeploySharedPVC(consumerImage, consumerInstances, sharedPVCName)

	done := make(chan bool, 1)
	<-done
}

// Test scenario:
// --------------
// Purpose: Create a bunch of deployments and then randomly delete some deployment and then restart it.
//
func TestSmoke3(t *testing.T) {
	fmt.Println("Starting Smoke Test 3")

	namespace := "storage-test"

	producerImage := "johan/boogeyman_producer"
	producerInstances := 5
	pvcSize := 2000 // 1000 GiB
	numberOfDeployments := 10

	boogeyman, err := NewBoogeyman(namespace)
	CheckError(err)

	HandleCtrlC(boogeyman) // Delete all created resources when Ctrl+C is pressed

	// Start the Chaos Monkey after 80 s, which then tries to a random deployment
	StartDeploymentChaosMonkey(boogeyman, 30)

	for i := 0; i < numberOfDeployments; i++ {
		boogeyman.Deploy(producerImage, producerInstances, pvcSize)
	}

	done := make(chan bool, 1)
	<-done
}

// Test scenario:
// --------------
// Purpose: Same as SmokeTest3, but now create new a deployment when a deploy is deleted.
//
func TestSmoke4(t *testing.T) {
	fmt.Println("Starting Smoke Test 4")

	namespace := "storage-test"

	producerImage := "johan/boogeyman_producer"
	producerInstances := 5
	pvcSize := 2000 // 2000 GiB
	numberOfDeployments := 10

	boogeyman, err := NewBoogeyman(namespace)
	CheckError(err)

	HandleCtrlC(boogeyman) // Delete all created resources when Ctrl+C is pressed

	// Start the Chaos Monkey after 80 s, which then tries to a random deployment
	StartDeploymentChaosMonkey(boogeyman, 30)

	for i := 0; i < numberOfDeployments; i++ {
		boogeyman.Deploy(producerImage, producerInstances, pvcSize)
	}

	// Undo the evil work done of the wicked ChaosMonkey!
	go func() {
		for {
			sleepTime := time.Duration(1) * time.Second // 1 seconds
			time.Sleep(sleepTime)
			if boogeyman.NumberOfDeployments() < numberOfDeployments {
				fmt.Println("-> recreating deleted deployment")
				boogeyman.Deploy(producerImage, producerInstances, pvcSize)
			}
		}

	}()

	done := make(chan bool, 1)
	<-done
}
