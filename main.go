package main

import (
	"os"

	cmd "github.com/Elfshot/go-xdcc/cmd"
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

func main() {
	initLog()

	cmd.Execute()
}

// TODO List
/*
add a better ready flag
theres too many sleeps. use more channels and events or sm
sometimes downloads stall. auto restart them
connection will sometimes hang before tls connection, do the events and recreate the client if need be ****
"Auto-ignore activated for USERNAME (USERNAME!~IP thing) lasting 1m50s. Further messages will increase duration."
"You already requested that pack"
Apply config defaults
check CRC checksums after resumes (optionally downloads)
*/
