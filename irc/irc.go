package irc

import (
	"bufio"
	"crypto/tls"
	"errors"
	"fmt"
	"hash/crc32"
	"math/rand"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/Elfshot/go-xdcc/config"
	"github.com/Elfshot/go-xdcc/progress"

	irc "github.com/fluffle/goirc/client"
	log "github.com/sirupsen/logrus"
)

const randomChars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
const cRed = "\033[31m"
const cBlue = "\033[34m"

// 50MB buffer | Low values create too many write calls | High values create too much memory usage
const bufferSize = 1024 * 1024 * 50

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

func randName(n uint) string {
	newString := make([]byte, n)
	for i := range newString {
		newString[i] = randomChars[rand.Intn(len(randomChars))]
	}
	return string(newString)
}

func uint32ToIP(n int) net.IP {
	a := byte((n >> 24) & 255)
	b := byte((n >> 16) & 255)
	c := byte((n >> 8) & 255)
	d := byte(n & 255)
	return net.IPv4(a, b, c, d)
}

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

type session struct {
	packData     *Pack
	transferData *transfer
}

type events struct {
	EventType    int
	PackData     *Pack
	TransferData *transfer
}

// Events
const (
	TRANSFER_START       = iota // 0
	TRANSFER_PAUSE              // 1
	TRANSFER_RESUME             // 2
	TRANSFER_CANCEL             // 3
	TRANSFER_FINISH             // 4
	TRANSFER_PROGRESS           // 5
	TRANSFER_ERROR              // 6
	TRANFER_PRECOMPLETED        // 7
	TRANFER_FILE_CLOSED         // 8
)

// States
const (
	TRANSFER_STATUS_IDLE     = iota // 0
	TRANSFER_STATUS_STARTED         // 1
	TRANSFER_STATUS_PAUSED          // 2
	TRANSFER_STATUS_CANCELED        // 3
	TRANSFER_STATUS_FINISHED        // 4
	TRANSFER_STATUS_ERROR           // 5
)

func (session *session) sendEvent(eventType int) {
	transfer := session.transferData

	switch eventType {
	case TRANSFER_START:
		transfer.status = TRANSFER_STATUS_STARTED
	case TRANSFER_PROGRESS:
		transfer.status = TRANSFER_STATUS_STARTED
	case TRANSFER_FINISH:
		transfer.status = TRANSFER_STATUS_FINISHED
	case TRANFER_PRECOMPLETED:
		transfer.status = TRANSFER_STATUS_FINISHED
	case TRANSFER_ERROR:
		transfer.status = TRANSFER_STATUS_ERROR
	case TRANSFER_CANCEL:
		transfer.status = TRANSFER_STATUS_CANCELED
	case TRANSFER_PAUSE:
		transfer.status = TRANSFER_STATUS_PAUSED
	case TRANSFER_RESUME:
		transfer.status = TRANSFER_STATUS_STARTED
	}
	transfer.events <- &events{EventType: eventType, TransferData: transfer, PackData: session.packData}
}

