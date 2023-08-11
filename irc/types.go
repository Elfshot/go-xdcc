package irc

import (
	"net"
	"time"

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

type events struct {
	EventType    int
	PackData     *Pack
	TransferData *transfer
}

type queue struct {
	pack    *Pack
	monitor *progress.Monitor
	attempt uint8
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
