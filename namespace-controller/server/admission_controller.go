package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"

	v1 "k8s.io/api/admission/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
)

const (
	jsonContentType = `application/json`
)

var (
	UniversalDeserializer = serializer.NewCodecFactory(runtime.NewScheme()).UniversalDeserializer()
)

type AdmissionController struct {
	projectResolver *ProjectResolver
}

type PatchOperation struct {
	Op    string      `json:"op"`
	Path  string      `json:"path"`
	Value interface{} `json:"value,omitempty"`
}

func IsKubeNamespace(ns string) bool {
	return ns == metav1.NamespacePublic || ns == metav1.NamespaceSystem
}

func PrettyPrint(prefix string, v interface{}) (err error) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err == nil {
		fmt.Println(prefix + string(b))
	}
	return
}

func NewAdmissionController(projectResolver *ProjectResolver) *AdmissionController {
	return &AdmissionController{projectResolver: projectResolver}
}

func (admissionController *AdmissionController) DoServeAdmitFunc(w http.ResponseWriter, r *http.Request) ([]byte, error) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return nil, fmt.Errorf("Invalid method %s, only POST requests are allowed", r.Method)
	}

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return nil, fmt.Errorf("Could not read request body: %v", err)
	}

	if contentType := r.Header.Get("Content-Type"); contentType != jsonContentType {
		w.WriteHeader(http.StatusBadRequest)
		return nil, fmt.Errorf("Unsupported content type %s, only %s is supported", contentType, jsonContentType)
	}

	var admissionReviewReq v1.AdmissionReview

	if _, _, err := UniversalDeserializer.Decode(body, nil, &admissionReviewReq); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return nil, fmt.Errorf("Could not deserialize request: %v", err)
	} else if admissionReviewReq.Request == nil {
		w.WriteHeader(http.StatusBadRequest)
		return nil, errors.New("Malformed admission review: request is nil")
	}
	//PrettyPrint("admissionReviewReq: ", admissionReviewReq)
	admissionReviewResponse := v1.AdmissionReview{
		Response: &v1.AdmissionResponse{
			UID: admissionReviewReq.Request.UID,
		},
	}

	var patchOps []PatchOperation
	allowed := true // by default
	deniedReason := "denied"

	switch admissionReviewReq.Request.Kind.Kind {
	case "Namespace":
		// Put namespaces into right project
		userID := admissionReviewReq.Request.UserInfo.Username
		plist, err := admissionController.projectResolver.ResolveProjectsByUser(userID)
		if err != nil {
			log.Print("AdmissionController: Failed to resolve project id, ignoring request")
			return nil, err
		}
		if len(plist) > 1 {
			log.Print("AdmissionController: user " + userID + " is member of several projects, ignoring request")
			return nil, err
		}
		if len(plist) == 0 {
			log.Print("AdmissionController: Failed to resolve project id, ignoring request")
			return nil, err
		}
		projectID := plist[0].ProjectID
		projectName := plist[0].ProjectName
		if plist[0].IsOwner {
			log.Print("AdmissionController: user " + userID + " is owner of project " + projectID + ", ignoring request")
			return nil, err
		}

		log.Print("userid: " + userID + " -> projectId: " + projectID)

		var ns *corev1.Namespace
		err = json.Unmarshal(admissionReviewReq.Request.Object.Raw, &ns)
		if err != nil {
			return nil, err
		}
		log.Print("AdmissionController: Adding annotation: " + userID + "->" + projectID + " to namespace " + ns.GetName())
		//PrettyPrint("ns: ", ns)

		// Check that namespace starts with projectName
		if !strings.HasPrefix(ns.GetName(), projectName) {
			deniedReason = fmt.Sprintf("Namespace must start with '%s'", projectName)
			log.Print(deniedReason)
			allowed = false
			break
		}

		kvp := make(map[string]string)
		for k, v := range ns.ObjectMeta.GetAnnotations() {
			kvp[k] = v
		}
		// This also solves problem if someone is trying to create namespace in other users
		// project
		kvp["field.cattle.io/projectId"] = projectID
		patchOps = append(patchOps, PatchOperation{
			Op:    "add",
			Path:  "/metadata/annotations",
			Value: kvp,
		})
	case "Pod":
		log.Print("AdmissionController: Handling Pod request")
		var pod *corev1.Pod
		err = json.Unmarshal(admissionReviewReq.Request.Object.Raw, &pod)
		if err != nil {
			return nil, err
		}
		namespace := admissionReviewReq.Request.Namespace
		projectID, _, _ := admissionController.projectResolver.ResolveProjectIdbyNamespace(namespace)
		if projectID == "" {
			log.Print("AdmissionController: Ignoring CREATE Pod request")
			break
		}
		log.Print("namespace " + namespace + " in project " + projectID)
		_, isOwner, err := admissionController.projectResolver.FindUserByProject(projectID)
		if err != nil || isOwner {
			log.Print("AdmissionController: Ignoring CREATE Pod request because " + projectID + " is not normal user project")
			break
		}
		for i := range pod.Spec.Containers {
			//var r *corev1.ResourceRequirements
			r := pod.Spec.Containers[i].Resources
			PrettyPrint("requests: ", r.Requests)
			PrettyPrint("limits: ", r.Limits)
			for rtype, rval := range r.Requests {
				lval, exists := r.Limits[rtype]
				if exists {
					// Cmp function does handle usage of suffixes like G, Gi, m etc
					if lval.Cmp(rval) == 1 {
						deniedReason = fmt.Sprintf("Resource limit.%s: %s is higher than the request.%s: %s",
							rtype, lval.String(), rtype, rval.String())
						log.Print(deniedReason)
						allowed = false
					}
					/* uncomment if we want to allow the request and set the limit to same as request:
					allowed = true
					fmt.Println("adding resource limit.%s: %s", rtype, rval.String)
					patchOps = append(patchOps, PatchOperation{
						Op:    "add",
						Path:  fmt.Sprintf("/spec/containers/%d/resources/limits/%s", i, rtype),
						Value: rval.String(),
					})
					*/
				} else {
					// Adding limit == request
					fmt.Println("adding resource limit.%s: %s", rtype, rval.String)
					patchOps = append(patchOps, PatchOperation{
						Op:    "add",
						Path:  fmt.Sprintf("/spec/containers/%d/resources/limits/%s", i, rtype),
						Value: rval.String(),
					})
				}
			}
			if allowed && r.Requests == nil {
				deniedReason = fmt.Sprintf("Resource requests are missing for container %s",
					pod.Spec.Containers[i].Name)
				log.Print(deniedReason)
				allowed = false
			}
		}
	default:
		log.Print("AdmissionController: Ignoring request of kind: " + admissionReviewReq.Request.Kind.Kind)
	}

	patchBytes, err := json.Marshal(patchOps)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return nil, fmt.Errorf("Could not marshal JSON patch: %v", err)
	}

	admissionReviewResponse.Response.Allowed = allowed
	if allowed {
		admissionReviewResponse.Response.Patch = patchBytes
	} else {
		admissionReviewResponse.Response.Result = &metav1.Status{
			Message: deniedReason,
		}
	}

	bytes, err := json.Marshal(&admissionReviewResponse)
	if err != nil {
		return nil, fmt.Errorf("Marshaling response: %v", err)
	}

	return bytes, nil
}

func (admissionController *AdmissionController) ServeAdmitFunc(w http.ResponseWriter, r *http.Request) {
	var writeErr error
	if bytes, err := admissionController.DoServeAdmitFunc(w, r); err != nil {
		log.Printf("AdmissionController: Error registering admission controller: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		_, writeErr = w.Write([]byte(err.Error()))
	} else {
		_, writeErr = w.Write(bytes)
	}

	if writeErr != nil {
		log.Printf("AdmissionController: Could not write response: %v", writeErr)
	}
}

func (admissionController *AdmissionController) AdmitFuncHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		admissionController.ServeAdmitFunc(w, r)
	})
}
