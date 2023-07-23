package irc

import (
	"bufio"
	"errors"
	"math/rand"
	"net"
	"os"
	"time"

	"github.com/Elfshot/go-xdcc/config"
	log "github.com/sirupsen/logrus"
)

func endTransfer(writer *bufio.Writer, conn net.Conn, file *os.File, transferData *transfer) {
	if writer.Buffered() > 0 {
		writer.Flush()
	}
	conn.Close()
	file.Close()
	transferData.unixEnd = time.Now().Unix()

	// Extra sleep to ensure that any sendEvent() calls have been processed
	time.Sleep(200 * time.Millisecond)
}

func uint32ToIP(n int) net.IP {
	a := byte((n >> 24) & 255)
	b := byte((n >> 16) & 255)
	c := byte((n >> 8) & 255)
	d := byte(n & 255)
	return net.IPv4(a, b, c, d)
}

func openTcpConn(ip net.IP, port int) (net.Conn, error) {
	switch {
	case boundIp == nil:
		boundIp = net.ParseIP(config.GetConfig().BoundIp)
		if boundIp == nil {
			boundIp = []byte{0, 0, 0, 0}
		}
	case boundIp[0] == 0:
		break
	default:
		break
	}

	conn, err := net.DialTCP("tcp", &net.TCPAddr{IP: boundIp, Port: 0}, &net.TCPAddr{IP: ip, Port: port})

	if err != nil {
		log.Error(err)
		return nil, err
	}
	return conn, nil
}

func voidTcpConn(ip net.IP, port int) {
	conn, err := openTcpConn(ip, port)

	if err != nil {
		log.Error(err)
		conn.Close()
		return
	}
	for i := 0; i < 25; i++ {
		conn.Read(make([]byte, 1024))
	}
	conn.Close()
}

func randStr(n uint) string {
	newString := make([]byte, n)
	for i := range newString {
		newString[i] = randomChars[rand.Intn(len(randomChars))]
	}
	return string(newString)
}

func getFileSize(f string) (int, error) {
	fileStub, _ := os.OpenFile(f, os.O_APPEND|os.O_CREATE, 0777)
	fileStub.Close()

	// ?Fast/Usually accurate
	/*
		g, err := os.Stat(f)

		if err != nil {
			log.Error(err)
			return 0, errors.New("error getting file size")
		}

		return int(g.Size()), nil
	*/

	// *Slow/Always accurate | High Memory
	/*
		data, err := os.ReadFile(f)
		if err != nil {
			log.Error(err)
			return 0, errors.New("error reading file")
		}
		return len(data), nil
	*/

	data, err := os.ReadFile(f)
	if err != nil {
		log.Error(err)
		return 0, errors.New("error reading file")
	}
	return len(data), nil
}
