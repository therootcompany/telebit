package genericlistener

import "net"
import "context"
import "fmt"

//Track -- used to track connection + domain
type Track struct {
	conn   net.Conn
	domain string
}

//NewTrack -- Constructor
func NewTrack(conn net.Conn, domain string) (p *Track) {
	p = new(Track)
	p.conn = conn
	p.domain = domain
	return
}

//Tracking --
type Tracking struct {
	connections map[string]*Track
	register    chan *Track
	unregister  chan net.Conn
}

//NewTracking -- Constructor
func NewTracking() (p *Tracking) {
	p = new(Tracking)
	p.connections = make(map[string]*Track)
	p.register = make(chan *Track)
	p.unregister = make(chan net.Conn)
	return
}

//Run -
func (p *Tracking) Run(ctx context.Context) {
	loginfo.Println("Tracking Running")

	for {
		select {

		case <-ctx.Done():
			loginfo.Println("Cancel signal hit")
			return

		case connection := <-p.register:
			key := connection.conn.RemoteAddr().String()
			loginfo.Println("register fired", key)
			p.connections[key] = connection
			p.list()

		case connection := <-p.unregister:
			key := connection.RemoteAddr().String()
			loginfo.Println("unregister fired", key)
			if _, ok := p.connections[key]; ok {
				delete(p.connections, key)
			}
			p.list()
		}
	}
}

func (p *Tracking) list() {
	for c := range p.connections {
		loginfo.Println(c)
	}
}

//Lookup --
// - get connection from key
func (p *Tracking) Lookup(key string) (c *Track, err error) {
	if _, ok := p.connections[key]; ok {
		c = p.connections[key]
	} else {
		err = fmt.Errorf("Lookup failed for %s", key)
		c = nil
	}
	return
}
