package trackers

import (
	"fmt"
	"os"
	"time"

	"github.com/Elfshot/go-xdcc/config"
	"github.com/Elfshot/go-xdcc/irc"
	"github.com/Elfshot/go-xdcc/progress"
	"github.com/Elfshot/go-xdcc/search"
	"github.com/go-co-op/gocron"

	log "github.com/sirupsen/logrus"
)

func InitTrackers(pw *progress.Monitor) {
	cfg := config.GetConfig()
	cron := gocron.NewScheduler(time.Local)
	cron.Every(cfg.DownloadInterval).Minutes().Do(runTrackers, pw)
	cron.StartBlocking()
}

func runTrackers(pw *progress.Monitor) {
	cfg := config.GetConfig()
	cfg.LoadTrackers()
	trackers := cfg.Trackers

	for _, tracker := range trackers {
		time.Sleep(5 * time.Second)
		packs, err := search.GetSeriesPacks(tracker.SearchName)
		if err != nil {
			continue
		}
		for _, pack := range packs {
			if pack.EpisodeNumber < tracker.EpisodeRange[0] || pack.EpisodeNumber > tracker.EpisodeRange[1] {
				continue
			}
			var newEp int
			// if the episode number is greater than the number of episodes in the season
			if pack.EpisodeNumber > (tracker.EpisodeRange[1] - tracker.EpisodeRange[0]) {
				// subtract the lower bound of the episode range from the episode number
				// (-1 because the episode range is 1-indexed)
				newEp = pack.EpisodeNumber - (tracker.EpisodeRange[0] - 1)
			} else {
				newEp = pack.EpisodeNumber
			}

			botN, err := search.GetBotName(pack.BotId)

			if err != nil {
				log.Error(err)
				continue
			}

			var sPack *irc.Pack = &irc.Pack{
				BotNick:     botN,
				FileName:    pack.Name,
				ModFileName: fmt.Sprintf("S%dE%d.mkv", tracker.Season, newEp),
				Size:        0,
				ShowName:    tracker.FileName,
				Season:      tracker.Season,
				Episode:     pack.EpisodeNumber,
				PackNumber:  pack.Id,
			}

			// Only download if the file doesn't already exist
			if _, err := os.Stat(sPack.GetFileDir()); err == nil {
				log.Debugf("File %s already exists | Skipping download", sPack.FileName)
				continue
			}

			irc.QueuePack(sPack, pw)
		}
	}
}
