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

	// go queuePack("CR-HOLLAND|NEW", 9434, &pw) // tondemo skill
	// go queuePack("CR-HOLLAND|NEW", 9431, &pw) // ningen
	// go queuePack("CR-HOLLAND|NEW", 9428, &pw) // cool danshi
	// go queuePack("CR-HOLLAND|NEW", 9426, &pw) // kubo
	// go queuePack("CR-HOLLAND|NEW", 8642, &pw) // 1080p
	// go queuePack("CR-HOLLAND|NEW", 8641, &pw) // kubo 1080p

	return &pw
}

func main() {
	pw := runXdcc()

	go trackers.InitTrackers(pw)

	// res, _ := search.GetSeriesPacks("Detective Conan")

	// for _, p := range res {
	// 	botName, id := search.GetBotName(p.BotId), p.Id
	// 	go queuePack(botName, id, pw)
	// }

	wg := sync.WaitGroup{}
	wg.Add(1)
	wg.Wait()
}

// add a better ready flag
// theres too many sleeps. use more channels and events or sm
// sometimes downloads stall. auto restart them
// add something for in-terminal reporting
// config file
// add cron tasks to scan for new anime to download
// make a thing that parses said now anime to/from file
// make speed readings more accurate
// connection will sometimes hand before tls connection, do the events and recreate the client if need be ****
// "Auto-ignore activated for USERNAME (USERNAME!~IP thing) lasting 1m50s. Further messages will increase duration."
// "You already requested that pack"
