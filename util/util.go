package util

import (
	"fmt"
	"hash/crc32"
	"math/rand"
	"net"
	"os"
	"runtime"

	"github.com/Elfshot/go-xdcc/config"
	log "github.com/sirupsen/logrus"
)

const randomChars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

var bufferSize = config.GetConfig().BufferSizeMB * 1024 * 1024
var boundIp net.IP

func CheckDefaultI(x ...int) bool {
	for _, v := range x {
		if v != 0 {
			return false
		}
	}
	return true
}

func CheckDefaultS(x ...string) bool {
	for _, v := range x {
		if v != "" {
			return false
		}
	}
	return true
}

func Uint32ToIP(n int) net.IP {
	a := byte((n >> 24) & 255)
	b := byte((n >> 16) & 255)
	c := byte((n >> 8) & 255)
	d := byte(n & 255)
	return net.IPv4(a, b, c, d)
}

func OpenTcpConn(ip net.IP, port int) (net.Conn, error) {
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

func VoidTcpConn(ip net.IP, port int) {
	conn, err := OpenTcpConn(ip, port)

	if err != nil {
		log.Error(err)
		conn.Close()
		return
	}
	for i := 0; i < 25; i++ {
		conn.Read(make([]byte, 1024*512)) // 512KB
	}
	conn.Close()
}

func RandStr(n uint) string {
	newString := make([]byte, n)
	for i := range newString {
		newString[i] = randomChars[rand.Intn(len(randomChars))]
	}
	return string(newString)
}

func GetFileSize(f string) (int, error) {
	file, _ := os.OpenFile(f, os.O_APPEND|os.O_CREATE, 0777)
	defer file.Close()

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

	size := 0
	var c error = nil
	data := make([]byte, bufferSize)

	for c == nil {
		n := 0
		n, c = file.Read(data)
		size += n
	}

	return size, nil
}

func RunGC() {
	log.Debug("Running GC")
	runtime.GC()
}

func GetCrc32(fileName string) (string, error) {
	file, err := os.Open(fileName)
	if err != nil {
		return "", err
	}
	defer file.Close()

	buffer := make([]byte, bufferSize)
	hash := crc32.NewIEEE()

	var gg error = nil

	for gg == nil {
		var n int
		n, gg = file.Read(buffer)

		// truncate the buffer to the actual data read if necessary
		if n < bufferSize {
			buffer = buffer[:n]
		}

		hash.Write(buffer)
	}

	// get the checksum
	checksum := fmt.Sprintf("%8X", hash.Sum32())

	RunGC()
	return checksum, nil
}
