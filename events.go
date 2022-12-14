package haws

import (
	"encoding/json"
)

const (
	EventStateChanged string = "state_changed"
)

type EventData struct {
	Data      json.RawMessage `json:"data"`
	EventType string          `json:"event_type"`
}

type wsCommandSubscribeEvents struct {
	wsCmd

	EventType string `json:"event_type"`
}

type EventHandler interface {
	OnEvent(eventData *EventData)
}

func (c *Client) subscribe(eventType string) error {
	return c.sendAndWait(&wsCommandSubscribeEvents{
		wsCmd: wsCmd{
			Type: "subscribe_events",
		},
		EventType: eventType,
	}, nil)
}

func (c *Client) AddEventHandler(eventType string, handler EventHandler) error {
	c.eventHandlerLock.Lock()
	oldHandler := c.eventHandlers[eventType]
	c.eventHandlers[eventType] = handler
	c.eventHandlerLock.Unlock()

	if !c.authOk {
		return nil
	}

	if oldHandler == nil {
		err := c.subscribe(eventType)
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *Client) resubscribe() error {
	c.eventHandlerLock.Lock()
	defer c.eventHandlerLock.Unlock()

	for eventType := range c.eventHandlers {
		err := c.subscribe(eventType)
		if err != nil {
			return c.handleError(err)
		}
	}

	return nil
}

func (c *Client) onEvent(event *EventData) {
	c.eventHandlerLock.Lock()
	defer c.eventHandlerLock.Unlock()

	handler := c.eventHandlers[event.EventType]
	if handler != nil {
		go handler.OnEvent(event)
	}
}
