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
	// The main connection table (should be just one of these created at startup)
	connectionTable *Table

	//used to track traffic for a domain.  Not use for lookup or validation only for tracking
	DomainTrack map[string]*DomainTrack

	// The websocket connection.
	conn *websocket.Conn

	// Buffered channel of outbound messages.
	send chan *SendTrack

	// Address of the Remote End Point
	source string

	// bytes in
	bytesIn int64

	// bytes out
	bytesOut int64

	// communications channel between go routines
	commCh chan bool

	// Connect Time
	connectTime time.Time

	//initialDomains - a list of domains from the JWT
	initialDomains []interface{}
}

//NewConnection -- Constructor
func NewConnection(connectionTable *Table, conn *websocket.Conn, remoteAddress string, initialDomains []interface{}) (p *Connection) {
	p = new(Connection)
	p.connectionTable = connectionTable
	p.conn = conn
	p.source = remoteAddress
	p.bytesIn = 0
	p.bytesOut = 0
	p.send = make(chan *SendTrack)
	p.commCh = make(chan bool)
	p.connectTime = time.Now()
	p.initialDomains = initialDomains
	p.DomainTrack = make(map[string]*DomainTrack)

	for _, domain := range initialDomains {
		p.AddTrackedDomain(string(domain.(string)))
	}
	return
}

//AddTrackedDomain -- Add a tracked domain
func (c *Connection) AddTrackedDomain(domain string) {
	p := new(DomainTrack)
	p.DomainName = domain
	c.DomainTrack[domain] = p
}

//InitialDomains -- Property
func (c *Connection) InitialDomains() (i []interface{}) {
	i = c.initialDomains
	return
}

//ConnectTime -- Property
func (c *Connection) ConnectTime() (t time.Time) {
	t = c.connectTime
	return
}

//BytesIn -- Property
func (c *Connection) BytesIn() (b int64) {
	b = c.bytesIn
	return
}

//BytesOut -- Property
func (c *Connection) BytesOut() (b int64) {
	b = c.bytesOut
	return
}

//SendCh -- property to sending channel
func (c *Connection) SendCh() chan *SendTrack {
	return c.send
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

//CommCh -- Property
func (c *Connection) CommCh() chan bool {
	return c.commCh
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
			w.Write(message.data)

			if err := w.Close(); err != nil {
				return
			}

			messageLen := int64(len(message.data))

			c.addOut(messageLen)

			//Support for tracking outbound traffic based on domain.
			if domainTrack, ok := c.DomainTrack[message.domain]; ok {
				//if ok then add to structure, else warn there is something wrong
				domainTrack.AddOut(messageLen)
				loginfo.Println("adding ", messageLen, " to ", message.domain)
			} else {
				logdebug.Println("attempting to add bytes to ", message.domain, "it does not exist")
				logdebug.Println(c.DomainTrack)
			}
			loginfo.Println(c)
		}
	}
}
