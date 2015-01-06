package marathoner

import (
	"log"
	"net"
	"reflect"
	"sync"
	"time"
)

// Updater is update coordinator
type Updater struct {
	mutex   sync.Mutex
	apps    State
	updates chan State
	clients map[string]chan State
}

// NewUpdater creates new updater
func NewUpdater() *Updater {
	return &Updater{
		mutex:   sync.Mutex{},
		apps:    nil,
		clients: map[string]chan State{},
	}
}

// ListenForUpdates starts listening for marathon state updates
// at specified marathon uri and with specified interval
func (u *Updater) ListenForUpdates(marathon []string, interval time.Duration) {
	m := NewMarathon(marathon)

	for {
		log.Println("getting state from marathon..")
		s, err := m.State()
		if err != nil {
			log.Println("error getting marathon state", err)
		} else {
			u.update(s)
		}

		time.Sleep(interval)
	}
}

// update updates internal state and sends updates to all connected listeners
func (u *Updater) update(s State) {
	u.mutex.Lock()

	if reflect.DeepEqual(u.apps, s) {
		u.mutex.Unlock()
		return
	}

	u.apps = s

	clients := u.clients
	u.mutex.Unlock()

	log.Printf("distributing update among %d clients\n", len(clients))

	wg := sync.WaitGroup{}
	for n, c := range clients {
		wg.Add(1)

		go func(n string, c chan State) {
			select {
			case c <- s:
				break
			case <-time.After(time.Second * 10):
				log.Println("client " + n + " failed to respond in 10s, closing channel")
				u.mutex.Lock()
				delete(u.clients, n)
				u.mutex.Unlock()

				close(c)
			}

			wg.Done()
		}(n, c)
	}

	wg.Wait()
}

// ListenForClients starts listening for rpc clients on specified location
func (u *Updater) ListenForClients(listen string) error {
	l, err := net.Listen("tcp", listen)
	if err != nil {
		return err
	}

	for {
		c, err := l.Accept()
		if err != nil {
			log.Println("error accepting connection", err)
			continue
		}

		go func() {
			err := u.handleConnection(NewClient(c))
			if err != nil {
				log.Println("error handling connection:", err)
			}
		}()
	}
}

// handleConnection handles connection with a client
func (u *Updater) handleConnection(c *client) error {
	defer func() {
		u.mutex.Lock()
		delete(u.clients, c.name)
		u.mutex.Unlock()

		c.Close()
	}()

	u.mutex.Lock()
	apps := u.apps

	// no apps -> no updates, closing instantly
	if apps == nil {
		u.mutex.Unlock()
		return nil
	}

	ch := make(chan State)
	u.clients[c.name] = ch

	u.mutex.Unlock()

	err := c.reload(apps)
	if err != nil {
		return err
	}

	for update := range ch {
		err := c.reload(update)
		if err != nil {
			return err
		}
	}

	return nil
}
