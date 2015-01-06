package main

import (
	"flag"
	"github.com/bobrik/marathoner"
	"log"
	"strings"
	"time"
)

func main() {
	l := flag.String("l", "0.0.0.0:7676", "listen for clients")
	m := flag.String("m", "http://127.0.0.1:8080", "maraton location")
	i := flag.Float64("i", 1.0, "update interval")
	flag.Parse()

	u := marathoner.NewUpdater()

	go u.ListenForUpdates(strings.Split(*m, ","), time.Duration(*i)*time.Second)

	err := u.ListenForClients(*l)
	if err != nil {
		log.Fatal(err)
	}
}
