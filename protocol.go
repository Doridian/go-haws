package haws

import (
	"encoding/json"
	"errors"
	"fmt"
	"sync/atomic"
)

type ErrPriorToAuth struct {
	error
}

type wsCmd struct {
	ID   uint64 `json:"id,omitempty"`
	Type string `json:"type"`
}

type wsIDCmd interface {
	SetID(id uint64)
	GetID() uint64

	GetType() string
}

func (c *wsCmd) GetID() uint64 {
	return c.ID
}

func (c *wsCmd) GetType() string {
	return c.Type
}

func (c *wsCmd) SetID(id uint64) {
	c.ID = id
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
			c.connLock.Lock()
			c.authOk = true
			c.authWaitDone()
			c.connLock.Unlock()

			go c.resubscribe()
		}

		if err != nil {
			c.handleError(err)
			return
		}
	}
}

func (c *Client) sendNoWait(cmd wsIDCmd) error {
	return c.send(cmd, nil, nil)
}

func (c *Client) sendAndWait(cmd wsIDCmd, result interface{}) error {
	errChan := make(chan error)
	defer close(errChan)

	err := c.send(cmd, result, errChan)
	if err != nil {
		return err
	}

	return <-errChan
}

func (c *Client) send(cmd wsIDCmd, result interface{}, errChan chan error) error {
	id := cmd.GetID()
	cmdType := cmd.GetType()

	cmdIsAuth := cmdType == "auth"

	if !cmdIsAuth {
		id = atomic.AddUint64(&c.lastEventID, 1)
		cmd.SetID(id)
	}

	if !c.authOk && !cmdIsAuth {
		return &ErrPriorToAuth{
			error: fmt.Errorf("tried to send command %s prior to auth", cmd.GetType()),
		}
	}

	c.connLock.Lock()
	err := c.conn.WriteJSON(cmd)
	c.connLock.Unlock()

	if err != nil {
		err = c.handleError(err)
		return err
	}

	if errChan != nil {
		c.respHandlerLock.Lock()
		c.respHandlers[id] = &respHandler{
			errChan: errChan,
			out:     result,
		}
		c.respHandlerLock.Unlock()
	}

	return nil
}
