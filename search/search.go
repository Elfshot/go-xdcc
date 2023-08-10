package search

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/Elfshot/go-xdcc/config"
	log "github.com/sirupsen/logrus"
)

type PackScheme struct {
	BotId         int    `json:"botId"`
	EpisodeNumber int    `json:"episodeNumber"`
	Id            int    `json:"number"`       // Pack ID
	LastModified  string `json:"lastModified"` // yyyy-MM-dd HH:mm:ss
	Name          string `json:"name"`
	Size          string `json:"size"` // 1.2GB or 1.2MB or 1.2KB
	SizeBytes     int    `json:"sizekbits"`

	// *Not always given
	Crc32   string `json:"-"`
	Version int    `json:"-"`
	//// Number        int    `json:"number"`
}

type BotScheme struct {
	BatchEnable   int          `json:"batchEnable"` // 0 or 1
	Id            int          `json:"id"`
	LastProcessed string       `json:"lastProcessed"` // yyyy-MM-dd HH:mm:ss
	Name          string       `json:"name"`
	Owner         string       `json:"owner"`
	PackList      []PackScheme `json:"packList"`
	PackSize      int          `json:"packSize"`
	//// LastSeen      string       `json:"lastSeen"`      // yyyy-MM-dd HH:mm:ss
}

type niblApiRes[T any] struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	Content T      `json:"content"`
}

type jsonContent[T any] struct {
	Content T `json:"content"`
}

var preferedBots []BotScheme

// Group 1: group; Group 2: show; Group 3: episode; Group 5: version (optional)
// TODO: add version check and add it to the pack struct (allow S01E01v2 [checksum?].mkv)
var nameRegexp = regexp.MustCompile(`^(\[[a-zA-z]+\]).*?\s(.*?)\s-\s(\d+)(v\d+)?.*$`)

// Group 2: quality
var qualRegexp = regexp.MustCompile(`(\[|\()(1080|720|540|480|360)p(\]|\))`)

// Group 1: CRC
// TODO: add CRC check and add it to the pack struct
var crcRegexp = regexp.MustCompile(`\[([a-zA-Z0-9]{8})\]`)

func getPreferedBots() []BotScheme {
	if preferedBots != nil {
		return preferedBots
	}
	preferedBots = []BotScheme{}
	botNames := config.GetConfig().PreferedBots
	bots, err := loadBots()
	if err != nil {
		log.Error("Cannot load bots:\n" + err.Error())
	}

	for _, botName := range botNames {
		for _, bot := range bots {
			if bot.Name == botName {
				preferedBots = append(preferedBots, bot)
			}
		}
	}
	return preferedBots
}

func GetBotName(id int) (string, error) {
	bots := getPreferedBots()
	for _, bot := range bots {
		if bot.Id == id {
			return bot.Name, nil
		}
	}
	return "", errors.New("Bot not found for ID: " + fmt.Sprint(id))
}

func formatString(s string) string {
	s = strings.ReplaceAll(s, "_", " ")
	s = strings.Trim(s, " _")
	s = strings.ToLower(s)
	return s
}

