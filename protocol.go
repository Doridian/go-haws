package haws

import (
	"encoding/json"
	"errors"
	"fmt"
	"sync/atomic"
)

type wsCmd struct {
	ID   uint64 `json:"id,omitempty"`
	Type string `json:"type"`
}

type wsErr struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e *wsErr) ToError() error {
	return fmt.Errorf("[%s] %s", e.Code, e.Message)
}

type wsCmdAuth struct {
	wsCmd
	AccessToken string `json:"access_token"`
}

type wsResp struct {
	wsCmd
	Type    string          `json:"type"`
	Success *bool           `json:"success"`
	Result  json.RawMessage `json:"result"`
	Event   *EventData      `json:"event"`
	Error   *wsErr          `json:"error"`
}

func (c *Client) reader() {
	defer c.readerWait.Done()

	for c.running {
		msg := wsResp{}
		err := c.conn.ReadJSON(&msg)

		switch msg.Type {
		case "result":
			c.respHandlerLock.Lock()
			respHandler := c.respHandlers[msg.ID]
			delete(c.respHandlers, msg.ID)
			c.respHandlerLock.Unlock()
			if respHandler != nil {
				var respErr error
				if msg.Error != nil {
					respErr = msg.Error.ToError()
				} else if msg.Success != nil && !*msg.Success {
					respErr = errors.New("unknown error response from HA")
				} else if respHandler.out != nil {
					respErr = json.Unmarshal(msg.Result, respHandler.out)
				}
				respHandler.errChan <- respErr
			}
		case "event":
			c.onEvent(msg.Event)
		case "auth_required":
			c.sendNoWait(&wsCmdAuth{
				wsCmd: wsCmd{
					Type: "auth",
				},
				AccessToken: c.token,
			})
		case "auth_ok":
			c.authOk = true
		}

		if err != nil {
			c.handleError(err)
			return
		}
	}
}

func (c *Client) sendNoWait(cmd interface{}) error {
	return c.send(cmd, nil, nil)
}

func (c *Client) sendAndWait(cmd interface{}, result interface{}) error {
	errChan := make(chan error)
	defer close(errChan)

	err := c.send(cmd, result, errChan)
	if err != nil {
		return err
	}

	return <-errChan
}

func (c *Client) send(cmd interface{}, result interface{}, errChan chan error) error {
	vWithID, ok := cmd.(*wsCmd)
	if ok {
		vWithID.ID = atomic.AddUint64(&c.lastEventID, 1)
	}
	err := c.conn.WriteJSON(cmd)
	if err != nil {
		err = c.handleError(err)
		return err
	}

	if errChan != nil {
		c.respHandlerLock.Lock()
		c.respHandlers[vWithID.ID] = &respHandler{
			errChan: errChan,
			out:     result,
		}
		c.respHandlerLock.Unlock()
	}

	return nil
}
