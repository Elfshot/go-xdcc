package progress

import (
	"flag"
	"time"

	progress "github.com/jedib0t/go-pretty/v6/progress"
)

var (
	flagAutoStop           = flag.Bool("auto-stop", false, "Auto-stop rendering?")
	flagHideETA            = flag.Bool("hide-eta", false, "Hide the ETA?")
	flagHideETAOverall     = flag.Bool("hide-eta-overall", false, "Hide the ETA in the overall tracker?")
	flagHideOverallTracker = flag.Bool("hide-overall", true, "Hide the Overall Tracker?")
	flagHidePercentage     = flag.Bool("hide-percentage", false, "Hide the progress percent?")
	flagHideTime           = flag.Bool("hide-time", false, "Hide the time taken?")
	flagHideValue          = flag.Bool("hide-value", true, "Hide the tracker value?")
	flagShowSpeed          = flag.Bool("show-speed", true, "Show the tracker speed?")
	flagShowSpeedOverall   = flag.Bool("show-speed-overall", true, "Show the overall tracker speed?")

	/*
		messageColors = []text.Color{
			text.FgRed,
			text.FgGreen,
			text.FgYellow,
			text.FgBlue,
			text.FgMagenta,
			text.FgCyan,
			text.FgWhite,
		}
	*/
)

type Monitor struct {
	writer progress.Writer
}

func (monitor *Monitor) Add(name string, total int) *progress.Tracker {
	tracker := &progress.Tracker{
		Message: name,
		Total:   int64(total),
		Units:   progress.UnitsBytes,
	}
	monitor.writer.AppendTracker(tracker)

	return tracker
}

func (monitor *Monitor) Init() {
	pw := progress.NewWriter()
	pw.SetAutoStop(*flagAutoStop)
	pw.SetTrackerLength(25)
	pw.SetMessageWidth(75)
	pw.SetNumTrackersExpected(3)
	pw.SetSortBy(progress.SortBy(progress.PositionLeft))
	pw.SetStyle(progress.StyleDefault)
	pw.SetTrackerPosition(progress.PositionRight)
	pw.SetUpdateFrequency(time.Millisecond * 100)
	pw.Style().Colors = progress.StyleColorsExample
	pw.Style().Options.PercentFormat = "%4.1f%%"
	pw.Style().Visibility.ETA = !*flagHideETA
	pw.Style().Visibility.ETAOverall = !*flagHideETAOverall
	pw.Style().Visibility.Percentage = !*flagHidePercentage
	pw.Style().Visibility.Speed = *flagShowSpeed
	pw.Style().Visibility.SpeedOverall = *flagShowSpeedOverall
	pw.Style().Visibility.Time = !*flagHideTime
	pw.Style().Visibility.TrackerOverall = !*flagHideOverallTracker
	pw.Style().Visibility.Value = !*flagHideValue
	//// pw.writer.SetPinnedMessages("Downloading Your Anime!")

	monitor.writer = pw
	go monitor.writer.Render()
}
