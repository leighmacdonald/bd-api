package main

import (
	"encoding/json"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/gocolly/colly"
	"github.com/gocolly/colly/debug"
	"github.com/gocolly/colly/extensions"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"io"
	"net/url"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"
)

type rowFilter func(doc *goquery.Selection) bool

type nextUrlFunc func(doc *goquery.Selection) string

type parseTimeFunc func(s string) (time.Time, error)

type parserFunc func(doc *goquery.Selection, nextUrl nextUrlFunc, timeParser parseTimeFunc) (string, []banData, error)

type Startable interface {
	Start() error
}

func Start() {
	if errConfig := readConfig("config.yml"); errConfig != nil {
		logger.Panic("Failed to load config", zap.Error(errConfig))
	}
	startProxies()
	defer stopProxies()

	for _, scraper := range []Startable{
		NewSkialScraper(),
		NewGFLScraper(),
		NewSpaceShipScraper(),
		NewLazyPurpleScraper(),
	} {
		go func(s Startable) {
			if errScrape := s.Start(); errScrape != nil {
				logger.Error("Scraper returned error", zap.Error(errScrape))
			}
		}(scraper)
	}
}

type Scraper struct {
	*colly.Collector
	name      string
	log       *zap.Logger
	results   []banData
	resultsMu sync.RWMutex
	baseUrl   string
	startPath string
	parser    parserFunc
	nextUrl   nextUrlFunc
	parseTIme parseTimeFunc
}

func (scraper *Scraper) Start() error {
	scraper.Collector.OnHTML("*", func(e *colly.HTMLElement) {
		nextUrl, results, parseErr := parseDefault(e.DOM, scraper.nextUrl, scraper.parseTIme)
		if parseErr != nil {
			logger.Error("Parser returned error", zap.Error(parseErr))
			return
		}
		scraper.resultsMu.Lock()
		scraper.results = append(scraper.results, results...)
		scraper.resultsMu.Unlock()
		if nextUrl != "" {
			if errVisit := scraper.Visit(scraper.url(nextUrl)); errVisit != nil {
				logger.Error("Failed to visit sub url", zap.Error(errVisit), zap.String("url", nextUrl))
				return
			}
		}
	})
	if errVisit := scraper.Visit(scraper.url(scraper.startPath)); errVisit != nil {
		return errVisit
	}
	scraper.Wait()
	return nil
}

func newScraper(name string, baseUrl string, startPath string, parser parserFunc, nextUrl nextUrlFunc, parseTime parseTimeFunc) *Scraper {
	u, errUrl := url.Parse(baseUrl)
	if errUrl != nil {
		logger.Panic("Failed to parse base url", zap.Error(errUrl))
	}

	scraper := Scraper{
		baseUrl:   baseUrl,
		name:      name,
		startPath: startPath,
		parser:    parser,
		nextUrl:   nextUrl,
		parseTIme: parseTime,
		log:       logger.Named(name),
		Collector: colly.NewCollector(
			colly.Debugger(&debug.LogDebugger{}),
			colly.AllowedDomains(u.Hostname()),
			colly.Async(true),
			colly.MaxDepth(2),
		),
	}

	extensions.RandomUserAgent(scraper.Collector)

	if errLimit := scraper.Limit(&colly.LimitRule{
		DomainGlob:  "*" + u.Hostname(),
		Parallelism: 2,
		RandomDelay: 5 * time.Second,
	}); errLimit != nil {
		logger.Panic("Failed to set limit", zap.Error(errLimit))
	}

	if errProxies := setupProxies(scraper.Collector); errProxies != nil {
		logger.Panic("Failed to setup proxies", zap.Error(errProxies))
	}

	scraper.OnError(func(r *colly.Response, err error) {
		logger.Error("Request error", zap.String("url", r.Request.URL.String()), zap.Error(err))
	})

	return &scraper
}

func (scraper *Scraper) url(path string) string {
	return scraper.baseUrl + path
}

func NewSkialScraper() *Scraper {
	return newScraper(
		"skial",
		"https://www.skial.com/sourcebans/",
		"index.php?p=banlist",
		parseDefault,
		nextUrlFirst,
		parseSkialTime,
	)
}

