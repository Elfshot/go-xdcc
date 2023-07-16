package cmd

import (
	"fmt"
	"os"
	"sync"

	"github.com/Elfshot/go-xdcc/irc"
	"github.com/Elfshot/go-xdcc/progress"
	"github.com/Elfshot/go-xdcc/trackers"
	"github.com/spf13/cobra"
)

func runXdcc() *progress.Monitor {
	var pw progress.Monitor
	pw.Init()
	go irc.QueueLoop()

	return &pw
}

func runTracker() {
	pw := runXdcc()

	go trackers.InitTrackers(pw)

	wg := sync.WaitGroup{}
	wg.Add(1)
	wg.Wait()
}

var trackerCmd = &cobra.Command{
	Use:   "tracker",
	Short: "Run the tracker",
	Run: func(cmd *cobra.Command, args []string) {
		runTracker()
	},
}

func Execute() {
	if err := trackerCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
