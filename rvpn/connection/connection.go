package connection

import (
	"encoding/hex"

	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

// Connection track websocket and faciliates in and out data
type Connection struct {
	connectionTable *Table

	// The websocket connection.
	conn *websocket.Conn

	// Buffered channel of outbound messages.
	send chan []byte

	// Address of the Remote End Point
	source string

	// bytes in
	bytesIn int64

	// bytes out
	bytesOut int64
}

//NewConnection -- Constructor
func NewConnection(connectionTable *Table, conn *websocket.Conn, remoteAddress string) (p *Connection) {
	p = new(Connection)
	p.connectionTable = connectionTable
	p.conn = conn
	p.source = remoteAddress
	p.bytesIn = 0
	p.bytesOut = 0
	p.send = make(chan []byte, 256)
	return
}

func (c *Connection) addIn(num int64) {
	c.bytesIn = c.bytesIn + num
}

func (c *Connection) addOut(num int64) {
	c.bytesOut = c.bytesOut + num
}

//ConnectionTable -- property
func (c *Connection) ConnectionTable() (table *Table) {
	table = c.connectionTable
	return
}

//Reader -- export the reader function
func (c *Connection) Reader() {
	defer func() {
		c.connectionTable.unregister <- c
		c.conn.Close()
	}()
	c.conn.SetReadLimit(1024)
	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway) {
				loginfo.Printf("error: %v", err)
			}
			break
		}
		loginfo.Println(hex.Dump(message))
		c.addIn(int64(len(message)))
		loginfo.Println(c)
	}
}

//Writer -- expoer the writer function
func (c *Connection) Writer() {
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