func NewGFLScraper() *Scraper {
	return newScraper("gfl", "https://sourcebans.gflclan.com/", "index.php?p=banlist",
		parseDefault, nextUrlLast, parseDefaultTime)
}

func NewSpaceShipScraper() *Scraper {
	return newScraper("spaceship", "https://sappho.io/bans/", "index.php?p=banlist",
		parseDefault, nextUrlLast, parseDefaultTime)
}

func NewUGCScraper() *Scraper {
	return newScraper("ugc", "https://sb.ugc-gaming.net/", "index.php?p=banlist",
		parseFluent, nextUrlFluent, parseDefaultTime)
}

func NewSirPleaseScraper() *Scraper {
	return newScraper("sirplease", "https://sirplease.gg/", "index.php?p=banlist",
		parseFluent, nextUrlFluent, parseDefaultTime)
}

func NewVidyaGaemsScraper() *Scraper {
	return newScraper("vidyagaems", "https://www.vidyagaems.net/", "index.php?p=banlist",
		parseFluent, nextUrlFluent, parseTrailYear)
}

func NewOwlTFScraper() *Scraper {
	return newScraper("owl", "https://kingpandagamer.xyz/sb/", "index.php?p=banlist",
		parseDefault, nextUrlLast, parseDefaultTime)
}

func NewZMBrasilScraper() *Scraper {
	return newScraper("zmbrasil", "http://bans.zmbrasil.com.br/", "index.php?p=banlist",
		parseDefault, nextUrlLast, parseSkialTime)
}

func NewScrapTFScraper() *Scraper {
	return newScraper("scraptf", "https://bans.scrap.tf/", "index.php?p=banlist",
		parseDefault, nextUrlLast, parseDefaultTime)
}

func NewWonderlandTFScraper() *Scraper {
	return newScraper("wonderland", "https://bans.wonderland.tf/", "index.php?p=banlist",
		parseDefault, nextUrlLast, parseWonderlandTime)
}

func NewLazyPurpleScraper() *Scraper {
	return newScraper("lazypurple", "https://www.lazypurple.com/sourcebans/", "index.php?p=banlist",
		parseDefault, nextUrlLast, parseDefaultTime)
}

func NewFirePoweredScraper() *Scraper {
	return newScraper("firepowered", "https://firepoweredgaming.com/sourcebanspp/", "index.php?p=banlist",
		parseDefault, nextUrlLast, parseSkialTime)
}

func NewHarpoonScraper() *Scraper {
	return newScraper("harpoongaming", "https://bans.harpoongaming.com/", "index.php?p=banlist",
		parseDefault, nextUrlLast, parseDefaultTime)
}

func NewPandaScraper() *Scraper {
	return newScraper("panda", "https://bans.panda-community.com/", "index.php?p=banlist",
		parseFluent, nextUrlFluent, parseDefaultTime)
}

func NewNeonHeightsScraper() *Scraper {
	return newScraper("neonheights", "https://neonheights.xyz/bans/", "index.php?p=banlist",
		parseDefault, nextUrlLast, parseSkialTime)
}

func NewPancakesScraper() *Scraper {
	return newScraper("pancakes", "https://pancakes.tf/", "index.php?p=banlist",
		parseDefault, nextUrlLast, parsePancakesTime)
}

func NewLOOSScraper() *Scraper {
	return newScraper("loos", "https://looscommunity.com/bans/", "index.php?p=banlist",
		parseDefault, nextUrlLast, parseDefaultTime)
}

func NewPubsTFScraper() *Scraper {
	return newScraper("pubstf", "https://bans.pubs.tf/", "index.php?p=banlist",
		parseDefault, nextUrlLast, parseSkialTime)
}

func NewServiliveClScraper() *Scraper {
	return newScraper("servilivecl", "https://sourcebans.servilive.cl/", "index.php?p=banlist",
		parseFluent, nextUrlFluent, parseDefaultTimeMonthFirst)
}

