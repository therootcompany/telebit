package main

import (
	"log"
	"net/http"

	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

// Connection track websocket and faciliates in and out data
type Connection struct {
	connectionTable *ConnectionTable

	// The websocket connection.
	conn *websocket.Conn

	// Buffered channel of outbound messages.
	send chan []byte

	// Address of the Remote End Point
	source string

	// admin flag.  Grants access to admin features
	admin bool

	// bytes in
	bytesIn int64

	// bytes out
	bytesOut int64
}

func (c *Connection) addIn(num int64) {
	c.bytesIn = c.bytesIn + num
}

func (c *Connection) addOut(num int64) {
	c.bytesOut = c.bytesOut + num
}

func (c *Connection) reader() {
	defer func() {
		c.connectionTable.unregister <- c
		c.conn.Close()
	}()
	c.conn.SetReadLimit(maxMessageSize)
	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway) {
				log.Printf("error: %v", err)
			}
			break
		}
		loginfo.Println(message)
		c.addIn(int64(len(message)))

		loginfo.Println(c)
	}
}

func (c *Connection) writer() {
	dwell := time.NewTicker(5 * time.Second)
	loginfo.Println("activate timer", dwell)
	defer func() {
		c.conn.Close()
	}()
	for {
		select {

		case message := <-c.send:
			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			if err := w.Close(); err != nil {
				return
			}

			c.addOut(int64(len(message)))
		}
	}
}

func (c *Connection) sender() {
	dwell := time.NewTicker(5 * time.Second)
	loginfo.Println("activate timer", dwell)
	defer func() {
		c.conn.Close()
	}()
	for {
		select {
		case <-dwell.C:
			loginfo.Println("Dwell Activated")
			c.send <- []byte("This is a test")
		}
	}
}

// handleConnectionWebSocket handles websocket requests from the peer.
func handleConnectionWebSocket(connectionTable *ConnectionTable, w http.ResponseWriter, r *http.Request, admin bool) {
	loginfo.Println("websocket opening ", r.RemoteAddr)

	if admin == true {
		loginfo.Println("Recognized Admin connection, waiting authentication")
	} else {
		loginfo.Println("Recognized connection, waiting authentication")
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		loginfo.Println("WebSocket upgrade failed", err)
		return
	}
	connection := &Connection{connectionTable: connectionTable, conn: conn, send: make(chan []byte, 256), source: r.RemoteAddr, admin: admin}
	connection.connectionTable.register <- connection
	go connection.writer()
	go connection.sender()
	connection.reader()
}