func (session *session) startTransfer(irc *irc.Conn) {
	transferData := session.transferData
	packData := session.packData

	newFileDir := packData.GetFileDir() + ".part"

	oldSize, err := getFileSize(newFileDir)
	if err != nil {
		session.sendEvent(TRANSFER_ERROR)
		log.Error(err)
		return
	}
	transferData.transferedBytes, transferData.startBytes = oldSize, oldSize

	if oldSize >= packData.Size {
		voidTcpConn(transferData.targetIp, transferData.targetPort)
		session.sendEvent(TRANFER_PRECOMPLETED)
		log.Debug("Already downloaded")
		return
	}

	if oldSize < packData.Size && oldSize != 0 {
		transferData.isResume = true
		// ddc resume
		irc.Ctcp(packData.BotNick, "DCC RESUME "+" \""+packData.FileName+"\" "+" "+strconv.Itoa(transferData.targetPort)+" "+strconv.Itoa(oldSize))
		session.sendEvent(TRANSFER_RESUME)
		// TODO fix instead of sleep wait for the ACCEPT message
		// TODO ACCEPT "[SUBSPLEASE] KOORI ZOKUSEI DANSHI TO COOL NA DOURYOU JOSHI - 05 (1080P)" 12354 473794808
		// TODO Method, "name", port, new start position
		time.Sleep(5 * time.Second)
	} else {
		transferData.isResume = false
	}

	file, err := os.OpenFile(newFileDir, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0777)
	if err != nil {
		session.sendEvent(TRANSFER_ERROR)
		log.Println(err)
		return
	}

	writer := bufio.NewWriterSize(file, bufferSize)

	conn, err := openTcpConn(transferData.targetIp, transferData.targetPort)

	if err != nil {
		session.sendEvent(TRANSFER_ERROR)
		log.Println(err)
		return
	}

	// Empty byte array takes too much memory for large files
	buffer := make([]byte, bufferSize)
	transferData.unixStart = time.Now().Unix()
	session.sendEvent(TRANSFER_START)

	inQuit := make(chan bool, 1)
	inProgress := make(chan bool, 1)
	outQuit := make(chan bool, 1)

	closeConnDur := time.Duration(config.GetConfig().IRC.MaxTcpIdleTime) * time.Second

	go func() {
		for {
			timer := time.NewTimer(closeConnDur)
			after := timer.C
			select {
			case <-inQuit:
				return

			case <-inProgress:
				if !timer.Stop() {
					<-after
				}
				continue

			case <-after:
				outQuit <- true
				return
			}
		}
	}()

	for {
		select {
		case <-outQuit:
			session.sendEvent(TRANSFER_ERROR)
			endTransfer(writer, conn, file, transferData)
			return
		default:
			conn.SetReadDeadline(time.Now().Add(closeConnDur))
			bytesRead, err := conn.Read(buffer)

			if err != nil {
				session.sendEvent(TRANSFER_ERROR)
				log.Println(err)
				endTransfer(writer, conn, file, transferData)
				inQuit <- true
				return
			}

			_, err = writer.Write(buffer[:bytesRead])

			if err != nil {
				session.sendEvent(TRANSFER_ERROR)
				log.Println(err)
				endTransfer(writer, conn, file, transferData)
				inQuit <- true
				return
			}

			buffer = make([]byte, bufferSize)

			transferData.transferedBytes += bytesRead

			session.sendEvent(TRANSFER_PROGRESS)
			inProgress <- true

			if session.stopTranferConditions() {
				session.sendEvent(TRANSFER_FINISH)
				endTransfer(writer, conn, file, transferData)

				inQuit <- true
				log.Debugf("Done Transfer for %s", packData.FileName)
				os.Rename(newFileDir, packData.GetFileDir())
				log.Debugf("Renamed file from %s to %s", newFileDir, packData.GetFileDir())
				session.sendEvent(TRANFER_FILE_CLOSED)
				return
			}
		}
	}
}

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

func (session *session) stopTranferConditions() bool {
	transfer := session.transferData
	if transfer.status == TRANSFER_STATUS_CANCELED ||
		transfer.status == TRANSFER_STATUS_PAUSED ||
		transfer.status == TRANSFER_STATUS_ERROR ||
		transfer.status == TRANSFER_STATUS_FINISHED ||
		transfer.transferedBytes >= session.packData.Size {
		return true
	} else {
		return false
	}
}

var boundIp net.IP

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

var ircClient *irc.Conn = nil

