package main

import (
	"flag"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"text/template"
	"time"

	"github.com/bobrik/marathoner"
)

func main() {
	u := flag.String("u", "127.0.0.1:7676", "updater location")
	p := flag.String("p", "", "pidfile of haproxy")
	t := flag.String("t", "", "config template path")
	c := flag.String("c", "/etc/haproxy/haproxy.cfg", "haproxy config path")
	b := flag.String("b", "127.0.0.1", "ip address to bind")
	m := flag.Int("m", 60, "maximum number seconds to keep previous haproxy running")
	flag.Parse()

	if *p == "" || *t == "" {
		flag.PrintDefaults()
		os.Exit(1)
	}

	timeout := time.Duration(*m) * time.Second

	ct, err := readTemplate(*t)
	if err != nil {
		log.Fatal("error reading template:", err)
	}

	conf := marathoner.NewHaproxyConfigurator(ct, *c, *b, *p, timeout)

	l := marathoner.NewListener(strings.Split(*u, ","), conf)
	l.Start()
}

// readTemplate reads haproxy config template from a file
func readTemplate(file string) (*template.Template, error) {
	tf, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, err
	}

	return template.New("config").Parse(string(tf))
}
