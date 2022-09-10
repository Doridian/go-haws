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

	c.closeNoLock()

	ws, _, err := websocket.DefaultDialer.Dial(c.url, c.hdr.Clone())
	if err != nil {
		return err
	}

	c.authWaitChan = make(chan bool)
	c.allowReconnect = true
	c.running = true
	c.conn = ws

	c.readerWait.Add(1)
	go c.reader()

	return nil
}

func (c *Client) WaitAuth() {
	<-c.authWaitChan
}

func (c *Client) Open() error {
	return c.openConditional(false)
}

func (c *Client) authWaitDone() {
	if c.authWaitChan == nil {
		return
	}
	close(c.authWaitChan)
}

func (c *Client) closeNoLock() {
	c.running = false
	if c.conn != nil {
		c.conn.Close()
	}

	c.authOk = false
	c.authWaitDone()

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
	if c.reconnectTime <= 0 || !c.allowReconnect {
		return
	}
	time.Sleep(c.reconnectTime)
	c.openConditional(true)
}
