package haws

import (
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type respHandler struct {
	errChan chan error
	out     interface{}
}

type Client struct {
	lastEventID uint64

	url   string
	token string
	hdr   http.Header

	running    bool
	readerWait sync.WaitGroup
	connLock   sync.Mutex
	conn       *websocket.Conn
	authOk     bool

	respHandlerLock sync.Mutex
	respHandlers    map[uint64]*respHandler

	eventHandlerLock sync.Mutex
	eventHandlers    map[string]EventHandler

	reconnectTime  time.Duration
	allowReconnect bool
}

func NewClient(url string, token string, reconnectTime time.Duration) *Client {
	return &Client{
		url:            url,
		token:          token,
		hdr:            http.Header{},
		respHandlers:   make(map[uint64]*respHandler),
		reconnectTime:  reconnectTime,
		allowReconnect: true,
	}
}
