package marathoner

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"text/template"
	"time"
)

// haproxyConfigContext defines context for haproxy config template
type haproxyConfigContext struct {
	Bind string
	Apps map[int]HaproxyApp
}

// HaproxyApp has port and list of servers for that port
type HaproxyApp struct {
	Port    int
	Servers []HaproxyServer
	Labels  map[string]string
}

// HaproxyServer has host and port where working service is located
type HaproxyServer struct {
	Host string
	Port int
}

// HaproxyConfigurator implements ConfiguratorImplementation for haproxy
type HaproxyConfigurator struct {
	state    State
	apps     map[int]HaproxyApp
	mutex    sync.Mutex
	template *template.Template
	conf     string
	bind     string
	pidfile  string
	timeout time.Duration
}

// NewHaproxyConfigurator creates configurator with specified config template,
// config file path, bind location, pidfile location and timeout
// to keep previous haproxy running after reload
func NewHaproxyConfigurator(template *template.Template, conf string, bind string, pidfile string, timeout time.Duration) *HaproxyConfigurator {
	return &HaproxyConfigurator{
		state:    nil,
		apps:     nil,
		mutex:    sync.Mutex{},
		template: template,
		conf:     conf,
		bind:     bind,
		pidfile:  pidfile,
		timeout: timeout,
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

	err = c.template.Execute(temp, haproxyConfigContext{
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

// reloadHaproxy gracefully reloads haproxy and starts haproxy if needed
func (c *HaproxyConfigurator) reloadHaproxy() error {
	log.Println("reloading haproxy, really..")

	if _, err := os.Stat(c.pidfile); os.IsNotExist(err) {
		log.Println("pid file not exists, starting haproxy")
		return c.startHaproxy()
	}

	p, err := ioutil.ReadFile(c.pidfile)
	if err != nil {
		return err
	}

	pid, err := strconv.Atoi(strings.TrimSpace(string(p)))
	if err != nil {
		return err
	}

	err = syscall.Kill(pid, 0)
	if err != nil {
		if err != syscall.ESRCH {
			return err
		}

		// process died somewhere
		log.Println("haproxy process not exists, starting haproxy")
		return c.startHaproxy()
	}

	c.scheduleTermination(pid)

	cmd := exec.Command("haproxy", "-D", "-f", c.conf, "-p", c.pidfile, "-sf", strconv.Itoa(pid))
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("error reloading conf: %s, output: %s", err, string(out))
	}

	return nil
}

// startHaproxy starts haproxy process in the background
func (c *HaproxyConfigurator) startHaproxy() error {
	return exec.Command("haproxy", "-D", "-f", c.conf, "-p", c.pidfile).Run()
}

// scheduleTermination schedules termination with sigterm
// of a previous haproxy instance after configured amount of time
func (c *HaproxyConfigurator) scheduleTermination(pid int) {
	log.Printf("scheduling termination for haproxy with pid %d in %s\n", pid, c.timeout)
	deadline := time.Now().Add(c.timeout)

	go func() {
		for {
			err := syscall.Kill(pid, 0)
			if err != nil {
				if err == syscall.ESRCH {
					log.Printf("haproxy with pid %d gracefully exited before we killed it\n", pid)
				} else {
					log.Println("error checking that haproxy with pid %d is still running: %s\n", pid, err)
				}

				break
			}

			if time.Now().After(deadline) {
				log.Printf("killing haproxy with pid %d with sigterm\n", pid)
				err := syscall.Kill(pid, syscall.SIGTERM)
				if err != nil {
					log.Printf("error killing haproxy with pid %d: %s\n", pid, err)
				}

				break
			}

			time.Sleep(time.Second)
		}
	}()
}

// stateToApps converts marathon state to haproxy apps
func stateToApps(s State) map[int]HaproxyApp {
	r := map[int]HaproxyApp{}

	for _, a := range s {
		for i, p := range a.Ports {
			if v, ok := a.Labels["marathoner_haproxy_enabled"]; !ok || (v != "true" && v != "1") {
				continue
			}

			app := HaproxyApp{
				Port:    p,
				Servers: []HaproxyServer{},
				Labels:  a.Labels,
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
