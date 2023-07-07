package main

import (
	"os"
	"sync"

	"github.com/Elfshot/go-xdcc/irc"
	"github.com/Elfshot/go-xdcc/progress"
	"github.com/Elfshot/go-xdcc/trackers"

	log "github.com/sirupsen/logrus"
)

func initLog() {
	logLevel := os.Getenv("LOG_LEVEL")

	switch logLevel {
	case "DEBUG":
		log.SetLevel(log.DebugLevel)
	case "INFO":
		log.SetLevel(log.InfoLevel)
	default:
		log.SetLevel(log.ErrorLevel)
	}
}

func runXdcc() *progress.Monitor {
	initLog()
	var pw progress.Monitor
	pw.Init()
	go irc.QueueLoop()

	return &pw
}

func main() {
	pw := runXdcc()

	go trackers.InitTrackers(pw)

	wg := sync.WaitGroup{}
	wg.Add(1)
	wg.Wait()
}

// add a better ready flag
// theres too many sleeps. use more channels and events or sm
// sometimes downloads stall. auto restart them
// connection will sometimes hang before tls connection, do the events and recreate the client if need be ****
// "Auto-ignore activated for USERNAME (USERNAME!~IP thing) lasting 1m50s. Further messages will increase duration."
// "You already requested that pack"
