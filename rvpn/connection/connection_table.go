package connection

//Table maintains the set of connections
type Table struct {
	connections map[*Connection]bool
	register    chan *Connection
	unregister  chan *Connection
}

//NewTable -- consructor
func NewTable() *Table {
	return &Table{
		register:    make(chan *Connection),
		unregister:  make(chan *Connection),
		connections: make(map[*Connection]bool),
	}
}

//Run -- Execute
func (c *Table) Run() {
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
			loginfo.Println("closing connection ", connection)
			if _, ok := c.connections[connection]; ok {
				delete(c.connections, connection)
				close(connection.send)

			}
		}
	}
}

//Register -- Property
func (c *Table) Register() (r chan *Connection) {
	r = c.register
	return
}
