package genericlistener

import (
	"context"
	"encoding/hex"
	"io"
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"git.daplie.com/Daplie/go-rvpn-server/rvpn/packer"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
}

// Connection track websocket and faciliates in and out data
type Connection struct {
	mutex sync.Mutex

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

	// requests
	requests int64

	// response
	responses int64

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
	p.connectionTable = connectionTable
	p.conn = conn
	p.source = remoteAddress
	p.bytesIn = 0
	p.bytesOut = 0
	p.requests = 0
	p.responses = 0
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
func (c *Connection) InitialDomains() []interface{} {
	return c.initialDomains
}

//ConnectTime -- Property
func (c *Connection) ConnectTime() time.Time {
	return c.connectTime
}

//BytesIn -- Property
func (c *Connection) BytesIn() int64 {
	return c.bytesIn
}

//BytesOut -- Property
func (c *Connection) BytesOut() int64 {
	return c.bytesOut
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

func (c *Connection) addRequests() {
	c.requests = c.requests + 1
}

func (c *Connection) addResponse() {
	c.responses = c.responses + 1
}

//ConnectionTable -- property
func (c *Connection) ConnectionTable() *Table {
	return c.connectionTable
}

//GetState -- Get state of Socket...this is a high level state.
func (c *Connection) GetState() bool {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	return c.wssState
}

//State -- Set the set of the high level connection
func (c *Connection) State(state bool) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.wssState = state
}

//Update -- updates the lastUpdate property tracking idle time
func (c *Connection) Update() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

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
func (c *Connection) NextWriter(wssMessageType int) (io.WriteCloser, error) {
	if c.GetState() == true {
		return c.conn.NextWriter(wssMessageType)
	}

	// Is returning a nil error actually the proper thing to do here?
	loginfo.Println("NextWriter aborted, state is not true")
	return nil, nil
}

//Write -- Wrapper to allow a high level state check before allowing a write to the socket.
func (c *Connection) Write(w io.WriteCloser, message []byte) (int, error) {
	if c.GetState() == true {
		return w.Write(message)
	}

	// Is returning a nil error actually the proper thing to do here?
	return 0, nil
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
			domainTrack.AddOut(int64(len(message)))
			domainTrack.AddResponses()
		}

		track.conn.Write(p.Data.Data())

		c.addIn(int64(len(message)))
		c.addResponse()
		loginfo.Println("end of read")
	}
}

//Writer -- expoer the writer function
func (c *Connection) Writer() {
	defer c.conn.Close()

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
			c.addRequests()

			//Support for tracking outbound traffic based on domain.
			if domainTrack, ok := c.DomainTrack[message.domain]; ok {
				//if ok then add to structure, else warn there is something wrong
				domainTrack.AddIn(messageLen)
				domainTrack.AddRequests()
				loginfo.Println("adding ", messageLen, " to ", message.domain)
			} else {
				logdebug.Println("attempting to add bytes to ", message.domain, "it does not exist")
				logdebug.Println(c.DomainTrack)
			}
			loginfo.Println(c)
		}
	}
}