func NewCutiePieScraper() *Scraper {
	return newScraper("cutiepie", "https://bans.cutiepie.tf/", "index.php?p=banlist",
		parseDefault, nextUrlLast, parseDefaultTime)
}
func NewSGGamingScraper() *Scraper {
	return newScraper("sggaming", "https://sg-gaming.net/bans/", "index.php?p=banlist",
		parseDefault, nextUrlLast, parseSGGamingTime)
}

func NewApeModeScraper() *Scraper {
	return newScraper("apemode", "https://sourcebans.apemode.tf/", "index.php?p=banlist",
		parseDefault, nextUrlLast, parseSkialTime)
}

func NewMaxDBScraper() *Scraper {
	return newScraper("maxdb", "https://bans.maxdb.net/", "index.php?p=banlist",
		parseDefault, nextUrlLast, parseDefaultTime)
}

func NewSvdosBrothersScraper() *Scraper {
	return newScraper("svdosbrothers", "https://bans.svdosbrothers.com/", "index.php?p=banlist",
		parseFluent, nextUrlFluent, parseSVDos)
}

func NewElectricScraper() *Scraper {
	return newScraper("electric", "http://168.181.184.179/", "index.php?p=banlist",
		parseFluent, nextUrlFluent, parseDefaultTime)
}

func NewGlobalParadiseScraper() *Scraper {
	return newScraper("globalparadise", "https://bans.theglobalparadise.org/", "index.php?p=banlist",
		parseDefault, nextUrlLast, parseDefaultTime)
}

func NewSavageServidoresScraper() *Scraper {
	return newScraper("savageservidores", "https://bans.savageservidores.com/", "index.php?p=banlist",
		parseFluent, nextUrlFluent, parseDefaultTime)
}

func NewCSIServersScraper() *Scraper {
	return newScraper("csiservers", "https://bans.csiservers.com/", "index.php?p=banlist",
		parseDefault, nextUrlLast, parseDefaultTime)
}

func NewLBGamingScraper() *Scraper {
	return newScraper("lbgaming", "https://bans.lbgaming.co/", "index.php?p=banlist",
		parseDefault, nextUrlLast, parseSkialTime)
}

func NewFluxTFScraper() *Scraper {
	return newScraper("fluxtf", "https://bans.flux.tf/", "index.php?p=banlist",
		parseDefault, nextUrlLast, parseFluxTime)
}

func NewDarkPyroScraper() *Scraper {
	return newScraper("darkpyro", "https://bans.darkpyrogaming.com/", "index.php?p=banlist",
		parseDefault, nextUrlLast, parseDarkPyroTime)
}

func NewOpstOnlineScraper() *Scraper {
	return newScraper("opstonline", "https://www.opstonline.com/bans/", "index.php?p=banlist",
		parseDefault, nextUrlLast, parseSkialTime)
}

type metaKey int

const (
	unknown metaKey = iota
	player
	steamId
	steam2
	steamComm
	invokedOn
	banLength // "Permanent"
	expiresOn // "Not applicable."
	reason
	last
)

type banData struct {
	Name      string
	SteamId   steamid.SID64
	Reason    string
	CreatedOn time.Time
	Length    time.Duration
	Permanent bool
}

// 05-17-23 03:07
func parseSkialTime(s string) (time.Time, error) {
	if s == "Not applicable." || s == "Permanent" {
		return time.Time{}, nil
	}
	return time.Parse("01-02-06 15:04", s)
}

// 17/05/23 - 03:07:05
func parseSVDos(s string) (time.Time, error) {
	if s == "Not applicable." || s == "Permanent" {
		return time.Time{}, nil
	}
	return time.Parse("02/01/06 - 15:04:05", s)
}

// 17/05/23 03:07 PM
func parseDarkPyroTime(s string) (time.Time, error) {
	if s == "Not applicable." || s == "Permanent" {
		return time.Time{}, nil
	}
	return time.Parse("02/01/06 15:04 PM", s)
}

// 17-05-2023 03:07:05
func parseTrailYear(s string) (time.Time, error) {
	if s == "Not applicable." || s == "Permanent" {
		return time.Time{}, nil
	}
	return time.Parse("02-01-2006 15:04:05", s)
}

