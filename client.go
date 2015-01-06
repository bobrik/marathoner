package marathoner

import (
	"log"
	"net"
	"net/rpc"
)

// client is rpc client to update configs on remote servers
type client struct {
	name string
	rc   *rpc.Client
}

// NewClient create client with given net.Conn
func NewClient(conn net.Conn) *client {
	return &client{
		name: conn.RemoteAddr().String(),
		rc:   rpc.NewClient(conn),
	}
}

// reload updates config on remote server
func (c *client) reload(s State) error {
	reloaded := false
	err := c.rc.Call("Configurator.Update", s, &reloaded)
	if err != nil {
		return err
	}

	if reloaded {
		log.Println("reloaded config on " + c.name)
	} else {
		log.Println("not reloaded config on " + c.name)
	}

	return nil
}

// Close closes underlying connection of a client
func (c *client) Close() error {
	return c.rc.Close()
}
