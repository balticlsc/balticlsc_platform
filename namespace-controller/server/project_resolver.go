package server

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	rancher "icekube/admission-controller/rancher-go/client"
	"log"
	"net/http"
)

type UserProjectInfo struct {
	IsOwner     bool   `json:"isowner"`
	ProjectID   string `json:"projectid"`
	ProjectName string `json:"projectname"`
}

type ProjectResolverCache struct {
	User2projects       map[string][]UserProjectInfo `json:"user2projects"`
	Namespace2ProjectID map[string]string            `json:"namespace2projectid"`
}

type ProjectResolver struct {
	client    rancher.Client
	clusterID string
	cache     ProjectResolverCache
}

const NOT_MEMBER_IN_ANY_PROJECTS = 0
const MEMBER_OF_MULTIPLE_PROJECTS = 1
const PROJECT_OWNER = 2

type ResolveError struct {
	ErrorType int
}

func newResolveError(errorType int) ResolveError {
	return ResolveError{ErrorType: errorType}
}

func (e ResolveError) Error() string {
	return fmt.Sprintf("Resolve error")
}

func NewProjectResolver(rancherURL string, clusterID string, token string, disableCertificateCheck bool) *ProjectResolver {
	if disableCertificateCheck {
		http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}

	rancherClient := rancher.NewClient(rancherURL, token)
	projectResolver := ProjectResolver{
		client:    rancherClient,
		clusterID: clusterID,
		cache: ProjectResolverCache{
			User2projects:       make(map[string][]UserProjectInfo),
			Namespace2ProjectID: make(map[string]string),
		},
	}

	return &projectResolver
}

func (projectResolver *ProjectResolver) ResolveProjectsByUser(userID string) ([]UserProjectInfo, error) {
	plist := projectResolver.cache.User2projects[userID]
	if len(plist) > 0 {
		return plist, nil
	}
	err := projectResolver.UpdateCache()
	if err != nil {
		return nil, err
	}
	plist = projectResolver.cache.User2projects[userID]
	if len(plist) > 0 {
		return plist, err
	}
	return nil, err
}

func (projectResolver *ProjectResolver) ResolveProjectIdbyNamespace(namespace string) (string, bool, error) {
	log.Print("ResolveProjectIdbyNamespace(" + namespace + ")")
	projectID := projectResolver.cache.Namespace2ProjectID[namespace]
	cacheHit := true
	var err error
	if projectID == "" {
		cacheHit = false
		err = projectResolver.UpdateCache()
		if err != nil {
			return "", false, err
		}
		projectID = projectResolver.cache.Namespace2ProjectID[namespace]
	}
	return projectID, cacheHit, nil
}

func (projectResolver *ProjectResolver) findUserByProject(projectID string) (string, bool, error) {
	for userId, plist := range projectResolver.cache.User2projects {
		if len(plist) > 0 {
			for _, project := range plist {
				if project.ProjectID == projectID {
					return userId, project.IsOwner, nil
				}
			}
		}
	}
	return "", false, errors.New("projectID not found")
}

func (projectResolver *ProjectResolver) FindUserByProject(projectID string) (string, bool, error) {
	userId, isOwner, err := projectResolver.findUserByProject(projectID)
	if err != nil {
		err = projectResolver.UpdateCache()
		if err != nil {
			return "", false, err
		}
		userId, isOwner, err = projectResolver.findUserByProject(projectID)
	} else {
		return userId, isOwner, nil
	}

	plist := projectResolver.cache.User2projects[userId]
	if len(plist) > 1 {
		return userId, false, nil
	}
	if len(plist) == 1 {
		return userId, plist[0].IsOwner, nil
	}

	return "", false, errors.New("projectID not found")
}

func (projectResolver *ProjectResolver) UpdateCache() error {
	log.Print("UpdateCache called")
	entities, err := projectResolver.client.GetProjects(projectResolver.clusterID)
	if err != nil {
		log.Print("GetProjects returned error %v", err)
		return err
	}
	e, _ := json.Marshal(entities)
	fmt.Println(string(e))
	projectResolver.ClearCache()
	for _, project := range entities {
		members, err := projectResolver.client.GetProjectMembers(project.ID)
		if err != nil {
			return err
		}
		PrettyPrint("project "+string(project.ID)+" members: ", members)
		for _, member := range members {
			plist := projectResolver.cache.User2projects[member.UserID]
			plist = append(plist, UserProjectInfo{
				IsOwner:     member.RoleTemplateID == "project-owner",
				ProjectID:   project.ID,
				ProjectName: project.Name,
			})
			projectResolver.cache.User2projects[member.UserID] = plist
		}

		// Get namespaces in project
		namespaces, err := projectResolver.client.GetProjectNamespaces(projectResolver.clusterID, project.ID)
		if err != nil {
			log.Print("Failed getting namespaces for project " + project.ID)
			return err
		}
		for _, namespace := range namespaces {
			projectResolver.cache.Namespace2ProjectID[namespace] = project.ID
		}
	}
	PrettyPrint("cache: ", projectResolver.cache)
	// Send cache to OPA

	json, _ := json.Marshal(projectResolver.cache)
	req, err := http.NewRequest(http.MethodPut, "https://opa.opa/v1/data/rancher", bytes.NewBuffer(json))
	if err != nil {
		log.Print("Error creating http Put request: " + err.Error())
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	// initialize http client
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Print("Error sending Put request to OPA")
		panic(err)
	}

	log.Print("Response code from OPA: ", resp.StatusCode, " ", http.StatusText(resp.StatusCode))
	return nil
}

func (projectResolver *ProjectResolver) ResolveProjectIDFromRancher(userID string) (string, error) {
	var projectIDs []string
	if len(projectIDs) == 0 {
		return "", newResolveError(NOT_MEMBER_IN_ANY_PROJECTS)
	}

	if len(projectIDs) > 1 {
		return "", newResolveError(MEMBER_OF_MULTIPLE_PROJECTS)
	}
	/*
		if projectOwner {
			return "", newResolveError(PROJECT_OWNER)
		}
	*/
	return projectIDs[0], nil
}

func (projectResolver *ProjectResolver) ClearCache() {
	for k := range projectResolver.cache.User2projects {
		delete(projectResolver.cache.User2projects, k)
	}
	for k := range projectResolver.cache.Namespace2ProjectID {
		delete(projectResolver.cache.Namespace2ProjectID, k)
	}

}
