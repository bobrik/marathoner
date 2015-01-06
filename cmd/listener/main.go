package main

import (
	"flag"
	"github.com/bobrik/marathoner"
	"strings"
)

func main() {
	u := flag.String("u", "127.0.0.1:7676", "updater location")
	p := flag.String("p", "", "pidfile of haproxy")
	c := flag.String("c", "/etc/haproxy/haproxy.cfg", "haproxy config path")
	b := flag.String("b", "127.0.0.1", "ip address to bind")
	flag.Parse()

	if *p == "" {
		flag.PrintDefaults()
		return
	}

	conf := marathoner.NewHaproxyConfigurator(*c, *b, *p)

	l := marathoner.NewListener(strings.Split(*u, ","), conf)
	l.Start()
}
