package server

import (
	"log"
	"net/http"
	"strconv"
)

const (
	tlsDir      = `/run/secrets/tls`
	tlsCertFile = `tls.crt`
	tlsKeyFile  = `tls.key`
)

func ListenForever(port int, certPath string, keyPath string, rancherURL string, token string, clusterID string, disableCertificateCheck bool) {
	projectResolver := NewProjectResolver(rancherURL, clusterID, token, disableCertificateCheck)
	admissionController := NewAdmissionController(projectResolver)

	mux := http.NewServeMux()
	mux.Handle("/mutate", admissionController.AdmitFuncHandler())
	server := &http.Server{
		Addr:    ":" + strconv.Itoa(port),
		Handler: mux,
	}

	log.Fatal(server.ListenAndServeTLS(certPath, keyPath))
}
