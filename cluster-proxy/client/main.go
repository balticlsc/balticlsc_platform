/*
 * Test client for ClusterProxy
 *
 */

package main

import (
	pb "cluster-proxy/clusterproxy"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/golang/protobuf/ptypes/empty"
	"google.golang.org/grpc"
)

const (
	defaultName = "client"
	bid         = "123e4567-e89b-12d3-a456-426655440000"
	mid         = "b-344223ab-3344-a223-f223-fff223423553"
)

var commands map[string]func() error
var clusterProxy pb.ClusterProxyClient
var ctx context.Context

func prepareWorkspace() error {
	r, err := clusterProxy.PrepareWorkspace(ctx,
		&pb.XWorkspace{
			BatchId: bid,
			Quota: &pb.XWorkspaceQuota{
				Cpus:    1000, // mCPUs
				Memory:  256,  // MB
				Storage: 10,   // GiB
				Gpus:    1,
			},
		},
	)
	if err != nil {
		log.Fatalf("PrepareWorkspace failed: %v", err)
	}
	fmt.Printf("Return: %+v\n", r)
	return err
}
func checkWorkspaceStatus() error {
	r, err := clusterProxy.CheckWorkspaceStatus(ctx,
		&pb.BatchId{
			Id: bid,
		},
	)
	if err != nil {
		log.Fatalf("checkWorkspaceStatus failed: %v", err)
	}
	fmt.Printf("Return: %+v\n", r)
	return err
}
func purgeWorkspace() error {
	r, err := clusterProxy.PurgeWorkspace(ctx,
		&pb.BatchId{
			Id: bid,
		},
	)
	if err != nil {
		log.Fatalf("purgeWorkspace failed: %v", err)
	}
	fmt.Printf("purgeWorkspace returned: %+v\n", r)
	return err
}

/*
  "Image": "balticlsc/balticmodulemanager:latest",
  "EnvironmentVariables": [
    {
      "Key": "ModuleManagerPort",
      "Value": "7301"
    },
    {
      "Key": "BatchManagerUrl",
      "Value": "https://host.docker.internal:7001"
    }
  ],
  "PortMappings": [
    {
      "ContainerPort": 7301,
      "PublishedPort": 7301
    },
    {
      "ContainerPort": 7300,
      "PublishedPort": 7300
    }
  ],
  "Resources": {
    "Cpus": 250,
    "Memory": 256,
    "Gpus": {}
  }
}

*/

func runBalticModule() error {

	envs := []*pb.XEnvironmentVariable{
		{
			Key:   "ModuleManagerPort",
			Value: "7301",
		},
		{
			Key:   "BatchManagerUrl",
			Value: "https://host.docker.internal:7001",
		},
	}
	bm := &pb.XBalticModuleBuild{
		BatchId:              bid,
		ModuleId:             mid,
		Image:                "busybox",
		EnvironmentVariables: envs,
		Command:              "/bin/sleep",
		CommandArguments:     []string{"300"},
		PortMappings: []*pb.XPortMapping{
			{
				ContainerPort: 1234,
				PublishedPort: 8888,
				Protocol:      pb.XPortMapping_TCP,
			},
		},
		Volumes: []*pb.XVolumeDescription{
			{
				Size: 1,
				//StorageClass: "rook-ceph-rbd",
				MountPath: "/storage",
			},
		},
		Resources: &pb.XResourceRequest{
			Cpus:   100,
			Memory: 128,
			Gpus: &pb.XGpuRequest{
				Quantity: 0,
				Type:     "nvidia-gtx-1080ti",
			},
		},
		ConfigFiles: []*pb.XConfigFileDescription{
			{
				Data:      "Lorem Ipsum\nSome more data...",
				MountPath: "/conffile1",
			},
			{
				Data:      "Lorem Ipsum2\nSome more data2...",
				MountPath: "/conffile2",
			},
		},
		Scope: pb.XBalticModuleBuild_CLUSTER,
	}
	r, err := clusterProxy.RunBalticModule(ctx, bm)
	if err != nil {
		log.Fatalf("RunBalticModule failed: %v", err)
	}
	fmt.Printf("RunBalticModule returned %+v\n", r)

	return nil
}
func checkBalticModuleStatus() error {

	r, err := clusterProxy.CheckBalticModuleStatus(ctx, &pb.Module{
		BatchId:  bid,
		ModuleId: mid,
	})
	if err != nil {
		log.Fatalf("checkBalticModuleStatus failed: %v", err)
	}
	fmt.Printf("checkBalticModuleStatus returned Status:%s Message:%s\n", r.Status, r.Message)

	return err
}
func disposeBalticModule() error {
	r, err := clusterProxy.DisposeBalticModule(ctx, &pb.Module{
		BatchId:  bid,
		ModuleId: mid,
	})
	if err != nil {
		log.Fatalf("disposeBalticModule failed: %v", err)
	}
	fmt.Printf("disposeBalticModule returned %+v\n", r)

	return err
}
func getClusterDescription() error {
	r, err := clusterProxy.GetClusterDescription(ctx, &empty.Empty{})
	if err != nil {
		log.Fatalf("getClusterDescription failed: %v", err)
	}
	json, err := json.MarshalIndent(r, "", "  ")
	if err == nil {
		fmt.Printf("getClusterDescription returned: %s\n", json)
	}
	return err
}

func usage() {
	fmt.Printf("Usage: %s <address> <command>\n", os.Args[0])
	fmt.Println("  Available commands: ")
	for cmd := range commands {
		fmt.Println("    " + cmd)
	}
	fmt.Printf("\n  Example: %s localhost:50051 GetClusterDescription\n", os.Args[0])
}

func main() {

	commands = map[string]func() error{
		"PrepareWorkspace":        prepareWorkspace,
		"CheckWorkspaceStatus":    checkWorkspaceStatus,
		"PurgeWorkspace":          purgeWorkspace,
		"RunBalticModule":         runBalticModule,
		"CheckBalticModuleStatus": checkBalticModuleStatus,
		"DisposeBalticModule":     disposeBalticModule,
		"GetClusterDescription":   getClusterDescription,
	}

	// parse args
	if len(os.Args) == 1 {
		usage()
		os.Exit(1)
	}
	addr := os.Args[1]
	cmd := os.Args[2]
	if commands[cmd] == nil {
		fmt.Printf("Invalid command \"%s\"\n", cmd)
		usage()
		os.Exit(1)
	}

	// Set up a connection to the server.
	fmt.Printf("Connecting to %s\n", addr)
	conn, err := grpc.Dial(addr, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	clusterProxy = pb.NewClusterProxyClient(conn)

	_ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	ctx = _ctx
	defer cancel()

	// Run program
	err = commands[cmd]()
	if err != nil {
		log.Printf("Error running command \"%s\": %v", cmd, err)
	}
}
