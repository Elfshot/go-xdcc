package main

import (
	"os"
	"runtime/debug"

	cmd "github.com/Elfshot/go-xdcc/cmd"
	log "github.com/sirupsen/logrus"
)

func initLog() {
	logLevel := os.Getenv("LOG_LEVEL")

	switch logLevel {
	case "DEBUG":
		log.SetLevel(log.DebugLevel)
	case "ERROR":
		log.SetLevel(log.ErrorLevel)
	default:
		log.SetLevel(log.InfoLevel)
	}
}

func setRtOps() {
	debug.SetGCPercent(20)
}

func main() {
	setRtOps()
	initLog()

	cmd.Execute()
}

// TODO List
/*
sometimes downloads stall. auto restart them
"Auto-ignore activated for USERNAME (USERNAME!~IP thing) lasting 1m50s. Further messages will increase duration."
"You already requested that pack"
Apply config defaults
force http to use specified laddr too -> https://stackoverflow.com/questions/50870994/use-dial-in-golang-with-specific-local-address
replace the sketchky progress checker with one that watches the progress package's progress in a goroutine
*/
