package cproxy

import (
	"errors"
	"fmt"
	"github.com/gorilla/websocket"
	"io"

	"strings"
	"sync"
	"time"
)

// Conn wraps websocket.Conn with Conn. It defines to listen and read
// data from Conn.
type Conn struct {
	Conn *websocket.Conn

	//AfterReadFunc   func(messageType int, r io.Reader)
	AfterReadFunc   func(messageType int, data string)
	BeforeCloseFunc func()

	once   sync.Once
	id     string
	stopCh chan struct{}
}

// Write write p to the websocket connection. The error returned will always
// be nil if success.
func (c *Conn) Write(p []byte) (n int, err error) {
	select {
	case <-c.stopCh:
		return 0, errors.New("Conn is closed, can't be written")
	default:
		err = c.Conn.WriteMessage(websocket.TextMessage, p)
		if err != nil {
			return 0, err
		}
		return len(p), nil
	}
}

// Listen listens for receive data from websocket connection. It blocks
// until websocket connection is closed.
func (c *Conn) Listen() {
	c.Conn.SetCloseHandler(func(code int, text string) error {
		if c.BeforeCloseFunc != nil {
			c.BeforeCloseFunc()
		}

		if err := c.Close(); err != nil {
			log.Println(err)
		}

		message := websocket.FormatCloseMessage(code, "")
		c.Conn.WriteControl(websocket.CloseMessage, message, time.Now().Add(time.Second))
		return nil
	})

	// Keeps reading from Conn util get error.
ReadLoop:
	for {
		select {
		case <-c.stopCh:
			break ReadLoop
		default:
			messageType, r, err := c.Conn.NextReader()
			if err != nil {
				log.Println("read from socket error:", err)
				// TODO: handle read error maybe
				break ReadLoop
			}
			var sb strings.Builder
			buf := make([]byte, 256)
			for {
				n, err := r.Read(buf)
				if err != nil {
					if err != io.EOF {
						fmt.Println("read error:", err)
					}
					break
				}
				sb.Write(buf[:n])
			}
			if c.AfterReadFunc != nil {
				c.AfterReadFunc(messageType, sb.String())
			}
		}
	}
}

// Close close the connection.
func (c *Conn) Close() error {
	select {
	case <-c.stopCh:
		return errors.New("Conn already been closed")
	default:
		c.Conn.Close()
		close(c.stopCh)
		return nil
	}
}

// NewConn wraps conn.
func NewConn(conn *websocket.Conn) *Conn {
	return &Conn{
		Conn:   conn,
		stopCh: make(chan struct{}),
	}
}
