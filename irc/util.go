package irc

import (
	"bufio"
	"net"
	"os"
	"time"

	"github.com/Elfshot/go-xdcc/util"
)

func endTransfer(writer *bufio.Writer, conn net.Conn, file *os.File, transferData *transfer) {
	if writer.Buffered() > 0 {
		writer.Flush()
	}
	conn.Close()
	file.Close()
	transferData.unixEnd = time.Now().Unix()
	util.RunGC()

	// Extra sleep to ensure that any sendEvent() calls have been processed
	time.Sleep(200 * time.Millisecond)
}
