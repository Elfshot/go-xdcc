package irc

import (
	"fmt"
	"net"
	"os"
	"time"

	"github.com/Elfshot/go-xdcc/config"
	"github.com/Elfshot/go-xdcc/progress"
)

type transfer struct {
	status          int
	unixStart       int64
	unixEnd         int64
	startBytes      int
	transferedBytes int
	targetPort      int
	targetIp        net.IP
	isResume        bool
	events          chan *events
}

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
	Crc32 string
}

func (p Pack) GetFileDir() string {
	dir := config.GetConfig().DownloadDir
	sep := string(os.PathSeparator)
	showDir := dir + p.ShowName + sep
	seasonDir := showDir + "Season " + fmt.Sprint(p.Season) + sep

	if _, err := os.Stat(seasonDir); os.IsNotExist(err) {
		os.MkdirAll(seasonDir, 0777)
	}

	return seasonDir + p.ModFileName
}

type events struct {
	EventType    int
	PackData     *Pack
	TransferData *transfer
}

type queue struct {
	pack    *Pack
	monitor *progress.Monitor
}

type counterT struct {
	count int
	limit int
}

func (c *counterT) inc() {
	c.count++
}

func (c *counterT) dec() {
	c.count--
}

func (c *counterT) wait() {
	for {
		if c.count < c.limit {
			return
		}
		time.Sleep(500 * time.Millisecond)
	}
}
