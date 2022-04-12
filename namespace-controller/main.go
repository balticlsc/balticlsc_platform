package main

import (
	server "icekube/admission-controller/server"
	"log"
	"os"
	"path/filepath"
)

const (
	tlsDir      = `/run/secrets/tls`
	tlsCertFile = `tls.crt`
	tlsKeyFile  = `tls.key`
)

func main() {
	certPath := filepath.Join(tlsDir, tlsCertFile)
	keyPath := filepath.Join(tlsDir, tlsKeyFile)

	rancherURL := os.Getenv("RANCHER_URL")
	token := os.Getenv("RANCHER_TOKEN")
	clusterID := os.Getenv("RANCHER_CLUSTER_ID")

	log.Print("AdmissionController: Starting web server")
	log.Print("AdmissionController:   rancherURL=" + rancherURL)
	log.Print("AdmissionController:   token=" + token)
	log.Print("AdmissionController:   clusterID=" + clusterID)

	server.ListenForever(8443, certPath, keyPath, rancherURL, token, clusterID, true)
}
