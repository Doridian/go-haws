package haws

import (
	"math"
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

	authWaitChan  chan bool
	authOk        bool
	authWaitTimer *time.Timer
	authTimeout   time.Duration

	respHandlerLock sync.Mutex
	respHandlers    map[uint64]*respHandler

	eventHandlerLock sync.Mutex
	eventHandlers    map[string]EventHandler

	reconnectTime  time.Duration
	allowReconnect bool
}

func NewClient(url string, token string, reconnectTime time.Duration) *Client {
	cl := &Client{
		url:   url,
		token: token,
		hdr:   http.Header{},

		authTimeout: time.Second * 5,

		respHandlers:  make(map[uint64]*respHandler),
		eventHandlers: make(map[string]EventHandler),

		reconnectTime:  reconnectTime,
		allowReconnect: false,
	}

	cl.authWaitTimer = time.AfterFunc(time.Duration(math.MaxInt64), cl.authError)

	return cl
}
