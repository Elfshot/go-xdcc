package search

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"regexp"
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
	Sizekbits     int    `json:"sizekbits"`
	// Number        int    `json:"number"`

}

type BotScheme struct {
	BatchEnable   int          `json:"batchEnable"` // 0 or 1
	Id            int          `json:"id"`
	LastProcessed string       `json:"lastProcessed"` // yyyy-MM-dd HH:mm:ss
	Name          string       `json:"name"`
	Owner         string       `json:"owner"`
	PackList      []PackScheme `json:"packList"`
	PackSize      int          `json:"packSize"`
	// LastSeen      string       `json:"lastSeen"`      // yyyy-MM-dd HH:mm:ss
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

// Group 2: name; Group 3: episode; Group 4: version
var nameRegexp = regexp.MustCompile(`^(\[\w+\]){1,2}\s(.*?)\s-\s(\d+)?(v\d+)?(\s(\(\w+p\)|(\[\w+p\]))?){1,2}.*$`)
var qualRegexp = regexp.MustCompile(`(\[|\()(1080|720|540|480|360)p(\]|\))`)

// return in order
func getPreferedBots() []BotScheme {
	if preferedBots != nil {
		return preferedBots
	}
	preferedBots = []BotScheme{}
	botIds := []int{835, 836, 1010, 696}
	bots, err := loadBots()
	if err != nil {
		log.Error("Cannot load bots:\n" + err.Error())
	}

	for _, botId := range botIds {
		for _, bot := range bots {
			if bot.Id == botId {
				preferedBots = append(preferedBots, bot)
			}
		}
	}
	return preferedBots
}

func GetBotName(id int) string {
	bots := getPreferedBots()
	for _, bot := range bots {
		if bot.Id == id {
			return bot.Name
		}
	}
	return ""
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
	bots := getPreferedBots()

	series = formatString(series)
	// TODO create a non n^2 search 💀
	for _, bot := range bots {

		for _, pack := range search {
			quals := qualRegexp.FindStringSubmatch(pack.Name)
			if len(quals) <= 0 {
				continue
			}

			if (pack.BotId == bot.Id) && strings.EqualFold(quals[2]+"p", config.GetConfig().PreferedFormat) { // prefered bot && quality
				names := nameRegexp.FindStringSubmatch(pack.Name)
				if len(names) <= 0 {
					continue
				}

				name := formatString(names[2])

				if strings.EqualFold(name, series) {
					packs = append(packs, pack)
				}
			}
		}

		if len(packs) > 0 {
			break
		}
	}

	return packs, nil
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
	botPath := cwd + "\\data\\bots.json"

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
		os.MkdirAll(cwd+"\\data", 0777)

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
