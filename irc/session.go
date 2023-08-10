package irc

import (
	"bufio"
	"errors"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/Elfshot/go-xdcc/config"
	"github.com/Elfshot/go-xdcc/util"
	irc "github.com/fluffle/goirc/client"
	log "github.com/sirupsen/logrus"
)

var bufferSize = config.GetConfig().BufferSizeMB * 1024 * 1024

type session struct {
	packData     *Pack
	transferData *transfer
}

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

	oldSize, err := util.GetFileSize(newFileDir)
	if err != nil {
		session.sendEvent(TRANSFER_ERROR)
		log.Error(err)
		return
	}

	// TODO CRC if = rather than >
	if oldSize >= packData.Size {
		log.Errorf("File %s already fully+ downloaded as part but likely corrupt. Redownloading...", packData.FileName)

		err := os.Remove(newFileDir)
		if err != nil {
			session.sendEvent(TRANSFER_ERROR)
			log.Errorf("Cannot remove file %s, %s", packData.FileName, err)
			util.VoidTcpConn(transferData.targetIp, transferData.targetPort)
			return
		}

		oldSize = 0
	}

	transferData.transferedBytes, transferData.startBytes = oldSize, oldSize

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

	conn, err := util.OpenTcpConn(transferData.targetIp, transferData.targetPort)

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
	transferStub.targetIp = util.Uint32ToIP(targetIp)

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

func QueueLoop() {
	if loopStarted {
		return
	} else {
		loopStarted = true
	}

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

				if !loopStarted {
					return
				}

				continue
			}
		}
	}()

	for v := range packQueue {
		counter.inc()

		sizeFree, _, err := util.DiskUsage(config.DownloadDir)

		if err != nil {
			log.Error("Cannot get disk space: " + err.Error() + "\nSkipping pack: " + v.pack.FileName)
			continue
		}

		if int(sizeFree) < (v.pack.Size + int(float64(v.pack.Size)*float64(0.1))) {
			log.Errorf("Not enough space to download %s, ~%d Mb needed, %d Mb free", v.pack.FileName, v.pack.Size, sizeFree)
			continue
		}

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

						crc32, err := util.GetCrc32(pack.GetFileDir())

						if err != nil {
							log.Errorf("Cannot read file to complete CRC check for %s, %s", pack.FileName, err)
							return
						}

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

	loopStarted = false
}
