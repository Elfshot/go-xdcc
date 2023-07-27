package util

import (
	"errors"
	"math/rand"
	"net"
	"os"
	"runtime"

	"github.com/Elfshot/go-xdcc/config"
	log "github.com/sirupsen/logrus"
)

const randomChars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

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
		conn.Read(make([]byte, 1024))
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
	RunGC()
	return len(data), nil
}

func RunGC() {
	runtime.GC()
}
