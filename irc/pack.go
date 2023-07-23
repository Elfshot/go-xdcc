package irc

import (
	"strconv"
	"strings"

	"github.com/Elfshot/go-xdcc/config"
	"github.com/Elfshot/go-xdcc/progress"
	log "github.com/sirupsen/logrus"
)

var packQueue = make(chan *queue, config.GetConfig().MaxDownloads)
var packs = make([]*Pack, 0)

func removeFinishedPack(pack *Pack) {
	for i, v := range packs {
		if v.FileName == pack.FileName {
			packs = append(packs[:i], packs[i+1:]...)
			return
		}
	}
}

func QueuePack(pack *Pack, monitor *progress.Monitor) {
	if findPack(pack.FileName) != nil {
		log.Debugf("Pack %s already in queue", pack.FileName)
		return
	}

	packQueue <- &queue{
		pack:    pack,
		monitor: monitor,
	}

	packs = append(packs, pack)
}

func getPack(bot string, packNum int, jobs chan *session) (quit chan bool) {
	quit, ircClient := getIrc(jobs, 0)

	ircClient.Privmsg(bot, "xdcc send #"+strconv.Itoa(packNum))

	return quit
}

func findPack(packName string) *Pack {
	for _, pack := range packs {
		if strings.Contains(pack.FileName, packName) {
			return pack
		}
	}
	return nil
}
