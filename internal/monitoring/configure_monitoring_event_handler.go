package monitoring

import (
	"errors"
	"fmt"
	"github.com/keptn-contrib/dynatrace-service/internal/dynatrace"

	keptncommon "github.com/keptn/go-utils/pkg/lib/keptn"
	keptnv2 "github.com/keptn/go-utils/pkg/lib/v0_2_0"
	log "github.com/sirupsen/logrus"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/keptn-contrib/dynatrace-service/internal/credentials"
)

type KeptnAPIConnectionCheck struct {
	APIURL               string
	ConnectionSuccessful bool
	Message              string
}

type ConfigureMonitoringEventHandler struct {
	event         *ConfigureMonitoringAdapter
	client        *dynatrace.Client
	incomingEvent cloudevents.Event
}

// NewConfigureMonitoringEventHandler returns a new ConfigureMonitoringEventHandler
func NewConfigureMonitoringEventHandler(event *ConfigureMonitoringAdapter, client *dynatrace.Client, incomingEvent cloudevents.Event) ConfigureMonitoringEventHandler {
	return ConfigureMonitoringEventHandler{
		event:         event,
		client:        client,
		incomingEvent: incomingEvent,
	}
}

func (eh ConfigureMonitoringEventHandler) HandleEvent() error {
	err := eh.configureMonitoring()
	if err != nil {
		log.WithError(err).Error("Configure monitoring failed")
	}
	return nil
}

func (eh *ConfigureMonitoringEventHandler) configureMonitoring() error {
	log.Info("Configuring Dynatrace monitoring")
	if eh.event.IsNotForDynatrace() {
		return nil
	}

	keptnAPICheck := &KeptnAPIConnectionCheck{}
	// check the connection to the Keptn API
	keptnCredentials, err := credentials.GetKeptnCredentials()
	if err != nil {
		log.WithError(err).Error("Failed to get Keptn API credentials")
		keptnAPICheck.Message = "Failed to get Keptn API Credentials"
		keptnAPICheck.ConnectionSuccessful = false
		keptnAPICheck.APIURL = "unknown"
	} else {
		keptnAPICheck.APIURL = keptnCredentials.APIURL
		log.WithField("apiUrl", keptnCredentials.APIURL).Print("Verifying access to Keptn API")

		err = credentials.CheckKeptnConnection(keptnCredentials)
		if err != nil {
			keptnAPICheck.ConnectionSuccessful = false
			keptnAPICheck.Message = "Warning: Keptn API connection cannot be verified. This might be due to a no-loopback policy of your LoadBalancer. The endpoint might still be reachable from outside the cluster."
			log.WithError(err).Warn(keptnAPICheck.Message)
		} else {
			keptnAPICheck.ConnectionSuccessful = true
		}
	}

	keptnHandler, err := keptnv2.NewKeptn(&eh.incomingEvent, keptncommon.KeptnOpts{})
	if err != nil {
		return fmt.Errorf("could not create Keptn handler: %v", err)
	}

	var shipyard *keptnv2.Shipyard
	if eh.event.GetProject() != "" {
		shipyard, err = keptnHandler.GetShipyard()
		if err != nil {
			msg := fmt.Sprintf("failed to retrieve shipyard for project %s: %v", eh.event.GetProject(), err)
			return eh.handleError(keptnHandler, eh.event, msg)
		}
	}

	cfg := NewConfiguration(eh.client, keptnHandler)

	configuredEntities, err := cfg.ConfigureMonitoring(eh.event.GetProject(), shipyard)
	if err != nil {
		return eh.handleError(keptnHandler, eh.event, err.Error())
	}

	log.Info("Dynatrace Monitoring setup done")

	if err := eh.sendConfigureMonitoringFinishedEvent(keptnHandler, eh.event, keptnv2.StatusSucceeded, keptnv2.ResultPass, getConfigureMonitoringResultMessage(keptnAPICheck, configuredEntities)); err != nil {
		log.WithError(err).Error("Failed to send configure monitoring finished event")
	}
	return nil
}

