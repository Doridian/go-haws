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

	c.authWaitTimer.Reset(c.authTimeout)

	if c.authWaitChan == nil {
		c.authWaitChan = make(chan bool)
	}
	c.allowReconnect = c.reconnectTime > 0
	c.running = true
	c.conn = ws

	c.readerWait.Add(1)
	go c.reader()

	return nil
}

func (c *Client) authError() {
	c.handleError(errors.New("auth timeout"))
}

func (c *Client) WaitAuth() error {
	c.connLock.Lock()
	ch := c.authWaitChan
	c.connLock.Unlock()

	<-ch
	if !c.authOk {
		return errors.New("auth failure")
	}
	return nil
}

func (c *Client) Open() error {
	return c.openConditional(false)
}

func (c *Client) authWaitDone() {
	c.authWaitTimer.Reset(time.Duration(math.MaxInt64))
	if c.authWaitChan == nil {
		return
	}
	close(c.authWaitChan)
	c.authWaitChan = nil
}

func (c *Client) closeNoLock() {
	c.authWaitTimer.Reset(time.Duration(math.MaxInt64))

	c.running = false
	if c.conn != nil {
		c.conn.Close()
	}

	c.authOk = false
	if !c.allowReconnect {
		c.authWaitDone()
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