func getIrc(jobs chan *session, retries int) (quit chan bool, client *irc.Conn) {
	if ircClient != nil && ircClient.Connected() {
		return quit, ircClient
	}

	ready := make(chan bool, 1)

	if ircClient != nil {
		ircClient.Close()
	}
	ircClient, quit = createIrcClient()

	registerHandlers(ircClient, jobs, ready, quit)

	finishConnect := make(chan bool, 1)

	go func() {
		if err := ircClient.Connect(); err != nil {
			log.Fatalf("Connection error: %s\n", err.Error())
			return
		}
		finishConnect <- true
	}()

	select {
	case <-finishConnect:
		break
	case <-time.After(30 * time.Second):
		if retries >= 3 {
			log.Fatalf("Connection error: %s\n", "maximum number of retries reached for IRC network connection")
		}
		if ircClient != nil {
			ircClient.Close()
			ircClient = nil
		}
		close(quit)
		return getIrc(jobs, retries+1)
	}

	select {
	case <-ready:
		return quit, ircClient
	case <-time.After(10 * time.Second):
		if retries >= 3 {
			log.Fatalf("Connection error: %s\n", "maximum number of retries reached for IRC channel connection")
		}
		if ircClient != nil {
			ircClient.Close()
			ircClient = nil
		}
		close(quit)
		return getIrc(jobs, retries+1)
	}
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

func parseCtcpString(ctcpRes *irc.Line) (*session, error) {

	transferStub := &transfer{
		unixStart:       0,
		unixEnd:         0,
		transferedBytes: 0,
		startBytes:      0,
		targetPort:      0,
		targetIp:        nil,

		events: make(chan *events, 1),
	}

	fields := strings.Fields(ctcpRes.Text())[1:]
	targetIp, err := strconv.Atoi(fields[len(fields)-3])
	if err != nil {
		log.Error(err)
		return nil, err
	}
	transferStub.targetIp = uint32ToIP(targetIp)

	targetPort, err := strconv.Atoi(fields[len(fields)-2])
	if err != nil {
		log.Error(err)
		return nil, err
	}
	transferStub.targetPort = targetPort

	packName := strings.ReplaceAll(strings.Join(fields[:len(fields)-4], " "), "\"", "")
	pack := findPack(packName)
	if pack == nil {
		log.Error("Pack not found")
		return nil, errors.New("Pack not found")
	}

	size, err := strconv.Atoi(fields[len(fields)-1])
	if err != nil {
		log.Error(err)
		return nil, err
	}
	pack.Size = size

	sessionStub := &session{
		packData:     pack,
		transferData: transferStub,
	}

	return sessionStub, nil
}

func createIrcClient() (*irc.Conn, chan bool) {
	mainConf := config.GetConfig()
	serverName := mainConf.IRC.Server
	cfg := irc.NewConfig(randName(12))
	cfg.SSL = true
	cfg.SSLConfig = &tls.Config{ServerName: serverName}
	cfg.Server = serverName + ":" + strconv.Itoa(mainConf.IRC.ServerPort)

	if len(mainConf.IRC.NickName) == 0 {
		cfg.NewNick = func(n string) string { return randName(16) }
	} else {
		cfg.NewNick = func(n string) string { return mainConf.IRC.NickName }
	}

	cfg.Me.Name = randName(8)
	cfg.Me.Ident = "go-xdcc"
	c := irc.Client(cfg)

	log.Infof("Using | Nick %s | Name %s | Ident %s", c.Me().Nick, c.Me().Name, c.Me().Ident)

	quit := make(chan bool)

	return c, quit
}

func registerHandlers(c *irc.Conn, jobs chan *session, ready chan bool, quit chan bool) {
	c.HandleFunc(irc.CONNECTED,
		func(conn *irc.Conn, line *irc.Line) {
			log.Debug(line.Text())
			conn.Join(config.GetConfig().IRC.ChannelName)
		})

	c.HandleFunc(irc.CTCP,
		func(conn *irc.Conn, line *irc.Line) { log.Debug(line.Text()) })

	c.HandleFunc(irc.ACTION,
		func(conn *irc.Conn, line *irc.Line) { log.Debug(line.Text()) })

	c.HandleFunc(irc.KICK,
		func(conn *irc.Conn, line *irc.Line) { log.Debug(line.Text()) })

	c.HandleFunc(irc.QUIT,
		func(conn *irc.Conn, line *irc.Line) { log.Debug(line.Text()) })

	c.HandleFunc(irc.REGISTER,
		func(conn *irc.Conn, line *irc.Line) { log.Debug(line.Text()) })

	c.HandleFunc(irc.PRIVMSG,
		func(conn *irc.Conn, line *irc.Line) { log.Debug(line.Text()) })

	c.HandleFunc(irc.CTCPREPLY,
		func(conn *irc.Conn, line *irc.Line) { log.Debug(line.Text()) })

	c.HandleFunc(irc.NOTICE,
		func(conn *irc.Conn, line *irc.Line) { log.Debug(line.Text()) })

	c.HandleFunc(irc.JOIN,
		func(conn *irc.Conn, line *irc.Line) {
			if line.Nick == c.Me().Nick {
				log.Info("Connected to Channel")
				ready <- true
			}
		})

	c.HandleFunc(irc.DISCONNECTED,
		func(conn *irc.Conn, line *irc.Line) {
			conn.Close()
		})

	c.HandleFunc(irc.ERROR,
		func(conn *irc.Conn, line *irc.Line) {
			log.Error("Error in IRC Client: " + line.Text())
			//// conn.Close()
		})

	c.HandleFunc(irc.CTCP,
		func(c *irc.Conn, l *irc.Line) {
			text := l.Text()
			textLower := strings.ToLower(text)
			arg0 := strings.SplitN(textLower, " ", 2)[0]
			// TODO "ACCEPT \"[HORRIBLESUBS] DR. STONE - 21 [1080P].MKV\" 41335 786432000"
			// TODO Should followup after this send handle to begin the transfer
			if strings.EqualFold(arg0, "send") {
				log.Info("CTCP: " + text)

				session, err := parseCtcpString(l)
				if err != nil {
					log.Error(err)
					return
				}
				jobs <- session
				go func() {
					session.startTransfer(c)
					quit <- true
				}()
			} else if strings.EqualFold(arg0, "accept") {
				return
			} else {
				return
			}
		})
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

type queue struct {
	pack    *Pack
	monitor *progress.Monitor
}

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

func QueueLoop() {
	config := config.GetConfig()
	max_dls := config.MaxDownloads
	closeConnMins := time.Duration(config.IRC.CloseConnectionMins)

	counter := counterT{count: 0, limit: max_dls}
	jobs := make(chan *session)

	progress := make(chan bool)

	go func() {
		for {
			timer := time.NewTimer(closeConnMins * time.Minute)
			after := timer.C
			select {
			case <-progress:
				if !timer.Stop() {
					<-after
				}
				continue
			case <-after:
				if ircClient != nil && ircClient.Connected() {
					log.Debugf("No progress in %d minutes, closing irc connection", closeConnMins)
					ircClient.Close()
					ircClient = nil
				}
				continue
			}
		}
	}()

	for v := range packQueue {
		counter.inc()

		quit := getPack(v.pack.BotNick, v.pack.PackNumber, jobs)
		monitor := v.monitor
		job := <-jobs

		pack := job.packData
		tracker := monitor.Add(job.packData.FileName, job.packData.Size)
		tracker.Total = int64(job.packData.Size - job.transferData.startBytes)

		go func() {
			for {
				select {
				case <-quit:
					counter.dec()
					return
				case event := <-job.transferData.events:
					transfer := event.TransferData
					status := event.EventType

					if status == TRANSFER_FINISH || status == TRANSFER_CANCEL {
						tracker.SetValue(int64(transfer.transferedBytes))
						removeFinishedPack(pack)
						counter.dec()
						if status == TRANSFER_CANCEL {
							return
						}
					} else if status == TRANSFER_PROGRESS {
						tracker.SetValue(int64(transfer.transferedBytes - transfer.startBytes))
						progress <- true

					} else if status == TRANSFER_RESUME {
						tracker.UpdateMessage(cRed + "[Resuming] " + cBlue + tracker.Message)
						tracker.UpdateTotal(int64(pack.Size - transfer.startBytes))
						tracker.SetValue(int64(transfer.transferedBytes - transfer.startBytes))

					} else if status == TRANFER_PRECOMPLETED {
						tracker.UpdateMessage(cRed + "[Already Complete!] " + cBlue + tracker.Message)
						tracker.UpdateTotal(0)
						tracker.MarkAsDone()
						counter.dec()
						return

					} else if status == TRANSFER_ERROR {
						tracker.SetValue(int64(transfer.transferedBytes))
						tracker.MarkAsErrored()
						counter.dec()

						// Keep track of retries + add retry flag/event
						removeFinishedPack(pack)
						QueuePack(pack, monitor)

						return
					} else if status == TRANFER_FILE_CLOSED && pack.Crc32 != "" {
						if !((config.CrcCheck == "resume" && transfer.isResume) ||
							config.CrcCheck == "always") {
							return
						}

						bytes, err := os.ReadFile(pack.GetFileDir())

						if err != nil {
							log.Errorf("Cannot read file to complete CRC check for %s, %s", pack.FileName, err)
							return
						}
						crc32 := fmt.Sprintf("%08X", crc32.ChecksumIEEE(bytes))

						if pack.Crc32 != crc32 {
							log.Errorf("CRC32 Checksum mismatch for %s, expected %s got %s", pack.FileName, pack.Crc32, crc32)
							err := os.Remove(pack.GetFileDir())

							if err != nil {
								log.Errorf("Cannot remove invalid file %s, %s", pack.GetFileDir(), err)
								return
							}

							go func() {
								time.Sleep(180 * time.Second)
								QueuePack(pack, monitor)
							}()
							return
						}
						log.Debugf("CRC32 Checksum match for %s, expected %s got %s", pack.FileName, pack.Crc32, crc32)
						return
					}
				}
			}

		}()

		waitIrcReady()
		counter.wait()

		time.Sleep(10 * time.Second)

	}
}

func waitIrcReady() {
	if ircClient != nil && ircClient.Connected() {
		return
	}
	ready := make(chan bool)

	go func() {
		for {
			if ircClient != nil && ircClient.Connected() {
				ready <- true
				return
			}
			time.Sleep(500 * time.Millisecond)
		}
	}()
	fails := 0
	maxWaits := config.GetConfig().IRC.MaxWaitIrcCycles
	for {
		select {
		case <-ready:
			return
		case <-time.After(10 * time.Second):
			log.Error("IRC Client not ready after 10 seconds")
			if fails += 1; fails > maxWaits {
				log.Fatalf("IRC Client not ready after %d waits", maxWaits)
			}
		}
	}
}
