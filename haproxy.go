package marathoner

import (
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"reflect"
	"strings"
	"sync"
	"text/template"
)

const haproxyConfigTemplate = `
global
  log 127.0.0.1 local0
  log 127.0.0.1 local1 notice
  stats socket /etc/haproxy/haproxy.sock level admin
  maxconn 4096

defaults
  log             global
  retries         3
  maxconn         2000
  timeout connect 5000
  timeout client  50000
  timeout server  50000

{{ $bind := .Bind }}

{{ range $app := .Apps }}
	listen app-{{ $app.Port }}
		bind {{ $bind }}:{{ $app.Port }}
		mode tcp
		option tcplog
		balance leastconn

		{{ range $server := $app.Servers }}
		server {{ $server.Host }}-{{ $server.Port }} {{ $server.Host }}:{{ $server.Port }} check
		{{ end }}
{{ end }}
`

// haproxyConfigContext defines context for haproxy config template
type haproxyConfigContext struct {
	Bind string
	Apps map[int]HaproxyApp
}

// HaproxyApp has port and list of servers for that port
type HaproxyApp struct {
	Port    int
	Servers []HaproxyServer
}

// HaproxyServer has host and port where working service is located
type HaproxyServer struct {
	Host string
	Port int
}

// HaproxyConfigurator implements ConfiguratorImplementation for haproxy
type HaproxyConfigurator struct {
	state   State
	mutex   sync.Mutex
	conf    string
	bind    string
	pidfile string
}

// NewHaproxyConfigurator creates configurator with specified config file
// path, bind location and pidfile location
func NewHaproxyConfigurator(conf string, bind string, pidfile string) *HaproxyConfigurator {
	return &HaproxyConfigurator{
		state:   nil,
		mutex:   sync.Mutex{},
		conf:    conf,
		bind:    bind,
		pidfile: pidfile,
	}
}

// Update runs actually update and logs error if it happens
func (c *HaproxyConfigurator) Update(s State, r *bool) error {
	err := c.update(s, r)
	if err != nil {
		log.Println("error updating configuration:", err)
	}

	return err
}

// Update updates haproxy config and reloads haproxy if needed
func (c *HaproxyConfigurator) update(s State, r *bool) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	log.Println("received update request")

	if reflect.DeepEqual(s, c.state) {
		log.Println("state is the same, not doing any updates")
		*r = false
		return nil
	}

	c.state = rearrangeTasks(c.state, s)

	err := c.updateConfig()
	if err != nil {
		log.Fatal(err)
		return err
	}

	log.Println("config updated")

	err = c.checkHaproxyConfig()
	if err != nil {
		return err
	}

	log.Println("config validity checked")

	err = c.reloadHaproxy()
	if err != nil {
		return err
	}

	log.Println("haproxy reloaded")

	*r = true
	return nil
}

// updateConfig writes new config for haproxy
// if template can be parsed and executed
func (c *HaproxyConfigurator) updateConfig() error {
	temp, err := os.Create(c.conf + ".next")
	if err != nil {
		return err
	}

	defer temp.Close()

	t := template.New("config")
	t.Funcs(template.FuncMap{
		"replace": func(old, new, s string) string {
			return strings.Replace(s, old, new, -1)
		},
	})

	p, err := t.Parse(haproxyConfigTemplate)
	if err != nil {
		return err
	}

	err = p.Execute(temp, haproxyConfigContext{
		Bind: c.bind,
		Apps: stateToApps(c.state),
	})

	if err != nil {
		return err
	}

	return os.Rename(temp.Name(), c.conf)
}

// checkHaproxyConfig checks if written haproxy config is valid
func (c *HaproxyConfigurator) checkHaproxyConfig() error {
	_, err := exec.Command("haproxy", "-c", "-f", c.conf).CombinedOutput()
	return err
}

// reloadHaproxy gracefully reloads haproxy and schedules
// haproxy killing in the future to ensure that haproxy
// processes do not live for too long after they are replaced
func (c *HaproxyConfigurator) reloadHaproxy() error {
	log.Println("reloading haproxy, really..")

	p, err := ioutil.ReadFile(c.pidfile)
	if err != nil {
		return err
	}

	cmd := exec.Command("haproxy", "-D", "-f", c.conf, "-p", c.pidfile, "-sf", strings.TrimSpace(string(p)))
	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Fatal("error: " + err.Error() + ", out: " + string(out))
		return err
	}

	return nil
}

// stateToApps converts marathon state to haproxy apps
func stateToApps(s State) map[int]HaproxyApp {
	r := map[int]HaproxyApp{}

	for _, a := range s {
		for i, p := range a.Ports {
			// separate app for each task
			if e, ok := a.Labels["marathoner_port_range"]; ok && e == "true" {
				for j, t := range a.Tasks {
					app := HaproxyApp{
						Port: p + j,
						Servers: []HaproxyServer{
							HaproxyServer{
								Host: t.Host,
								Port: t.Ports[i],
							},
						},
					}

					r[app.Port] = app
				}
			} else {
				app := HaproxyApp{
					Port:    p,
					Servers: []HaproxyServer{},
				}

				for _, t := range a.Tasks {
					server := HaproxyServer{
						Host: t.Host,
						Port: t.Ports[i],
					}

					app.Servers = append(app.Servers, server)
				}

				r[app.Port] = app
			}
		}
	}

	return r
}

func rearrangeTasks(prev, s State) State {
	if prev == nil {
		return s
	}

	r := map[string]App{}

	for i, a := range s {
		// no need for port range - skipping
		if e, ok := a.Labels["marathoner_port_range"]; !ok || e != "true" {
			r[i] = a
			continue
		}

		// new app - skipping
		if _, ok := prev[i]; !ok {
			r[i] = a
			continue
		}

		prevPositions := make(map[string]int, len(prev[i].Tasks))
		for i, t := range prev[i].Tasks {
			prevPositions[t.ID] = i
		}

		currPositions := make(map[string]int, len(a.Tasks))
		for i, t := range a.Tasks {
			currPositions[t.ID] = i
		}

		remaining := make(map[string]Task)
		for _, t := range a.Tasks {
			remaining[t.ID] = t
		}

		places := make(map[int]Task, len(a.Tasks))

		// placing old tasks to their prev places
		// if they are still present in new version
		for id, i := range prevPositions {
			if _, ok := remaining[id]; !ok {
				continue
			}

			places[i] = a.Tasks[currPositions[id]]
			delete(remaining, id)
		}

		// placing remaining new tasks in the holes
		tasks := make([]Task, len(a.Tasks))
		for i := 0; i < len(a.Tasks); i++ {
			if t, ok := places[i]; ok {
				tasks[i] = t
				continue
			}

			found := false
			for id, t := range remaining {
				tasks[i] = t
				found = true
				delete(remaining, id)
				break
			}

			if !found {
				log.Println("we screwed rearrangement, discarding it")
				log.Printf("prev = %#v\n", prev)
				log.Printf("s = %#v\n", s)
				return s
			}
		}

		rearranged := App{
			Name:   a.Name,
			Labels: a.Labels,
			Ports:  a.Ports,
			Tasks:  tasks,
		}

		r[i] = rearranged
	}

	return r
}