func GetSeriesPacks(series string) ([]PackScheme, error) {
	// 1080, 720, 540, 480, 360 res options
	search, err := getPacksSearch(series, 0, "botId")

	if err != nil {
		return nil, err
	}

	packs := []PackScheme{}

	series = formatString(series)
	for _, pack := range search {
		quals := qualRegexp.FindStringSubmatch(pack.Name)
		if len(quals) <= 0 {
			continue
		}

		if strings.EqualFold(quals[2]+"p", config.GetConfig().PreferedFormat) { // prefered quality
			names := nameRegexp.FindStringSubmatch(pack.Name)
			if len(names) <= 0 {
				continue
			}

			name := formatString(names[2])

			if len(names) >= 5 {
				if ver := names[4]; ver != "" {
					ver, err := strconv.Atoi(strings.Split(ver, "v")[1])
					if err == nil {
						pack.Version = ver
					}
				} else {
					pack.Version = 1
				}
			}

			crc32 := ""
			if crcR := crcRegexp.FindStringSubmatch(pack.Name); len(crcR) > 0 {
				crc32 = crcR[1]
			}

			if len(crc32) > 0 {
				pack.Crc32 = crc32
			}

			if (pack.EpisodeNumber <= 0) && (len(names[3]) > 0) {
				epInt, err := strconv.Atoi(names[3])

				if err != nil {
					continue
				}

				pack.EpisodeNumber = epInt
			}

			inArrI, arrPack := findInPacks(packs, pack.EpisodeNumber)

			// If pack already in array, check if we have a newer version or have a more prefered bot
			// Newer version is prioritized over prefered bot
			if inArrI >= 0 {
				if arrPack.Version < pack.Version {
					packs[inArrI] = pack
					continue
				} else if arrPack.Version > pack.Version {
					continue
				}

				pefInArr := getBotIdPrefPos(arrPack.BotId)
				prefNew := getBotIdPrefPos(pack.BotId)

				if (pefInArr < 0) || ((prefNew > 0) && (prefNew < pefInArr)) {
					packs[inArrI] = pack
					continue
				}
			} else if strings.EqualFold(name, series) {
				packs = append(packs, pack)
				continue
			}

		}
	}

	return packs, nil
}

func getBotIdPrefPos(botId int) int {
	bots := getPreferedBots()
	for i, bot := range bots {
		if bot.Id == botId {
			return i
		}
	}
	return -1
}

func findInPacks(packs []PackScheme, epNum int) (int, *PackScheme) {
	for i, p := range packs {
		if p.EpisodeNumber == epNum {
			return i, &p
		}
	}
	return -1, nil
}

// episode = 0 for all; sort is key of PackScheme (json version)
func getPacksSearch(search string, episode int, sort string) ([]PackScheme, error) {
	search = url.QueryEscape(search)
	return getNiblPacks(fmt.Sprintf("/search/page?query=%s&episodeNumber=%d&page=0&size=10000000&sort=%s&direction=ASC", search, episode, sort))
}

func GetPacksLastest(history int) ([]PackScheme, error) {
	return getNiblPacks(fmt.Sprintf("/latest?size=%d", history))
}

func getBots() ([]BotScheme, error) {
	var bots niblApiRes[[]BotScheme]
	_, err := getNibl("/bots", &bots)
	if err != nil {
		return nil, err
	}

	return bots.Content, nil
}

func loadBots() ([]BotScheme, error) {
	// load from cwd/data/bots.json
	// if not exist, get from nibl and save to cwd/data/bots.json

	//ERROR if dir not exist
	cwd, err := os.Getwd()
	if err != nil {
		log.Fatal("Cannot get current working directory:\n" + err.Error())
	}

	sep := string(os.PathSeparator)
	botPath := fmt.Sprintf("%s%sdata%sbots.json", cwd, sep, sep)

	var data jsonContent[[]BotScheme]

	_, err = loadJson(botPath, &data)
	if err != nil {
		log.Info("Cannot load bots from " + botPath + ":\n" + err.Error())
		log.Info("Getting bots from nibl...")
		bots, err := getBots()
		if err != nil {
			return nil, err
		}

		plan, err := json.MarshalIndent(jsonContent[[]BotScheme]{bots}, "", "  ")
		if err != nil {
			return nil, err
		}

		// Ensure bothPath exists
		os.MkdirAll(cwd+sep+"data", 0777)

		err = os.WriteFile(botPath, plan, 0777)
		if err != nil {
			log.Error("Cannot save bots to " + botPath + ":\n" + err.Error())
		}
		return bots, nil
	}
	return data.Content, nil
}

func loadJson(file string, strucPass any) (any, error) {
	plan, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(plan, strucPass)
	if err != nil {
		return nil, err
	}

	return strucPass, nil
}

func getNiblPacks(endpoint string) ([]PackScheme, error) {
	var packs niblApiRes[[]PackScheme]
	_, err := getNibl(endpoint, &packs)
	if err != nil {
		return nil, err
	}
	return packs.Content, nil
}

func getNibl(endpoint string, structPass any) (any, error) {
	_, err := get("https://api.nibl.co.uk/nibl"+endpoint, structPass)
	if err != nil {
		return nil, err
	}

	return structPass, nil
}
