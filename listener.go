package marathoner

import (
	"errors"
	"log"
	"math/rand"
	"net"
	"net/rpc"
	"time"
)

// Listener listens for configuration updates and applies them
type Listener struct {
	updaters []string
	conf     ConfiguratorImplementation
	rand     *rand.Rand
}

// NewListener creates new listener for specified updater
// and configurator implementation
func NewListener(updaters []string, conf ConfiguratorImplementation) *Listener {
	return &Listener{
		updaters: updaters,
		conf:     conf,
		rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// Start runs infinite listener loop
func (l *Listener) Start() {
	for {
		c, err := l.dialUpdater()
		if err != nil {
			log.Println("connection error", err)
			time.Sleep(time.Second * 3)
			continue
		}

		s := rpc.NewServer()
		s.Register(&Configurator{l.conf})

		s.ServeConn(c)

		log.Println("disconnected from updater, sleeping")
		time.Sleep(time.Second * 3)
	}
}

// dialUpdater connects to random updater
func (l *Listener) dialUpdater() (net.Conn, error) {
	for _, i := range l.rand.Perm(len(l.updaters)) {
		resp, err := net.Dial("tcp", l.updaters[i])
		if err != nil {
			log.Println("error connecting to updater " + l.updaters[i] + ", " + err.Error())
			continue
		}

		log.Println("dial succeeded", l.updaters[i])

		return resp, nil
	}

	return nil, errors.New("all updater endpoints are unreachable")
}
