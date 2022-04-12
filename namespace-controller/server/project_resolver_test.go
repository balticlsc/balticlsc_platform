package server

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestResolveProjectIDFromRancher(t *testing.T) {
	rancherClient := SetupRancher()

	projectID, err := CreateProject(rancherClient, "test_project")
	assert.Nil(t, err)

	userID, err := CreateUser(rancherClient, "test_username", "test_password")
	assert.Nil(t, err)

	err = AddUserAsMember(rancherClient, projectID, userID)
	assert.Nil(t, err)

	projectResolver := NewProjectResolver(rancherURL, clusterID, token, true)
	resolvedProjectID, err := projectResolver.ResolveProjectIDFromRancher(userID)
	assert.Nil(t, err)
	assert.Equal(t, projectID, resolvedProjectID)

	time.Sleep(delay)

	RemoveUser(rancherClient, userID)
	RemoveProject(rancherClient, projectID)
}

func TestResolveProjectID(t *testing.T) {
	rancherClient := SetupRancher()

	projectID, err := CreateProject(rancherClient, "test_project")
	assert.Nil(t, err)

	userID, err := CreateUser(rancherClient, "test_username", "test_password")
	assert.Nil(t, err)

	err = AddUserAsMember(rancherClient, projectID, userID)
	assert.Nil(t, err)

	projectResolver := NewProjectResolver(rancherURL, clusterID, token, true)
	resolvedProjectID, _, err := projectResolver.ResolveProjectID(userID)
	assert.Nil(t, err)
	assert.Equal(t, projectID, resolvedProjectID)

	time.Sleep(delay)

	RemoveUser(rancherClient, userID)
	RemoveProject(rancherClient, projectID)
}

func TestResolveProjectIDFromRancherNoMember(t *testing.T) {
	rancherClient := SetupRancher()

	userID, err := CreateUser(rancherClient, "test_username", "test_password")
	assert.Nil(t, err)

	// An error will be return since the there is not member of any project
	projectResolver := NewProjectResolver(rancherURL, clusterID, token, true)
	_, err = projectResolver.ResolveProjectIDFromRancher(userID)
	assert.Error(t, err)
	switch err.(type) {
	case ResolveError:
		assert.True(t, err.(ResolveError).ErrorType == NOT_MEMBER_IN_ANY_PROJECTS)
	default:
		assert.Fail(t, err.Error())
	}

	time.Sleep(delay)

	RemoveUser(rancherClient, userID)
}

func TestResolveProjectIDNoMember(t *testing.T) {
	rancherClient := SetupRancher()

	userID, err := CreateUser(rancherClient, "test_username", "test_password")
	assert.Nil(t, err)

	// An error will be return since the there is not member of any project
	projectResolver := NewProjectResolver(rancherURL, clusterID, token, true)
	_, _, err = projectResolver.ResolveProjectID(userID)
	assert.Error(t, err)
	switch err.(type) {
	case ResolveError:
		assert.True(t, err.(ResolveError).ErrorType == NOT_MEMBER_IN_ANY_PROJECTS)
	default:
		assert.Fail(t, err.Error())
	}

	time.Sleep(delay)

	RemoveUser(rancherClient, userID)
}

func TestResolveProjectIDFromRancherMultipleProject(t *testing.T) {
	rancherClient := SetupRancher()

	projectID1, err := CreateProject(rancherClient, "test_project_1")
	assert.Nil(t, err)

	projectID2, err := CreateProject(rancherClient, "test_project_2")
	assert.Nil(t, err)

	userID, err := CreateUser(rancherClient, "test_username", "test_password")
	assert.Nil(t, err)

	err = AddUserAsMember(rancherClient, projectID1, userID)
	assert.Nil(t, err)

	err = AddUserAsMember(rancherClient, projectID2, userID)
	assert.Nil(t, err)

	// An error will be return since the user is member of multiple project
	projectResolver := NewProjectResolver(rancherURL, clusterID, token, true)
	_, err = projectResolver.ResolveProjectIDFromRancher(userID)
	assert.Error(t, err)
	switch err.(type) {
	case ResolveError:
		assert.True(t, err.(ResolveError).ErrorType == MEMBER_OF_MULTIPLE_PROJECTS)
	default:
		assert.Fail(t, err.Error())
	}

	time.Sleep(delay)

	RemoveUser(rancherClient, userID)
	RemoveProject(rancherClient, projectID1)
	RemoveProject(rancherClient, projectID2)
}

