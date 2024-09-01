package cli

import (
	"github.com/Elfshot/go-xdcc/client/tui"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "goxdcc",
	Short: "goxdcc is a TUI for downloading files from XDCC bots.",
	Run: func(cmd *cobra.Command, args []string) {
		tui.DoStuff()
	},
}
