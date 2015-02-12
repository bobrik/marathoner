package main

import (
	"flag"
	"fmt"
	"github.com/bobrik/marathoner"
	"os"
	"strings"
	"time"
)

type stdOutStateLogger struct{}

func (s stdOutStateLogger) Write(p []byte) (n int, err error) {
	t := time.Now().Format("2006-01-02T15:04:05.999999999Z0700") // iso8601
	return os.Stdout.Write([]byte(fmt.Sprintf("%s: %s\n", t, string(p))))
}

func main() {
	u := flag.String("u", "127.0.0.1:7676", "updater location")
	flag.Parse()

	c := marathoner.NewStateLogger(stdOutStateLogger{})

	l := marathoner.NewListener(strings.Split(*u, ","), c)
	l.Start()
}