func TestResolveProjectIDMultipleProject(t *testing.T) {
	rancherClient := SetupRancher()

	projectID1, err := CreateProject(rancherClient, "test_project_1")
	assert.Nil(t, err)

	projectID2, err := CreateProject(rancherClient, "test_project_2")
	assert.Nil(t, err)

	userID, err := CreateUser(rancherClient, "test_username", "test_password")
	assert.Nil(t, err)

	err = AddUserAsMember(rancherClient, projectID1, userID)
	assert.Nil(t, err)

	err = AddUserAsMember(rancherClient, projectID2, userID)
	assert.Nil(t, err)

	// An error will be return since the user is member of multiple project
	projectResolver := NewProjectResolver(rancherURL, clusterID, token, true)
	_, _, err = projectResolver.ResolveProjectID(userID)
	assert.Error(t, err)
	switch err.(type) {
	case ResolveError:
		assert.True(t, err.(ResolveError).ErrorType == MEMBER_OF_MULTIPLE_PROJECTS)
	default:
		assert.Fail(t, err.Error())
	}

	time.Sleep(delay)

	RemoveUser(rancherClient, userID)
	RemoveProject(rancherClient, projectID1)
	RemoveProject(rancherClient, projectID2)
}

func TestResolveProjectIDFromRancherProjectOwner(t *testing.T) {
	rancherClient := SetupRancher()

	projectID, err := CreateProject(rancherClient, "test_project")
	assert.Nil(t, err)

	userID, err := CreateUser(rancherClient, "test_username", "test_password")
	assert.Nil(t, err)

	err = AddUserAsOwner(rancherClient, projectID, userID)
	assert.Nil(t, err)

	// An error will be return since the user is a project owner
	projectResolver := NewProjectResolver(rancherURL, clusterID, token, true)
	_, err = projectResolver.ResolveProjectIDFromRancher(userID)
	assert.Error(t, err)
	switch err.(type) {
	case ResolveError:
		assert.True(t, err.(ResolveError).ErrorType == PROJECT_OWNER)
	default:
		assert.Fail(t, err.Error())
	}

	time.Sleep(delay)

	RemoveUser(rancherClient, userID)
	RemoveProject(rancherClient, projectID)
}

func TestResolveProjectIDProjectOwner(t *testing.T) {
	rancherClient := SetupRancher()

	projectID, err := CreateProject(rancherClient, "test_project")
	assert.Nil(t, err)

	userID, err := CreateUser(rancherClient, "test_username", "test_password")
	assert.Nil(t, err)

	err = AddUserAsOwner(rancherClient, projectID, userID)
	assert.Nil(t, err)

	// An error will be return since the user is a project owner
	projectResolver := NewProjectResolver(rancherURL, clusterID, token, true)
	_, _, err = projectResolver.ResolveProjectID(userID)
	assert.Error(t, err)
	switch err.(type) {
	case ResolveError:
		assert.True(t, err.(ResolveError).ErrorType == PROJECT_OWNER)
	default:
		assert.Fail(t, err.Error())
	}

	time.Sleep(delay)

	RemoveUser(rancherClient, userID)
	RemoveProject(rancherClient, projectID)
}

func TestCacheMechanism(t *testing.T) {
	rancherClient := SetupRancher()

	projectID, err := CreateProject(rancherClient, "test_project")
	assert.Nil(t, err)

	userID, err := CreateUser(rancherClient, "test_username", "test_password")
	assert.Nil(t, err)

	err = AddUserAsMember(rancherClient, projectID, userID)
	assert.Nil(t, err)

	projectResolver := NewProjectResolver(rancherURL, clusterID, token, true)
	resolvedProjectID, cacheHit, err := projectResolver.ResolveProjectID(userID)
	assert.Nil(t, err)
	assert.Equal(t, projectID, resolvedProjectID)
	assert.False(t, cacheHit)

	resolvedProjectID, cacheHit, err = projectResolver.ResolveProjectID(userID)
	assert.Nil(t, err)
	assert.Equal(t, projectID, resolvedProjectID)
	assert.True(t, cacheHit)

	time.Sleep(delay)

	RemoveUser(rancherClient, userID)
	RemoveProject(rancherClient, projectID)
}
