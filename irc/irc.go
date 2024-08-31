package irc

import (
	"crypto/tls"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/Elfshot/go-xdcc/config"
	"github.com/Elfshot/go-xdcc/util"

	irc "github.com/fluffle/goirc/client"
	log "github.com/sirupsen/logrus"
)

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
	case <-time.After(time.Duration(60*math.Pow(2.17, float64(retries))) * time.Second): // Rough exponential backoff
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

func createIrcClient() (*irc.Conn, chan bool) {
	mainConf := config.GetConfig()
	serverName := mainConf.IRC.Server
	cfg := irc.NewConfig(util.RandStr(12))
	cfg.SSL = true
	cfg.SSLConfig = &tls.Config{ServerName: serverName}
	cfg.Server = serverName + ":" + strconv.Itoa(mainConf.IRC.ServerPort)

	if len(mainConf.IRC.NickName) == 0 {
		cfg.NewNick = func(n string) string { return util.RandStr(16) }
	} else {
		cfg.NewNick = func(n string) string { return mainConf.IRC.NickName }
	}

	cfg.Me.Name = util.RandStr(12)
	cfg.Me.Ident = util.RandStr(8)
	c := irc.Client(cfg)

	log.Infof("Using | Nick %s | Name %s | Ident %s", c.Me().Nick, c.Me().Name, c.Me().Ident)

	quit := make(chan bool)

	return c, quit
}

func registerHandlers(c *irc.Conn, jobs chan *session, ready chan bool, quit chan bool) {
	c.HandleFunc(irc.CONNECTED,
		func(conn *irc.Conn, line *irc.Line) {
			log.Debug("irc.CONNECTED: " + line.Raw)
			conn.Join(config.GetConfig().IRC.ChannelName)
		})

	c.HandleFunc(irc.ACTION,
		func(conn *irc.Conn, line *irc.Line) { log.Debug("irc.ACTION: " + line.Raw) })

	c.HandleFunc(irc.KICK,
		func(conn *irc.Conn, line *irc.Line) { log.Debug("irc.KICK: " + line.Raw) })

	c.HandleFunc(irc.QUIT,
		func(conn *irc.Conn, line *irc.Line) { log.Debug("irc.QUIT: " + line.Raw) })

	c.HandleFunc(irc.REGISTER,
		func(conn *irc.Conn, line *irc.Line) { log.Debug("irc.REGISTER: " + line.Raw) })

	c.HandleFunc(irc.PRIVMSG,
		func(conn *irc.Conn, line *irc.Line) { log.Debug("irc.PRIVMSG: " + line.Raw) })

	c.HandleFunc(irc.CTCPREPLY,
		func(conn *irc.Conn, line *irc.Line) { log.Debug("irc.CTCPREPLY: " + line.Raw) })

	c.HandleFunc(irc.NOTICE,
		func(conn *irc.Conn, line *irc.Line) { log.Debug("irc.NOTICE: " + line.Raw) })

	c.HandleFunc(irc.AUTHENTICATE,
		func(conn *irc.Conn, line *irc.Line) { log.Debug("irc.AUTHENTICATE: " + line.Raw) })

	c.HandleFunc(irc.INVITE,
		func(conn *irc.Conn, line *irc.Line) { log.Debug("irc.INVITE: " + line.Raw) })

	c.HandleFunc(irc.PING,
		func(conn *irc.Conn, line *irc.Line) { log.Debug("irc.PING: " + line.Raw) })

	c.HandleFunc(irc.PONG,
		func(conn *irc.Conn, line *irc.Line) { log.Debug("irc.PONG: " + line.Raw) })

	c.HandleFunc(irc.PASS,
		func(conn *irc.Conn, line *irc.Line) { log.Debug("irc.PASS: " + line.Raw) })

	c.HandleFunc(irc.TOPIC,
		func(conn *irc.Conn, line *irc.Line) { log.Debug("irc.TOPIC: " + line.Raw) })

	c.HandleFunc(irc.USER,
		func(conn *irc.Conn, line *irc.Line) { log.Debug("irc.USER: " + line.Raw) })

	c.HandleFunc(irc.WHO,
		func(conn *irc.Conn, line *irc.Line) { log.Debug("irc.WHO: " + line.Raw) })

	c.HandleFunc(irc.WHOIS,
		func(conn *irc.Conn, line *irc.Line) { log.Debug("irc.WHOIS: " + line.Raw) })

	c.HandleFunc(irc.JOIN,
		func(conn *irc.Conn, line *irc.Line) {
			log.Debug("irc.JOIN: " + line.Raw)
			if line.Nick == c.Me().Nick {
				log.Info("Connected to Channel")
				ready <- true
			}
		})

	c.HandleFunc(irc.DISCONNECTED,
		func(conn *irc.Conn, line *irc.Line) {
			log.Debug("irc.DISCONNECTED: " + line.Raw)
			if line.Nick == c.Me().Nick {
				log.Info("Disconnected from Channel")
				conn.Close()
			}
		})

	c.HandleFunc(irc.ERROR,
		func(conn *irc.Conn, line *irc.Line) {
			log.Error("Error in IRC Client: " + line.Text())
			//// conn.Close()
		})

	c.HandleFunc(irc.CTCP,
		func(c *irc.Conn, l *irc.Line) {
			log.Debug("irc.CTCP: " + l.Raw)
			text := l.Text()
			textLower := strings.ToLower(text)
			arg0 := strings.SplitN(textLower, " ", 2)[0]
			// TODO "ACCEPT \"[HORRIBLESUBS] DR. STONE - 21 [1080P].MKV\" 41335 786432000"
			// TODO Should followup after this send handle to begin the transfer
			if strings.EqualFold(arg0, "send") {
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
