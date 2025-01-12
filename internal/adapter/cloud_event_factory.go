package adapter

import (
	"fmt"
	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/keptn-contrib/dynatrace-service/internal/event"
)

type CloudEventFactoryInterface interface {
	CreateCloudEvent() (*cloudevents.Event, error)
}

type CloudEventFactory struct {
	event     TriggeredCloudEventContentAdapter
	eventType string
	payload   interface{}
}

func NewCloudEventFactory(event TriggeredCloudEventContentAdapter, eventType string, payload interface{}) *CloudEventFactory {
	return &CloudEventFactory{
		event:     event,
		eventType: eventType,
		payload:   payload,
	}
}

func (f *CloudEventFactory) CreateCloudEvent() (*cloudevents.Event, error) {
	ce, err := NewCloudEventFactoryBase(f.event, f.eventType, f.payload).CreateCloudEvent()
	if err != nil {
		return nil, err
	}

	if f.event.GetEventID() != "" {
		ce.SetExtension("triggeredid", f.event.GetEventID())
	}

	return ce, nil
}

type CloudEventFactoryBase struct {
	event     CloudEventContentAdapter
	eventType string
	payload   interface{}
}

func NewCloudEventFactoryBase(event CloudEventContentAdapter, eventType string, payload interface{}) *CloudEventFactoryBase {
	return &CloudEventFactoryBase{
		event:     event,
		eventType: eventType,
		payload:   payload,
	}
}

func (f *CloudEventFactoryBase) CreateCloudEvent() (*cloudevents.Event, error) {
	ev := cloudevents.NewEvent()
	ev.SetSource(event.GetEventSource())
	ev.SetDataContentType(cloudevents.ApplicationJSON)
	ev.SetType(f.eventType)
	ev.SetExtension("shkeptncontext", f.event.GetShKeptnContext())

	err := ev.SetData(cloudevents.ApplicationJSON, f.payload)
	if err != nil {
		return nil, fmt.Errorf("could not marshal cloud event payload: %v", err)
	}

	return &ev, nil
}