// 2023-05-17 03:07:05
func parseDefaultTime(s string) (time.Time, error) {
	if s == "Not applicable." {
		return time.Time{}, nil
	}
	return time.Parse("2006-01-02 15:04:05", s)
}

// 2023-17-05 03:07:05
func parseDefaultTimeMonthFirst(s string) (time.Time, error) {
	if s == "Not applicable." {
		return time.Time{}, nil
	}
	return time.Parse("2006-02-01 15:04:05", s)
}

// Thu, May 11, 2023 7:14 PM
func parsePancakesTime(s string) (time.Time, error) {
	if s == "Not applicable." || s == "never, this is permanent" {
		return time.Time{}, nil
	}
	return time.Parse("Mon, Jan 02, 2006 15:04 PM", s)
}

// May 11, 2023 7:14 PM
func parseSGGamingTime(s string) (time.Time, error) {
	if s == "Not applicable." || s == "never, this is permanent" {
		return time.Time{}, nil
	}
	return time.Parse("Jan 02, 2006 15:04 PM", s)
}

// Sunday 11th of May 2023 7:14:05 PM
func parseFluxTime(s string) (time.Time, error) {
	if s == "Not applicable." || s == "never, this is permanent" {
		return time.Time{}, nil
	}
	for _, k := range []string{"st ", "nd ", "rd ", "th "} {
		s = strings.Replace(s, k, " ", -1)
	}
	return time.Parse("Monday _2 of January 2006 15:04:05 PM", s)
}

// May 17th, 2023 (6:56)
func parseWonderlandTime(s string) (time.Time, error) {
	if s == "Not applicable." {
		return time.Time{}, nil
	}
	for _, k := range []string{"st", "nd", "rd", "th"} {
		s = strings.Replace(s, k, "", -1)
	}
	return time.Parse("January 2, 2006 (15:04)", s)
}

func nextUrlFluent(doc *goquery.Selection) string {
	nextPage, _ := doc.Find(".pagination a[href]").First().Attr("href")
	return nextPage
}

func nextUrlFirst(doc *goquery.Selection) string {
	nextPage, _ := doc.Find("#banlist-nav a[href]").First().Attr("href")
	return nextPage
}

func nextUrlLast(doc *goquery.Selection) string {
	nextPage, _ := doc.Find("#banlist-nav a[href]").Last().Attr("href")
	return nextPage
}

// https://github.com/aXenDeveloper/sourcebans-web-theme-fluent
func parseFluent(doc *goquery.Selection, urlFunc nextUrlFunc, parseTime parseTimeFunc) (string, []banData, error) {
	var (
		bans   []banData
		curBan banData
	)
	doc.Find("ul.ban_list_detal li").Each(func(i int, selection *goquery.Selection) {
		child := selection.Children()
		key := strings.TrimSpace(strings.ToLower(child.First().Contents().Text()))
		value := strings.TrimSpace(child.Last().Contents().Text())
		switch key {
		case "player":
			curBan.Name = value
		case "steam community":
			pts := strings.Split(value, " ")
			sid64, errSid := steamid.StringToSID64(pts[0])
			if errSid != nil {
				logger.Error("Failed to parse sid", zap.Error(errSid))
				return
			}
			curBan.SteamId = sid64
		case "invoked on":
			t, errTime := parseTime(value)
			if errTime != nil {
				logger.Error("Failed to parse invoke time", zap.Error(errTime))
				return
			}
			curBan.CreatedOn = t
		case "ban length":
			lowerVal := strings.ToLower(value)
			if strings.Contains(lowerVal, "unbanned") {
				curBan.SteamId = 0 // invalidate it
			} else if "permanent" == lowerVal {
				curBan.Permanent = true
			}
			curBan.Length = 0
		case "expires on":
			if curBan.Permanent || !curBan.SteamId.Valid() {
				return
			}
			t, errTime := parseTime(value)
			if errTime != nil {
				logger.Error("Failed to parse expire time", zap.Error(errTime))
				return
			}
			curBan.Length = t.Sub(curBan.CreatedOn)
		case "reason":
			curBan.Reason = value
			if curBan.SteamId.Valid() {
				bans = append(bans, curBan)
			}
			curBan = banData{}
		}
	})
	return urlFunc(doc), bans, nil
}