func getConfigureMonitoringResultMessage(apiCheck *KeptnAPIConnectionCheck, entities *dynatrace.ConfiguredEntities) string {
	if entities == nil {
		return ""
	}
	msg := "Dynatrace monitoring setup done.\nThe following entities have been configured:\n\n"

	if entities.ManagementZonesEnabled && len(entities.ManagementZones) > 0 {
		msg = msg + "---Management Zones:--- \n"
		for _, mz := range entities.ManagementZones {
			if mz.Success {
				msg = msg + "  - " + mz.Name + ": Created successfully \n"
			} else {
				msg = msg + "  - " + mz.Name + ": Error: " + mz.Message + "\n"
			}
		}
		msg = msg + "\n\n"
	}

	if entities.TaggingRulesEnabled && len(entities.TaggingRules) > 0 {
		msg = msg + "---Automatic Tagging Rules:--- \n"
		for _, mz := range entities.TaggingRules {
			if mz.Success {
				msg = msg + "  - " + mz.Name + ": Created successfully \n"
			} else {
				msg = msg + "  - " + mz.Name + ": Error: " + mz.Message + "\n"
			}
		}
		msg = msg + "\n\n"
	}

	if entities.ProblemNotificationsEnabled {
		msg = msg + "---Problem Notification:--- \n"
		msg = msg + "  - " + entities.ProblemNotifications.Message
		msg = msg + "\n\n"
	}

	if entities.MetricEventsEnabled && len(entities.MetricEvents) > 0 {
		msg = msg + "---Metric Events:--- \n"
		for _, mz := range entities.MetricEvents {
			if mz.Success {
				msg = msg + "  - " + mz.Name + ": Created successfully \n"
			} else {
				msg = msg + "  - " + mz.Name + ": Error: " + mz.Message + "\n"
			}
		}
		msg = msg + "\n\n"
	}

	if entities.DashboardEnabled && entities.Dashboard.Message != "" {
		msg = msg + "---Dashboard:--- \n"
		msg = msg + "  - " + entities.Dashboard.Message
		msg = msg + "\n\n"
	}

	if apiCheck != nil {
		msg = msg + "---Keptn API Connection Check:--- \n"
		msg = msg + "  - Keptn API URL: " + apiCheck.APIURL + "\n"
		msg = msg + fmt.Sprintf("  - Connection Successful: %v. %s\n", apiCheck.ConnectionSuccessful, apiCheck.Message)
		msg = msg + "\n"
	}

	return msg
}

func (eh *ConfigureMonitoringEventHandler) handleError(client *keptnv2.Keptn, adapter *ConfigureMonitoringAdapter, msg string) error {
	log.Error(msg)
	if err := eh.sendConfigureMonitoringFinishedEvent(client, adapter, keptnv2.StatusErrored, keptnv2.ResultFailed, msg); err != nil {
		log.WithError(err).Error("Failed to send configure monitoring finished event")
	}
	return errors.New(msg)
}

func (eh *ConfigureMonitoringEventHandler) sendConfigureMonitoringFinishedEvent(client *keptnv2.Keptn, adapter *ConfigureMonitoringAdapter, status keptnv2.StatusType, result keptnv2.ResultType, message string) error {

	cmFinishedEvent := &keptnv2.ConfigureMonitoringFinishedEventData{
		EventData: keptnv2.EventData{
			Project: adapter.GetProject(),
			Service: adapter.GetService(),
			Status:  status,
			Result:  result,
			Message: message,
		},
	}

	ev := cloudevents.NewEvent()
	ev.SetSource("dynatrace-service")
	ev.SetDataContentType(cloudevents.ApplicationJSON)
	ev.SetType(keptnv2.GetFinishedEventType(keptnv2.ConfigureMonitoringTaskName))
	ev.SetData(cloudevents.ApplicationJSON, cmFinishedEvent)
	ev.SetExtension("shkeptncontext", adapter.GetShKeptnContext())
	ev.SetExtension("triggeredid", eh.incomingEvent.Context.GetID())

	if err := client.SendCloudEvent(ev); err != nil {
		return fmt.Errorf("could not send %s event: %s", keptnv2.GetFinishedEventType(keptnv2.ConfigureMonitoringTaskName), err.Error())
	}

	return nil
}