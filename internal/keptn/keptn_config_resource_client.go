package keptn

import (
	"errors"
	"fmt"
	"github.com/keptn-contrib/dynatrace-service/internal/common"
	keptnmodels "github.com/keptn/go-utils/pkg/api/models"
	api "github.com/keptn/go-utils/pkg/api/utils"
	log "github.com/sirupsen/logrus"
)

// ConfigResourceClientInterface defines the methods for interacting with resources of Keptn's configuration service
type ConfigResourceClientInterface interface {
	GetResource(project string, stage string, service string, resourceURI string) (string, error)
	GetProjectResource(project string, resourceURI string) (string, error)
	GetStageResource(project string, stage string, resourceURI string) (string, error)
	GetServiceResource(project string, stage string, service string, resourceURI string) (string, error)
	UploadResource(contentToUpload []byte, remoteResourceURI string, project string, stage string, service string) error
}

// ResourceError represents an error for a resource that was not found
type ResourceError struct {
	uri     string
	project string
	stage   string
	service string
}

// ResourceNotFoundError represents an error for a resource that was not found
type ResourceNotFoundError ResourceError

// Error returns a string representation of this error
func (e *ResourceNotFoundError) Error() string {
	return fmt.Sprintf("could not find resource: '%s' %s", e.uri, getLocation(e.service, e.stage, e.project))
}

// ResourceEmptyError represents an error for a resource that was found, but is empty
type ResourceEmptyError ResourceError

// Error returns a string representation of this error
func (e *ResourceEmptyError) Error() string {
	return fmt.Sprintf("found resource: '%s' %s, but it is empty", e.uri, getLocation(e.service, e.stage, e.project))
}

// ResourceUploadFailedError represents an error for a resource that could not be uploaded
type ResourceUploadFailedError struct {
	ResourceError
	message string
}

// Error returns a string representation of this error
func (e *ResourceUploadFailedError) Error() string {
	return fmt.Sprintf("could not upload resource: '%s' %s: %s", e.uri, getLocation(e.service, e.stage, e.project), e.message)
}

// ResourceRetrievalFailedError represents an error for a resource that could not be retrieved because of an error
type ResourceRetrievalFailedError struct {
	ResourceError
	message string
}

// Error returns a string representation of this error
func (e *ResourceRetrievalFailedError) Error() string {
	return fmt.Sprintf("could not retrieve resource: '%s' %s: %s", e.uri, getLocation(e.service, e.stage, e.project), e.message)
}

func getLocation(service string, stage string, project string) string {
	var location string

	if service != "" {
		location += fmt.Sprintf(" for service '%s'", service)
	}
	if stage != "" {
		location += fmt.Sprintf(" at stage '%s'", stage)
	}
	if project != "" {
		location += fmt.Sprintf(" of project '%s'", project)
	}

	return location
}

// ConfigResourceClient is the default implementation for the ConfigResourceClientInterface using a Keptn api.ResourceHandler
type ConfigResourceClient struct {
	handler *api.ResourceHandler
}

// NewDefaultConfigResourceClient creates a new ResourceClient with a default Keptn resource handler for the configuration service
func NewDefaultConfigResourceClient() *ConfigResourceClient {
	return NewConfigResourceClient(
		api.NewResourceHandler(common.GetConfigurationServiceURL()))
}

// NewConfigResourceClient creates a new ResourceClient with a Keptn resource handler for the configuration service
func NewConfigResourceClient(handler *api.ResourceHandler) *ConfigResourceClient {
	return &ConfigResourceClient{
		handler: handler,
	}
}

// GetResource tries to find the first instance of a given resource on service, stage or project level
func (rc *ConfigResourceClient) GetResource(project string, stage string, service string, resourceURI string) (string, error) {
	var rnfErrorType *ResourceNotFoundError
	if project != "" && stage != "" && service != "" {
		keptnResourceContent, err := rc.GetServiceResource(project, stage, service, resourceURI)
		if errors.As(err, &rnfErrorType) {
			log.WithFields(
				log.Fields{
					"project": project,
					"stage":   stage,
					"service": service,
				}).Debugf("%s not available for service", resourceURI)
		} else if err != nil {
			return "", err
		} else {
			log.WithFields(
				log.Fields{
					"project": project,
					"stage":   stage,
					"service": service,
				}).Infof("Found %s for service", resourceURI)
			return keptnResourceContent, nil
		}
	}

	if project != "" && stage != "" {
		keptnResourceContent, err := rc.GetStageResource(project, stage, resourceURI)
		if errors.As(err, &rnfErrorType) {
			log.WithFields(
				log.Fields{
					"project": project,
					"stage":   stage,
				}).Debugf("%s not available for stage", resourceURI)
		} else if err != nil {
			return "", err
		} else {
			log.WithFields(
				log.Fields{
					"project": project,
					"stage":   stage,
				}).Infof("Found %s for stage", resourceURI)
			return keptnResourceContent, nil
		}
	}

	if project != "" {
		keptnResourceContent, err := rc.GetProjectResource(project, resourceURI)
		if err == api.ResourceNotFoundError {
			log.WithField("project", project).Debugf("%s not available for project", resourceURI)
		} else if err != nil {
			return "", err
		} else {
			log.WithField("project", project).Infof("Found %s for project", resourceURI)
			return keptnResourceContent, nil
		}
	}

	log.Infof("%s not found", resourceURI)
	return "", &ResourceNotFoundError{uri: resourceURI, project: project, stage: stage, service: service}
}

