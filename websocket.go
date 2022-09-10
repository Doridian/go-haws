package haws

import (
	"log"

	"github.com/gorilla/websocket"
)

func (c *Client) Open() error {
	c.connLock.Lock()
	defer c.connLock.Unlock()

	c.closeNoLock()

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
	c.Close()
	// TODO: Reconnect?
}
