package haws

import (
	"log"
	"time"

	"github.com/gorilla/websocket"
)

func (c *Client) openConditional(checkIfRunning bool) error {
	c.connLock.Lock()
	defer c.connLock.Unlock()

	if checkIfRunning && c.running {
		return nil
	}

	dialer := websocket.Dialer{}
	ws, _, err := dialer.Dial(c.url, c.hdr.Clone())
	if err != nil {
		return err
	}

	c.running = true
	c.readerWait.Add(1)
	go c.reader()

	c.conn = ws
	return nil
}

func (c *Client) Open() error {
	return c.openConditional(false)
}

func (c *Client) closeNoLock() {
	c.running = false
	if c.conn != nil {
		c.conn.Close()

	}
	c.readerWait.Wait()

	c.authOk = false
	c.conn = nil
}

func (c *Client) Close() error {
	c.connLock.Lock()
	defer c.connLock.Unlock()

	c.closeNoLock()
	return nil
}

func (c *Client) handleError(err error) error {
	if err == nil {
		return nil
	}
	go c.handleErrorSync(err)
	return err
}

func (c *Client) handleErrorSync(err error) {
	log.Printf("Error in WS: %v", err)
	go c.timedReconnect()
}

func (c *Client) timedReconnect() {
	c.Close()
	if c.reconnectTime <= 0 {
		return
	}
	time.Sleep(c.reconnectTime)
	c.openConditional(true)
}
