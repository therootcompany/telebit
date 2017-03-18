package genericlistener

import (
	"strconv"
	"time"

	"git.daplie.com/Daplie/go-rvpn-server/rvpn/packer"

	"sync"

	"io"

	"context"

	"encoding/hex"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
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

	// Connect Time
	connectTime time.Time

	//lastUpdate
	lastUpdate time.Time

	//initialDomains - a list of domains from the JWT
	initialDomains []interface{}

	connectionTrack *Tracking

	///wssState tracks a highlevel status of the connection, false means do nothing.
	wssState bool

	//connectionID
	connectionID int64
}

//NewConnection -- Constructor
func NewConnection(connectionTable *Table, conn *websocket.Conn, remoteAddress string, initialDomains []interface{}, connectionTrack *Tracking) (p *Connection) {
	connectionID = connectionID + 1

	p = new(Connection)
	p.mutex = &sync.Mutex{}
	p.connectionTable = connectionTable
	p.conn = conn
	p.source = remoteAddress
	p.bytesIn = 0
	p.bytesOut = 0
	p.send = make(chan *SendTrack)
	p.connectTime = time.Now()
	p.initialDomains = initialDomains
	p.connectionTrack = connectionTrack
	p.DomainTrack = make(map[string]*DomainTrack)

	for _, domain := range initialDomains {
		p.AddTrackedDomain(string(domain.(string)))
	}

	p.State(true)
	p.connectionID = connectionID
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

//Source --
func (c *Connection) Source() string {
	return c.source
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

//LastUpdate -- retrieve last update
func (c *Connection) LastUpdate() time.Time {
	return c.lastUpdate
}

//ConnectionID - Get
func (c *Connection) ConnectionID() int64 {
	return c.connectionID
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
func (c *Connection) Reader(ctx context.Context) {
	connectionTrack := c.connectionTrack

	defer func() {
		c.connectionTable.unregister <- c
		c.conn.Close()
		loginfo.Println("reader defer", c)
	}()

	loginfo.Println("Reader Start ", c)

	c.conn.SetReadLimit(65535)
	for {
		msgType, message, err := c.conn.ReadMessage()

		loginfo.Println("ReadMessage", msgType, err)

		c.Update()

		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway) {
				c.State(false)
				loginfo.Printf("error: %v", err)
			}
			break
		}

		// unpack the message.
		p, err := packer.ReadMessage(message)
		key := p.Header.Address().String() + ":" + strconv.Itoa(p.Header.Port)
		track, err := connectionTrack.Lookup(key)

		loginfo.Println(hex.Dump(p.Data.Data()))

		if err != nil {
			loginfo.Println("Unable to locate Tracking for ", key)
			continue
		}

		//Support for tracking outbound traffic based on domain.
		if domainTrack, ok := c.DomainTrack[track.domain]; ok {
			//if ok then add to structure, else warn there is something wrong
			domainTrack.AddIn(int64(len(message)))
		}

		track.conn.Write(p.Data.Data())

		c.addIn(int64(len(message)))
		loginfo.Println("end of read")
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