func parseDefault(doc *goquery.Selection, urlFunc nextUrlFunc, parseTime parseTimeFunc) (string, []banData, error) {
	var (
		bans     []banData
		curBan   banData
		curState = unknown
		isValue  bool
	)
	doc.Find("#banlist .listtable table tr td").Each(func(i int, selection *goquery.Selection) {
		// "#banlist table table tr td
		txt := strings.TrimSpace(selection.Text())
		if !isValue {
			switch strings.ToLower(txt) {
			case "player":
				curState = player
				isValue = true
			case "steam id":
				curState = steamId
				isValue = true
			case "steam2":
				curState = steam2
				isValue = true
			case "steam community":
				curState = steamComm
				isValue = true
			case "invoked on":
				curState = invokedOn
				isValue = true
			case "banlength":
				curState = banLength
				isValue = true
			case "expires on":
				curState = expiresOn
				isValue = true
			case "reason":
				curState = reason
				isValue = true
			}
		} else {
			isValue = false
			switch curState {
			case player:
				curBan.Name = txt
			case steamComm:
				pts := strings.Split(txt, " ")
				sid64, errSid := steamid.StringToSID64(pts[0])
				if errSid != nil {
					logger.Error("Failed to parse sid", zap.Error(errSid))
					return
				}
				curBan.SteamId = sid64
			case invokedOn:
				t, errTime := parseTime(txt)
				if errTime != nil {
					logger.Error("Failed to parse invoke time", zap.Error(errTime))
					return
				}
				curBan.CreatedOn = t
			case banLength:
				if "permanent" == strings.ToLower(txt) {
					curBan.Permanent = true
				}
				curBan.Length = 0
			case expiresOn:
				if curBan.Permanent {
					return
				}
				t, errTime := parseTime(txt)
				if errTime != nil {
					logger.Error("Failed to parse expire time", zap.Error(errTime))
					return
				}
				curBan.Length = t.Sub(curBan.CreatedOn)
			case reason:
				curBan.Reason = txt
				if curBan.SteamId.Valid() {
					bans = append(bans, curBan)
				}
				curBan = banData{}
				curState = last
			}
			curState = unknown
		}

	})
	return urlFunc(doc), bans, nil
}

type megaScatterNode struct {
	Id                  string `json:"id"`
	Id3                 string `json:"id3"`
	Id1                 string `json:"id1"`
	Label               string `json:"label"`
	BorderWidthSelected int    `json:"borderWidthSelected"`
	Shape               string `json:"shape"`
	Color               struct {
		Border     string `json:"border"`
		Background string `json:"background"`
		Highlight  struct {
			Border     string `json:"border"`
			Background string `json:"background"`
		} `json:"highlight"`
	} `json:"color"`
	X       float64  `json:"x"`
	Y       float64  `json:"y"`
	Aliases []string `json:"-"`
}

func parseMegaScatter(bodyReader io.Reader) ([]banData, error) {
	body, errBody := io.ReadAll(bodyReader)
	if errBody != nil {
		return nil, errBody
	}
	rx := regexp.MustCompile(`(?s)var nodes = new vis.DataSet\((\[.+?])\);`)
	match := rx.FindStringSubmatch(string(body))
	if len(match) == 0 {
		return nil, errors.New("Failed to match data")
	}
	pass1 := strings.Replace(match[1], "'", "", -1)
	replacer := regexp.MustCompile(`\s(\S+?):\s`)
	pass2 := replacer.ReplaceAllString(pass1, "\"$1\": ")

	replacer2 := regexp.MustCompile(`]},\s]$`)
	pass3 := replacer2.ReplaceAllString(pass2, "]}]")

	fmt.Println(string(pass3[0:1024]))

	fmt.Println(string(pass3[len(match[1])-2048]))

	o, _ := os.Create("temp.json")
	io.WriteString(o, pass3)
	o.Close()
	var msNodes []megaScatterNode
	if errJson := json.Unmarshal([]byte(pass3), &msNodes); errJson != nil {
		return nil, errJson
	}
	return nil, nil
}
