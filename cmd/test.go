package cmd

import (
	"github.com/Elfshot/go-xdcc/search"
	"github.com/spf13/cobra"

	log "github.com/sirupsen/logrus"
)

func init() {
	trackerCmd.AddCommand(versionCmd)
}

var versionCmd = &cobra.Command{
	Use:   "test",
	Short: "Test the connection with xdcc source",
	Run: func(cmd *cobra.Command, args []string) {
		res, err := search.GetPacksLastest(10)

		if err != nil || len(res) != 10 {
			log.Fatal(err)
		}
	},
}
