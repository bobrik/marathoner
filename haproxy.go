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
	apps    map[int]HaproxyApp
	mutex   sync.Mutex
	conf    string
	bind    string
	pidfile string
}

// NewHaproxyConfigurator creates configurator with specified config file
// path, bind location and pidfile location
func NewHaproxyConfigurator(conf string, bind string, pidfile string) *HaproxyConfigurator {
	return &HaproxyConfigurator{
		apps:    nil,
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

	apps := stateToApps(s)

	if reflect.DeepEqual(apps, c.apps) {
		log.Println("state is the same, not doing any updates")
		*r = false
		return nil
	}

	c.apps = apps

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
		Apps: c.apps,
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

	return r
}
