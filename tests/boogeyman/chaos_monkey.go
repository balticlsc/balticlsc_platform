package boogeyman

import (
	"fmt"
	"math/rand"
	"strconv"
	"time"
)

func StartPodChaosMonkey(boogeyman *Boogeyman, startTime int, fractionOfPodsToKill float64) {
	go func() {
		fmt.Println("-> chaos monkey is waiting to " + strconv.Itoa(startTime) + " seconds to start")
		time.Sleep(time.Duration(startTime) * time.Second)
		fmt.Println("-> chaos monkey is now active (killing pods makes me feel good!)")
		go func() {
			for {
				t := rand.ExpFloat64() / 0.1
				timeToNextExecution := time.Duration(t) * time.Second
				fmt.Println("-> chaos monkey, time to next pod execution " + timeToNextExecution.String())
				time.Sleep(timeToNextExecution)
				numberOfPods := boogeyman.NumberOfPods()
				mu := float64(numberOfPods) * fractionOfPodsToKill
				sigma := mu * 0.5
				numberOfPodsToKill := int(rand.NormFloat64()*sigma + mu)
				fmt.Printf("-> chais monkey, trying to kill %d pods", numberOfPodsToKill)
				boogeyman.DeleteRandomPod(rand.Float32() < 0.5, numberOfPodsToKill) // 50 % chance of using the force flag
			}
		}()
	}()
}

func StartDeploymentChaosMonkey(boogeyman *Boogeyman, startTime int) {
	go func() {
		fmt.Println("-> chaos monkey is waiting to " + strconv.Itoa(startTime) + " seconds to start")
		time.Sleep(time.Duration(startTime) * time.Second)
		fmt.Println("-> chaos monkey is now active (killing deployments makes me feel even greater!)")
		go func() {
			for {
				t := rand.ExpFloat64() / 0.1
				timeToNextExecution := time.Duration(t) * time.Second
				fmt.Println("-> chaos monkey, time to next deployment execution " + timeToNextExecution.String())
				time.Sleep(timeToNextExecution)
				boogeyman.DeleteRandomDeployment(rand.Float32() < 0.5) // 50 % chance of using the force flag
			}
		}()
	}()
}
