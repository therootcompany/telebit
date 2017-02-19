package connection

import (
	"encoding/hex"
	"time"

	"sync"

	"io"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

// Connection track websocket and faciliates in and out data
type Connection struct {
	mutex *sync.Mutex

	// The main connection table (should be just one of these created at startup)
	connectionTable *Table

	//used to track traffic for a domain.  Not use for lookup or validation only for tracking
	DomainTrack map[string]*DomainTrack

	// The websocket connection.
	conn *websocket.Conn

	// Buffered channel of outbound messages.
	send chan *SendTrack

	// WssState channel
	// Must check state via channel before xmit

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

	//lastUpdate
	lastUpdate time.Time

	//initialDomains - a list of domains from the JWT
	initialDomains []interface{}

	///wssState tracks a highlevel status of the connection, false means do nothing.
	wssState bool
}

//NewConnection -- Constructor
func NewConnection(connectionTable *Table, conn *websocket.Conn, remoteAddress string, initialDomains []interface{}) (p *Connection) {
	p = new(Connection)
	p.mutex = &sync.Mutex{}
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

	p.State(true)
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

//GetState -- Get state of Socket...this is a high level state.
func (c *Connection) GetState() bool {
	defer func() {
		c.mutex.Unlock()
	}()
	c.mutex.Lock()
	return c.wssState
}

//State -- Set the set of the high level connection
func (c *Connection) State(state bool) {
	defer func() {
		c.mutex.Unlock()
	}()

	c.mutex.Lock()
	c.wssState = state
}

//Update -- updates the lastUpdate property tracking idle time
func (c *Connection) Update() {
	defer func() {
		c.mutex.Unlock()
	}()

	c.mutex.Lock()
	c.lastUpdate = time.Now()
}

//NextWriter -- Wrapper to allow a high level state check before offering NextWriter
//The libary failes if client abends during write-cycle.  a fast moving write is not caught before socket state bubbles up
//A synchronised state is maintained
func (c Connection) NextWriter(wssMessageType int) (w io.WriteCloser, err error) {
	if c.GetState() == true {
		w, err = c.conn.NextWriter(wssMessageType)
	} else {
		loginfo.Println("NextWriter aborted, state is not true")
	}
	return
}

//Write -- Wrapper to allow a high level state check before allowing a write to the socket.
func (c *Connection) Write(w io.WriteCloser, message []byte) (cnt int, err error) {
	if c.GetState() == true {
		cnt, err = w.Write(message)
	}
	return
}

//Reader -- export the reader function
func (c *Connection) Reader() {
	defer func() {
		c.connectionTable.unregister <- c
		c.conn.Close()
		loginfo.Println("reader defer", c)
	}()

	loginfo.Println("Reader Start ", c)

	c.conn.SetReadLimit(1024)
	for {
		_, message, err := c.conn.ReadMessage()

		loginfo.Println("ReadMessage")
		c.Update()

		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway) {
				c.State(false)
				loginfo.Printf("error: %v", err)
				loginfo.Println(c.conn)
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

	loginfo.Println("Writer Start ", c)

	for {
		select {

		case message := <-c.send:
			w, err := c.NextWriter(websocket.BinaryMessage)
			loginfo.Println("next writer ", w)
			if err != nil {
				return
			}
			loginfo.Println(c)
			loginfo.Println(w)

			c.Update()

			_, err = c.Write(w, message.data)

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
