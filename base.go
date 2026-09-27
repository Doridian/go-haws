package haws

import (
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type ConnectionState string

const (
	STATE_DISCONNECTED ConnectionState = "disconnected"
	STATE_CONNECTING   ConnectionState = "connecting"
	STATE_CONNECTED    ConnectionState = "connected"
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

	state ConnectionState

	authDone      bool
	authOk        bool
	authWaitTimer *time.Timer
	authTimeout   time.Duration

	respHandlerLock sync.Mutex
	respHandlers    map[uint64]*respHandler

	eventHandlerLock sync.Mutex
	eventHandlers    map[string]EventHandler
	stateHandler     func(state ConnectionState)

	reconnectTime  time.Duration
	allowReconnect bool
}

func NewClient(url string, token string, stateHandler func(state ConnectionState), reconnectTime time.Duration) *Client {
	cl := &Client{
		url:   url,
		token: token,
		hdr:   http.Header{},

		authTimeout:  time.Second * 5,
		stateHandler: stateHandler,

		respHandlers:  make(map[uint64]*respHandler),
		eventHandlers: make(map[string]EventHandler),

		reconnectTime:  reconnectTime,
		allowReconnect: false,

		state: STATE_DISCONNECTED,
	}

	return cl
}

func (c *Client) GetState() ConnectionState {
	return c.state
}
