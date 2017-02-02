package main

// ConnectionTable maintains the set of connections
type ConnectionTable struct {
	connections map[*Connection]bool
	register    chan *Connection
	unregister  chan *Connection
}

func newConnectionTable() *ConnectionTable {
	return &ConnectionTable{
		register:    make(chan *Connection),
		unregister:  make(chan *Connection),
		connections: make(map[*Connection]bool),
	}
}

func (c *ConnectionTable) run() {
	loginfo.Println("ConnectionTable starting")
	for {
		select {
		case connection := <-c.register:
			loginfo.Println("register fired")
			c.connections[connection] = true

			for conn := range c.connections {
				loginfo.Println(conn)
			}

		case connection := <-c.unregister:
			if _, ok := c.connections[connection]; ok {
				delete(c.connections, connection)
				close(connection.send)
			}
		}
	}
}