// GetServiceResource tries to retrieve a resourceURI on service level
func (rc *ConfigResourceClient) GetServiceResource(project string, stage string, service string, resourceURI string) (string, error) {
	return getResourceByFunc(
		func() (*keptnmodels.Resource, error) {
			return rc.handler.GetServiceResource(project, stage, service, resourceURI)
		},
		func() *ResourceNotFoundError {
			return &ResourceNotFoundError{uri: resourceURI, project: project, stage: stage, service: service}
		},
		func(msg string) *ResourceRetrievalFailedError {
			return &ResourceRetrievalFailedError{ResourceError{uri: resourceURI, project: project, stage: stage, service: service}, msg}
		},
		func() *ResourceEmptyError {
			return &ResourceEmptyError{uri: resourceURI, project: project, stage: stage, service: service}
		})
}

// GetStageResource tries to retrieve a resourceURI on stage level
func (rc *ConfigResourceClient) GetStageResource(project string, stage string, resourceURI string) (string, error) {
	return getResourceByFunc(
		func() (*keptnmodels.Resource, error) { return rc.handler.GetStageResource(project, stage, resourceURI) },
		func() *ResourceNotFoundError {
			return &ResourceNotFoundError{uri: resourceURI, project: project, stage: stage}
		},
		func(msg string) *ResourceRetrievalFailedError {
			return &ResourceRetrievalFailedError{ResourceError{uri: resourceURI, project: project, stage: stage}, msg}
		},
		func() *ResourceEmptyError {
			return &ResourceEmptyError{uri: resourceURI, project: project, stage: stage}
		})
}

// GetProjectResource tries to retrieve a resourceURI on project level
func (rc *ConfigResourceClient) GetProjectResource(project string, resourceURI string) (string, error) {
	return getResourceByFunc(
		func() (*keptnmodels.Resource, error) { return rc.handler.GetProjectResource(project, resourceURI) },
		func() *ResourceNotFoundError { return &ResourceNotFoundError{uri: resourceURI, project: project} },
		func(msg string) *ResourceRetrievalFailedError {
			return &ResourceRetrievalFailedError{ResourceError{uri: resourceURI, project: project}, msg}
		},
		func() *ResourceEmptyError { return &ResourceEmptyError{uri: resourceURI, project: project} })
}

func getResourceByFunc(
	resFunc func() (*keptnmodels.Resource, error),
	rnfErrFunc func() *ResourceNotFoundError,
	rrfErrFunc func(msg string) *ResourceRetrievalFailedError,
	reErrFunc func() *ResourceEmptyError) (string, error) {
	resource, err := resFunc()
	if err != nil {
		if err == api.ResourceNotFoundError {
			return "", rnfErrFunc()
		}

		return "", rrfErrFunc(err.Error())
	}
	if resource.ResourceContent == "" {
		return "", reErrFunc()
	}

	return resource.ResourceContent, nil
}

// UploadResource tries to upload a resourceURI on service level
func (rc *ConfigResourceClient) UploadResource(contentToUpload []byte, remoteResourceURI string, project string, stage string, service string) error {
	resources := []*keptnmodels.Resource{{ResourceContent: string(contentToUpload), ResourceURI: &remoteResourceURI}}
	_, err := rc.handler.CreateResources(project, stage, service, resources)
	if err != nil {
		return &ResourceUploadFailedError{
			ResourceError{
				uri:     remoteResourceURI,
				project: project,
				stage:   stage,
				service: service,
			},
			err.GetMessage(),
		}
	}

	log.WithField("remoteResourceURI", remoteResourceURI).Info("Uploaded file")
	return nil
}
