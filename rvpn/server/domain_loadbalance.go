package server

import (
	"fmt"
	"sync"
)

const (
	lbmUnSupported      string = "unsuported"
	lbmRoundRobin       string = "round-robin"
	lbmLeastConnections string = "least-connections"
)

//DomainLoadBalance -- Use as a structure for domain connections
//and load balancing those connections.  Initial modes are round-robin
//but suspect we will need least-connections, and sticky
type DomainLoadBalance struct {
	mutex sync.Mutex

	//lb method, supported round robin.
	method string

	//the last connection based on calculation
	lastmember int

	// a list of connections in this load balancing context
	connections []*Connection

	//a counter to track total connections, so we aren't calling len all the time
	count int

	//true if the system belives a recalcuation is required
	recalc bool
}

//NewDomainLoadBalance -- Constructor
func NewDomainLoadBalance(defaultMethod string) (p *DomainLoadBalance) {
	p = new(DomainLoadBalance)
	p.method = defaultMethod
	p.lastmember = 0
	p.count = 0
	return
}

//Connections -- Access connections
func (p *DomainLoadBalance) Connections() []*Connection {
	return p.connections
}

//NextMember -- increments the lastmember, and then checks if >= to count, if true
//the last is reset to 0
func (p *DomainLoadBalance) NextMember() (conn *Connection) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	//check for round robin, if not RR then drop out and call calculate
	loginfo.Println("NextMember:", p)
	if p.method == lbmRoundRobin {
		p.lastmember++
		if p.lastmember >= p.count {
			p.lastmember = 0
		}
		nextConn := p.connections[p.lastmember]
		return nextConn
	}

	// Not round robin
	switch method := p.method; method {
	default:
		panic(fmt.Errorf("fatal unsupported loadbalance method %s", method))
	}
}

//AddConnection -- Add an additional connection to the list of connections for this domain
//this should not affect the next member calculation in RR.  However it many in other
//methods
func (p *DomainLoadBalance) AddConnection(conn *Connection) []*Connection {
	loginfo.Println("AddConnection", fmt.Sprintf("%p", conn))
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.connections = append(p.connections, conn)
	p.count++
	loginfo.Println("AddConnection", p)
	return p.connections
}

//RemoveConnection -- removes a matching connection from the list. This may
//affect the nextmember calculation if found so the recalc flag is set.
func (p *DomainLoadBalance) RemoveConnection(conn *Connection) {
	loginfo.Println("RemoveConnection", fmt.Sprintf("%p", conn))

	p.mutex.Lock()
	defer p.mutex.Unlock()

	//scan all the connections
	for pos := range p.connections {
		loginfo.Println("RemoveConnection", pos, len(p.connections), p.count)
		if p.connections[pos] == conn {
			//found connection remove it
			loginfo.Printf("found connection %p", conn)
			p.connections[pos], p.connections = p.connections[len(p.connections)-1], p.connections[:len(p.connections)-1]
			p.count--
			break
		}
	}
	loginfo.Println("RemoveConnection:", p)
}
