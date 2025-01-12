package keptn

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"strings"
)

// is the local test resource path for the dynatrace.conf.yaml
const localConfigFilename = "dynatrace/_dynatrace.conf.yaml"

type LocalResourceClient struct {
}

func NewLocalResourceClient() *LocalResourceClient {
	return &LocalResourceClient{}
}

func (c *LocalResourceClient) GetDynatraceConfig(project string, stage string, service string) (string, error) {
	return c.GetResource(project, stage, service, configFilename)
}

func (c *LocalResourceClient) GetResource(project string, stage string, service string, resourceURI string) (string, error) {
	// hack to retrieve the local config file
	if resourceURI == configFilename {
		resourceURI = localConfigFilename
	}

	localFileContent, err := ioutil.ReadFile(resourceURI)
	if err != nil {
		log.WithFields(
			log.Fields{
				"resourceURI": resourceURI,
				"service":     service,
				"stage":       stage,
				"project":     project,
			}).Info("File not found locally")
		return "", nil
	}

	log.WithField("resourceURI", resourceURI).Info("Loaded LOCAL file")
	return string(localFileContent), nil
}

func (c *LocalResourceClient) GetProjectResource(project string, resourceURI string) (string, error) {
	return c.GetResource(project, "", "", strings.ToLower(strings.ReplaceAll(resourceURI, "dynatrace/", "../../../dynatrace/project_")))
}

func (c *LocalResourceClient) getStageResource(project string, stage string, resourceURI string) (string, error) {
	return c.GetResource(project, stage, "", strings.ToLower(strings.ReplaceAll(resourceURI, "dynatrace/", "../../../dynatrace/stage_")))
}

func (c *LocalResourceClient) GetServiceResource(project string, stage string, service string, resourceURI string) (string, error) {
	return c.GetResource(project, stage, service, strings.ToLower(strings.ReplaceAll(resourceURI, "dynatrace/", "../../../dynatrace/service_")))
}

func (c *LocalResourceClient) UploadKeptnResource(contentToUpload []byte, remoteResourceURI string) error {
	// if we run in a runlocal mode we are just getting the file from the local disk
	err := ioutil.WriteFile(remoteResourceURI, contentToUpload, 0644)
	if err != nil {
		return fmt.Errorf("couldnt write local file %s: %v", remoteResourceURI, err)
	}

	log.WithField("remoteResourceURI", remoteResourceURI).Info("Local file written")
	return nil
}
