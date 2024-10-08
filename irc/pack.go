package irc

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/Elfshot/go-xdcc/config"
	"github.com/Elfshot/go-xdcc/progress"
	log "github.com/sirupsen/logrus"
)

type Pack struct {
	BotNick     string
	FileName    string
	ModFileName string
	Size        int
	ShowName    string
	Season      int
	Episode     int
	PackNumber  int

	// * Not always given
	Crc32   string
	Version int
}

// TODO: Get rid of this and replace with redis or something
var packQueue = make(chan *queue, config.GetConfig().MaxDownloads)
var packs = make([]*Pack, 0)

func (p Pack) GetFileDir() string {
	dir := config.GetConfig().DownloadDir

	if config.GetConfig().SkipOrganization {
		return dir + p.FileName
	}
	sep := string(os.PathSeparator)
	showDir := dir + p.ShowName + sep
	seasonDir := showDir + "Season " + fmt.Sprint(p.Season) + sep
	versionAppend := ""

	if p.Version > 1 {
		versionAppend = fmt.Sprintf("v%d", p.Version)
	}
	if _, err := os.Stat(seasonDir); os.IsNotExist(err) {
		os.MkdirAll(seasonDir, 0777)
	}

	return seasonDir + p.ModFileName + versionAppend + ".mkv"
}

func (p Pack) DelOldVersions() {
	pack := p
	version := pack.Version
	for i := version - 1; i > 0; i-- {
		pack.Version = i
		if _, err := os.Stat(pack.GetFileDir()); err == nil {
			log.Debugf("Deleting old version of <%s - %d>, Ver <%d>", pack.ShowName, pack.Episode, pack.Version)
			os.Remove(pack.GetFileDir())
		}
	}
}

func removeFinishedPack(pack *Pack) {
	for i, v := range packs {
		if v.FileName == pack.FileName {
			packs = append(packs[:i], packs[i+1:]...)
			return
		}
	}
}

func QueuePack(pack *Pack, monitor *progress.Monitor, attempt uint8) {
	if attempt > config.GetConfig().MaxDlRetries {
		log.Errorf("Max retries reached for pack %s", pack.FileName)

		// Add pack to list to ensure it doesn't retry
		packs = append(packs, pack)
		return
	}

	if findPack(pack.FileName) != nil {
		log.Debugf("Pack %s already in queue", pack.FileName)
		return
	}

	packQueue <- &queue{
		pack:    pack,
		monitor: monitor,
		attempt: attempt,
	}

	packs = append(packs, pack)
}

func ReQueuePack(pack *Pack, monitor *progress.Monitor, attempt uint8) {
	time.Sleep(180 * time.Second)
	QueuePack(pack, monitor, attempt+1)
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
