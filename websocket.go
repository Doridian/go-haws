package haws

import (
	"errors"
	"log"
	"math"
	"time"

	"github.com/gorilla/websocket"
)

func (c *Client) openConditional(checkIfRunning bool) error {
	c.connLock.Lock()
	defer c.connLock.Unlock()

	if checkIfRunning && c.running {
		return nil
	}

	c.closeNoLock()

	dialer := &websocket.Dialer{
		HandshakeTimeout: time.Second * 5,
	}
	ws, _, err := dialer.Dial(c.url, c.hdr.Clone())
	if err != nil {
		return err
	}

	c.authWaitTimer = time.AfterFunc(time.Duration(math.MaxInt64), c.authError)

	c.authDone = false
	c.allowReconnect = c.reconnectTime > 0
	c.running = true
	c.conn = ws

	c.readerWait.Add(1)
	go c.reader()

	c.reconnectHandler()

	return nil
}

func (c *Client) authError() {
	c.handleError(errors.New("auth timeout"))
}

func (c *Client) WaitAuth() error {
	for {
		c.connLock.Lock()
		authDone := c.authDone
		c.connLock.Unlock()
		if authDone {
			break
		}
		time.Sleep(time.Millisecond * 100)
	}

	if !c.authOk {
		return errors.New("auth failure")
	}
	return nil
}

func (c *Client) Open() error {
	return c.openConditional(false)
}

func (c *Client) stopAuthWaitTimer() {
	if c.authWaitTimer == nil {
		return
	}
	c.authWaitTimer.Stop()
	c.authWaitTimer = nil
}

func (c *Client) authWaitDone() {
	c.stopAuthWaitTimer()

	c.authDone = true
}

func (c *Client) closeNoLock() {
	c.stopAuthWaitTimer()

	c.running = false
	if c.conn != nil {
		c.conn.Close()
	}

	c.authOk = false
	if !c.allowReconnect {
		c.authDone = true
	}

	c.readerWait.Wait()
	c.conn = nil
}

func (c *Client) Close() error {
	c.allowReconnect = false
	return c.close()
}

func (c *Client) close() error {
	c.connLock.Lock()
	defer c.connLock.Unlock()
	c.closeNoLock()
	return nil
}

func (c *Client) handleError(err error) error {
	if err == nil {
		return nil
	}
	log.Printf("Error in WS: %v", err)
	go c.timedReconnect()
	return err
}

func (c *Client) timedReconnect() {
	c.close()
	if !c.allowReconnect {
		return
	}
	time.Sleep(c.reconnectTime)
	c.openConditional(true)
}
