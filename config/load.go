package config

import (
	"os"

	log "github.com/sirupsen/logrus"

	"gopkg.in/yaml.v3"
)

var gConfig *Config

// Create a config struct to hold the config data
type Config struct {
	PreferedBots     []int     `yaml:"preferedBots"`
	MaxDownloads     int       `yaml:"maxDownloads"`
	PreferedFormat   string    `yaml:"preferedFormat"`
	DownloadDir      string    `yaml:"downloadDir"`
	BoundIp          string    `yaml:"boundIp"`
	DownloadInterval int       `yaml:"downloadInterval"`
	IRC              ircConfig `yaml:"irc"`
	Trackers         []Tracker
}

type ircConfig struct {
	Server              string `yaml:"server"`
	ServerPort          int    `yaml:"serverPort"`
	ChannelName         string `yaml:"channelName"`
	NickName            string `yaml:"nick"`
	CloseConnectionMins int    `yaml:"closeConnectionMins"`
	MaxWaitIrcCycles    int    `yaml:"maxWaitIrcCycles"`
}

type Tracker struct {
	SearchName   string `yaml:"searchName"`
	FileName     string `yaml:"fileName"`
	Season       int    `yaml:"season"`
	EpisodeRange [2]int `yaml:"episodeRange"`
}

// Load yaml config file "./config.yaml" into Config struct and return it

func (config *Config) LoadBaseConfig() {

	// Create config file if it doesn't exist then exit
	if _, err := os.Stat("config/config.yaml"); os.IsNotExist(err) {
		os.Mkdir("config", 0766)
		os.Create("config/config.yaml")
		os.Chmod("config/config.yaml", 0766)
		os.Chown("config/config.yaml", os.Getuid(), os.Getgid())
		log.Fatal("Config file created. Please edit config/config.yaml and restart the program" +
			"Consider using the example config file as a template and adding trackers.")
	}

	// Read config file into byte array
	byteValue, err := os.ReadFile("config/config.yaml")
	if err != nil {
		log.Fatalln(err)
	}

	// Unmarshal config file into Config struct
	err = yaml.Unmarshal(byteValue, config)
	if err != nil {
		log.Fatalln(err)
	}

	// Check critial config values
	// Stub

	// Add trailing slash to download dir if it doesn't exist
	if config.DownloadDir[len(config.DownloadDir)-1:] != "/" {
		config.DownloadDir += "/"
	}
	// Create download dir if it doesn't exist
	if _, err := os.Stat(config.DownloadDir); os.IsNotExist(err) {
		os.Mkdir(config.DownloadDir, 0777)
	}
}

func (cfg *Config) LoadTrackers() {
	if len(cfg.Trackers) > 0 {
		cfg.Trackers = cfg.Trackers[:0]
	}

	files, err := os.ReadDir("config/trackers")
	if err != nil {
		log.Error("Failed to read trackers directory: ", err.Error())
		return
	}
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		if file.Name()[len(file.Name())-5:] != ".yaml" {
			continue
		}
		byteValue, err := os.ReadFile("config/trackers/" + file.Name())
		if err != nil {
			log.Errorf("Failed to read tracker file: %s\n%s", file.Name(), err.Error())
			continue
		}
		tracker := Tracker{}
		err = yaml.Unmarshal(byteValue, &tracker)
		if err != nil {
			log.Errorf("Failed to unmarshal tracker file: %s\n%s", file.Name(), err.Error())
			continue
		}
		cfg.Trackers = append(cfg.Trackers, tracker)
	}
}

func loadConfig() *Config {
	config := &Config{}

	config.LoadBaseConfig()
	config.LoadTrackers()

	return config
}

func GetConfig() *Config {
	if gConfig != nil {
		return gConfig
	}

	gConfig = loadConfig()
	return gConfig
}
