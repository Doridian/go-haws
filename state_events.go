package haws

import (
	"encoding/json"
	"log"
	"sync"
)

type StateChangeEvent struct {
	EntityID string `json:"entity_id"`
	NewState *State `json:"new_state"`
	OldState *State `json:"old_state"`
}

type StateChangeHandler interface {
	OnStateChange(event *StateChangeEvent)
}

type StateChangeEventHandler struct {
	eventHandlerLock sync.Mutex
	eventHandlers    map[string]StateChangeHandler
	defaultHandler   StateChangeHandler
	client           *Client
}

func NewStateChangeEventHandler(client *Client) (*StateChangeEventHandler, error) {
	res := &StateChangeEventHandler{
		eventHandlers: make(map[string]StateChangeHandler),
		client:        client,
	}
	return res, client.AddEventHandler(EventStateChanged, res)
}

func (c *StateChangeEventHandler) AddHandler(entityID string, handler StateChangeHandler) {
	c.eventHandlerLock.Lock()
	defer c.eventHandlerLock.Unlock()

	c.eventHandlers[entityID] = handler
}

func (c *StateChangeEventHandler) SetDefaultHandler(handler StateChangeHandler) {
	c.defaultHandler = handler
}

func (c *StateChangeEventHandler) OnEvent(eventData *EventData) {
	evt := &StateChangeEvent{}
	err := json.Unmarshal(eventData.Data, evt)
	if err != nil {
		log.Printf("Invalid state change event JSON: %v", err)
		return
	}

	c.eventHandlerLock.Lock()
	handler := c.eventHandlers[evt.EntityID]
	c.eventHandlerLock.Unlock()

	if handler == nil {
		handler = c.defaultHandler
		if handler == nil {
			return
		}
	}
	go handler.OnStateChange(evt)
}
