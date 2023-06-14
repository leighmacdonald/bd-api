package main

import (
	"context"
	"github.com/PuerkitoBio/goquery"
	"github.com/gocolly/colly"
	"github.com/gocolly/colly/debug"
	"github.com/gocolly/colly/extensions"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"net/url"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

type nextURLFunc func(scraper *sbScraper, doc *goquery.Selection) string

type parseTimeFunc func(s string) (time.Time, error)

type parserFunc func(doc *goquery.Selection, timeParser parseTimeFunc, scraperName string) ([]sbRecord, int, error)

func initScrapers(ctx context.Context, db *pgStore, scrapers []*sbScraper) error {
	for _, scraper := range scrapers {
		var s sbSite
		if errSave := db.sbSiteGetOrCreate(ctx, scraper.name, &s); errSave != nil {
			return errors.Wrap(errSave, "Database error")
		}
		scraper.ID = uint32(s.SiteID)
	}
	return nil
}

func startScrapers(ctx context.Context, config *appConfig, scrapers []*sbScraper, db *pgStore) {
	do := func() {
		if config.ProxiesEnabled {
			startProxies(config)
			defer stopProxies()
			for _, scraper := range scrapers {
				if errProxies := setupProxies(scraper.Collector, config); errProxies != nil {
					logger.Panic("Failed to setup proxies", zap.Error(errProxies))
				}
			}
		}
		wg := &sync.WaitGroup{}
		for _, scraper := range scrapers {
			wg.Add(1)
			go func(s *sbScraper) {
				defer wg.Done()
				if errScrape := s.start(ctx, db); errScrape != nil {
					s.log.Error("Scraper returned error", zap.Error(errScrape))
				}
			}(scraper)
		}
		wg.Wait()
	}
	do()
	t0 := time.NewTicker(time.Hour * 24)
	for {
		select {
		case <-t0.C:
			do()
		case <-ctx.Done():
			return
		}
	}

}

type sbRecord struct {
	Name      string
	SteamID   steamid.SID64
	Reason    string
	CreatedOn time.Time
	Length    time.Duration
	Permanent bool
}

var errEmpty = errors.New("value empty")

func (r *sbRecord) setPlayer(scraperName string, name string) bool {
	if name == "" {
		logger.Error("Failed to set player", zap.String("scraper", scraperName), zap.Error(errEmpty))
		return false
	}
	r.Name = name
	return true
}

func (r *sbRecord) setInvokedOn(scraperName string, parseTime parseTimeFunc, value string) bool {
	t, errTime := parseTime(value)
	if errTime != nil {
		logger.Error("Failed to parse invoke time", zap.Error(errTime), zap.String("scraper", scraperName))
		return false
	}
	r.CreatedOn = t
	return true
}

func (r *sbRecord) setBanLength(value string) bool {
	lowerVal := strings.ToLower(value)
	if strings.Contains(lowerVal, "unbanned") {
		r.SteamID = 0 // invalidate it
	} else if lowerVal == "permanent" {
		r.Permanent = true
	}
	r.Length = 0
	return true
}

func (r *sbRecord) setExpiredOn(scraperName string, parseTime parseTimeFunc, value string) bool {
	if r.Permanent || !r.SteamID.Valid() {
		return false
	}
	t, errTime := parseTime(value)
	if errTime != nil {
		logger.Error("Failed to parse expire time", zap.Error(errTime), zap.String("scraper", scraperName))
		return false
	}
	r.Length = t.Sub(r.CreatedOn)
	return true
}

func (r *sbRecord) setReason(value string) bool {
	if value == "" {
		//logger.Error("Reason is empty", zap.String("scraper", scraperName))
		return false
	}
	r.Reason = value
	return true
}

func (r *sbRecord) setSteam(scraperName string, value string) bool {
	if r.SteamID.Valid() {
		return true
	}
	if value == "[U:1:0]" || value == "76561197960265728" {
		return false
	}
	sid64, errSid := steamid.StringToSID64(value)
	if errSid != nil {
		logger.Error("Failed to parse sid3", zap.Error(errSid), zap.String("scraper", scraperName))
		return false
	}
	r.SteamID = sid64
	return true
}

type sbScraper struct {
	*colly.Collector
	name      string
	theme     string
	log       *zap.Logger
	curPage   int
	results   []sbRecord
	resultsMu sync.RWMutex
	baseURL   string
	sleepTime time.Duration
	startPath string
	parser    parserFunc
	nextURL   nextURLFunc
	parseTIme parseTimeFunc
}

func createScrapers() []*sbScraper {
	return []*sbScraper{
		new7MauScraper(), newAceKillScraper(), newAMSGamingScraper(), newApeModeScraper(), newAstraManiaScraper(),
		newBachuruServasScraper(), newBaitedCommunityScraper(), newBierwieseScraper(), newBouncyBallScraper(), newCedaPugScraper(),
		newCSIServersScraper(), newCuteProjectScraper(), newCutiePieScraper(), newDarkPyroScraper(), newDefuseRoScraper(),
		newDiscFFScraper(), newDreamFireScraper(), newECJScraper(), newElectricScraper(), newFirePoweredScraper(),
		newFluxTFScraper(), newFurryPoundScraper(), newG44Scraper(), newGameSitesScraper(), newGamesTownScraper(),
		newGFLScraper(), newGhostCapScraper(), newGlobalParadiseScraper(), newGunServerScraper(), newHarpoonScraper(),
		newHellClanScraper(), newJumpAcademyScraper(), newLBGamingScraper(), newLOOSScraper(), newLazyNeerScraper(),
		newLazyPurpleScraper(), newMagyarhnsScraper(), newMaxDBScraper(), newNeonHeightsScraper(), newNideScraper(),
		newOpstOnlineScraper(), newOreonScraper(), newOwlTFScraper(), newPancakesScraper(), newPandaScraper(),
		newPetrolTFScraper(), newPhoenixSourceScraper(), newPowerFPSScraper(), newProGamesZetScraper(), newPubsTFScraper(),
		newRetroServersScraper(), newSGGamingScraper(), newSameTeemScraper(), newSavageServidoresScraper(), newScrapTFScraper(),
		newServiliveClScraper(), newSettiScraper(), newSirPleaseScraper(), newSkialScraper(), newSlavonServerScraper(),
		newSneaksScraper(), newSpaceShipScraper(), newSpectreScraper(), newSvdosBrothersScraper(), newSwapShopScraper(),
		newTF2MapsScraper(), newTF2ROScraper() /*newTawernaScraper(),*/, newTheVilleScraper(), newTitanScraper(),
		newTriggerHappyScraper(), newUGCScraper(), newVaticanCityScraper(), newVidyaGaemsScraper(), newWonderlandTFGOOGScraper(),
		newZMBrasilScraper(),
	}
}

func (scraper *sbScraper) start(ctx context.Context, db *pgStore) error {
	scraper.log.Info("Starting scrape job", zap.String("name", scraper.name), zap.String("theme", scraper.theme))
	lastURL := ""
	startTime := time.Now()
	totalErrorCount := 0
	scraper.Collector.OnHTML("body", func(e *colly.HTMLElement) {
		results, errorCount, parseErr := scraper.parser(e.DOM, scraper.parseTIme, scraper.name)
		if parseErr != nil {
			logger.Error("Parser returned error", zap.Error(parseErr))
			return
		}
		nextURL := scraper.nextURL(scraper, e.DOM)
		totalErrorCount += errorCount
		scraper.resultsMu.Lock()
		scraper.results = append(scraper.results, results...)
		scraper.resultsMu.Unlock()
		for _, result := range results {
			pr := newPlayerRecord(result.SteamID)
			if errPlayer := db.playerGetOrCreate(ctx, result.SteamID, &pr); errPlayer != nil {
				scraper.log.Error("failed to get player record", zap.Int64("sid64", result.SteamID.Int64()), zap.Error(errPlayer))
				continue
			}
			br := sbBanRecord{
				SiteID:      int(scraper.ID),
				SteamID:     pr.SteamID,
				Reason:      result.Reason,
				Duration:    result.Length,
				PersonaName: result.Name,
				Permanent:   result.Permanent,
				timeStamped: timeStamped{
					UpdatedOn: time.Now(),
					CreatedOn: result.CreatedOn,
				},
			}
			if errBanSave := db.sbBanSave(ctx, &br); errBanSave != nil {
				scraper.log.Error("Failed to save ban record", zap.Int64("sid64", pr.SteamID.Int64()), zap.Error(errBanSave))
			}
		}
		if nextURL != "" && nextURL != lastURL {
			lastURL = nextURL
			if scraper.sleepTime > 0 {
				time.Sleep(scraper.sleepTime)
			}
			scraper.log.Debug("Visiting next url", zap.String("url", nextURL))
			if errVisit := e.Request.Visit(nextURL); errVisit != nil && !errors.Is(errVisit, colly.ErrAlreadyVisited) {
				scraper.log.Error("Failed to visit sub url", zap.Error(errVisit), zap.String("url", nextURL))
				return
			}
		}
	})
	if errVisit := scraper.Visit(scraper.url(scraper.startPath)); errVisit != nil {
		return errVisit
	}
	scraper.Wait()
	scraper.log.Info("Completed scrape job", zap.String("name", scraper.name), zap.Int("valid", len(scraper.results)), zap.Int("skipped", totalErrorCount), zap.Duration("duration", time.Since(startTime)))
	return nil
}

type scrapeLogger struct {
	logger *zap.Logger
	start  time.Time
}

func (log *scrapeLogger) Init() error {
	log.start = time.Now()
	return nil
}

func (log *scrapeLogger) Event(e *debug.Event) {
	args := []zap.Field{zap.Uint32("col_id", e.CollectorID), zap.Uint32("req_id", e.RequestID), zap.Duration("duration", time.Since(log.start))}
	u, ok := e.Values["url"]
	if ok {
		args = append(args, zap.String("url", u))
	}
	switch e.Type {
	case "error":
		log.logger.Error("Error scraping url", args...)
	default:
		args = append(args, zap.String("type", e.Type))
		log.logger.Debug("Scraped url", args...)
	}
}

func newScraper(name string, baseURL string, startPath string, parser parserFunc, nextURL nextURLFunc, parseTime parseTimeFunc) *sbScraper {
	u, errURL := url.Parse(baseURL)
	if errURL != nil {
		logger.Panic("Failed to parse base url", zap.Error(errURL))
	}
	debugLogger := scrapeLogger{logger: logger}
	scraper := sbScraper{
		baseURL:   baseURL,
		name:      name,
		theme:     "default",
		startPath: startPath,
		curPage:   1,
		parser:    parser,
		nextURL:   nextURL,
		parseTIme: parseTime,
		log:       logger.Named(name),
		Collector: colly.NewCollector(
			colly.UserAgent("bd"),
			colly.CacheDir(filepath.Join(cacheDir, "scrapers")),
			colly.Debugger(&debugLogger),
			colly.AllowedDomains(u.Hostname()),
			//colly.Async(true),
			//colly.MaxDepth(2),
		),
	}
	scraper.SetRequestTimeout(time.Second * 30)
	scraper.OnRequest(func(r *colly.Request) {
		scraper.log.Debug("Visiting", zap.String("url", r.URL.String()))
	})
	extensions.RandomUserAgent(scraper.Collector)
	if errLimit := scraper.Limit(&colly.LimitRule{
		DomainGlob:  "*" + u.Hostname(),
		Parallelism: 2,
		RandomDelay: 5 * time.Second,
	}); errLimit != nil {
		scraper.log.Panic("Failed to set limit", zap.Error(errLimit))
	}
	scraper.OnError(func(r *colly.Response, err error) {
		scraper.log.Error("Request error", zap.String("url", r.Request.URL.String()), zap.Error(err))
	})
	return &scraper
}

func (scraper *sbScraper) url(path string) string {
	return scraper.baseURL + path
}

// 05-17-23 03:07
func parseSkialTime(s string) (time.Time, error) {
	if s == "Not applicable." || s == "Permanent" {
		return time.Time{}, nil
	}
	return time.Parse("01-02-06 15:04", s)
}

// 05-17-23 03:07
func parseRushyTime(s string) (time.Time, error) {
	if s == "Not applicable." || s == "Permanent" {
		return time.Time{}, nil
	}
	return time.Parse("15:04 pm 01/02/2006", s)
}

// 17-05-23 03:07
func parseBachuruServasTime(s string) (time.Time, error) {
	if s == "Not applicable." || s == "Permanent" {
		return time.Time{}, nil
	}
	return time.Parse("02-01-2006, 15:04", s)
}

// 05-17-23 03:07
func parseBaitedTime(s string) (time.Time, error) {
	if s == "Not applicable." || s == "Permanent" {
		return time.Time{}, nil
	}
	return time.Parse("02-01-2006 15:04", s)
}

// 05-17-23 03:07
func parseSkialAltTime(s string) (time.Time, error) {
	if s == "Not applicable." || s == "Permanent" {
		return time.Time{}, nil
	}
	return time.Parse("06-01-02 15:04", s)
}

// 05-17-23 03:07
func parseGunServer(s string) (time.Time, error) {
	if s == "Not applicable." || s == "Permanent" {
		return time.Time{}, nil
	}
	return time.Parse("02.01.2006 15:04", s)
}

// 08.06.2023 в 21:21
func parseProGamesZetTime(s string) (time.Time, error) {
	if s == "Not applicable." || s == "Permanent" || s == "Никогда." {
		return time.Time{}, nil
	}
	return time.Parse("02.01.2006 в 15:04", s)
}

// 17/05/23 - 03:07:05
func parseSVDos(s string) (time.Time, error) {
	if s == "Not applicable." || s == "Permanent" {
		return time.Time{}, nil
	}
	return time.Parse("02/01/06 - 15:04:05", s)
}

// 17/05/23 - 03:07:05
func parseTriggerHappyTime(s string) (time.Time, error) {
	if s == "Not applicable." || s == "Permanent" {
		return time.Time{}, nil
	}
	return time.Parse("02/01/2006 15:04:05", s)
}

// 17/05/23 03:07 PM
func parseDarkPyroTime(s string) (time.Time, error) {
	if s == "Not applicable." || s == "Permanent" {
		return time.Time{}, nil
	}
	return time.Parse("01/02/06 15:04 PM", s)
}

// 17-05-2023 03:07:05
func parseTrailYear(s string) (time.Time, error) {
	if s == "Not applicable." || s == "Permanent" {
		return time.Time{}, nil
	}
	return time.Parse("02-01-2006 15:04:05", s)
}

// 17-05-2023 03:07:05
func parseHellClanTime(s string) (time.Time, error) {
	if s == "Not applicable." || s == "Permanent" {
		return time.Time{}, nil
	}
	return time.Parse("02-01-2006 15:04 MST", s)
}

// 05-31-2023 9:57 PM CDT
func parseSneakTime(s string) (time.Time, error) {
	if s == "Not applicable." || s == "Permanent" {
		return time.Time{}, nil
	}
	return time.Parse("01-02-2006 15:04 PM MST", s)
}

// 24-06-2023 11:15:11 IST
func parseAMSGamingTime(s string) (time.Time, error) {
	if s == "Not applicable." || s == "Permanent" {
		return time.Time{}, nil
	}
	return time.Parse("02-01-2006 15:04:05 MST", s)
}

// 2023-05-17 03:07:05
func parseDefaultTime(s string) (time.Time, error) {
	if s == "Not applicable." {
		return time.Time{}, nil
	}
	return time.Parse("2006-01-02 15:04:05", s)
}

// 2023-17-05 03:07:05  / 2023-26-05 10:56:53
//
// nolint
func parseDefaultTimeMonthFirst(s string) (time.Time, error) {
	if s == "Not applicable." {
		return time.Time{}, nil
	}
	return time.Parse("2006-02-01 15:04:05", s)
}

// Thu, May 11, 2023 7:14 PM    / Fri, Jun 2, 2023 6:40 PM
func parsePancakesTime(s string) (time.Time, error) {
	if s == "Not applicable." || s == "never, this is permanent" {
		return time.Time{}, nil
	}
	return time.Parse("Mon, Jan 2, 2006 15:04 PM", s)
}

// Thu, May 11, 2023 7:14 PM
//func parseOtakuTime(s string) (time.Time, error) {
//	if s == "Not applicable." || s == "never, this is permanent" {
//		return time.Time{}, nil
//	}
//	return time.Parse("Jan-2-2006 15:04:05", s)
//}

// Thu, May 11, 2023 7:14 PM
func parseTitanTime(s string) (time.Time, error) {
	if s == "Not applicable." || s == "never, this is permanent" {
		return time.Time{}, nil
	}
	return time.Parse("Monday, 2 Jan 2006 15:04:05 PM", s)
}

// May 11, 2023 7:14 PM
func parseSGGamingTime(s string) (time.Time, error) {
	if s == "Not applicable." || s == "never, this is permanent" {
		return time.Time{}, nil
	}
	return time.Parse("Jan 02, 2006 15:04 PM", s)
}

// May 11, 2023 7:14 PM   / June 7, 2022, 1:15 am
func parseFurryPoundTime(s string) (time.Time, error) {
	if s == "Not applicable." || s == "never, this is permanent" {
		return time.Time{}, nil
	}
	return time.Parse("January _2, 2006, 15:04 pm", s)
}

// Sunday 11th of May 2023 7:14:05 PM
func parseFluxTime(s string) (time.Time, error) {
	rx := regexp.MustCompile(`\s(\d+)(st|nd|rd|th)\s`)
	t := rx.ReplaceAllString(s, " $1 ")
	if s == "Not applicable." || s == "never, this is permanent" {
		return time.Time{}, nil
	}
	return time.Parse("Monday _2 of January 2006 15:04:05 PM", t)
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

func nextURLFluent(scraper *sbScraper, doc *goquery.Selection) string {
	nextPage := ""
	nodes := doc.Find(".pagination a[href]")
	nodes.EachWithBreak(func(i int, selection *goquery.Selection) bool {
		v := selection.Text()
		if strings.Contains(strings.ToLower(v), "next") {
			nextPage, _ = selection.Attr("href")
			return false
		}
		return true
	})
	return scraper.url(nextPage)
}

func nextURLFirst(scraper *sbScraper, doc *goquery.Selection) string {
	nextPage, _ := doc.Find("#banlist-nav a[href]").First().Attr("href")
	return scraper.url(nextPage)
}

func nextURLLast(scraper *sbScraper, doc *goquery.Selection) string {
	nextPage, _ := doc.Find("#banlist-nav a[href]").Last().Attr("href")
	if !strings.Contains(nextPage, "page=") {
		return ""
	}
	return scraper.url(nextPage)
}

type mappedKey string

const (
	keyCommunityLinks mappedKey = "community links"
	keyInvokedOn      mappedKey = "invoked on"
	keySteamCommunity mappedKey = "steam community"
	keyBanLength      mappedKey = "ban length"
	keyExpiredOn      mappedKey = "expires on"
	//keyReasonUnbanned mappedKey = "reason unbanned"
	keyReason   mappedKey = "reason"
	keySteam3ID mappedKey = "steam3"
	keyPlayer   mappedKey = "player"
)

var keyMap = map[string]mappedKey{
	"community links":     "community links",
	"был выдан":           "invoked on",
	"datum a čas udělení": "invoked on",
	"invoked on":          "invoked on",
	"steam community":     "steam community",
	"steam komunitní":     "steam community",
	"délka":               "ban length",
	"banlength":           "ban length",
	"ban length":          "ban length",
	"длительность":        "ban length",
	"vyprší":              "expires on",
	"будет снят":          "expires on",
	"expires on":          "expires on",
	"причина разбана":     "reason unbanned",
	"důvod":               "reason",
	"reason":              "reason",
	"разбанен админом":    "unbanned by",
	"причина бана":        "reason",
	"игрок":               "player",
	"player":              "player",
	"steam3 id":           "steam3 id",
}

var spaceRm = regexp.MustCompile(`\s+`)

func normKey(s string) string {
	return strings.ReplaceAll(spaceRm.ReplaceAllString(strings.TrimSpace(strings.ToLower(s)), " "), "\n", "")
}

func getMappedKey(s string) (mappedKey, bool) {
	mk, found := keyMap[s]
	if !found {
		return "", false
	}
	return mk, true
}

// https://github.com/SB-MaterialAdmin/Web/tree/stable-dev
func parseMaterial(doc *goquery.Selection, parseTime parseTimeFunc, scraperName string) ([]sbRecord, int, error) {
	var (
		bans      []sbRecord
		curBan    sbRecord
		skipCount int
	)
	doc.Find("div.opener .card-body").Each(func(_ int, selection *goquery.Selection) {
		selection.First().Children().Children().Each(func(i int, selection *goquery.Selection) {
			children := selection.First().Children()
			first := children.First()
			second := children.Last()
			key := normKey(first.Contents().Text())
			value := strings.TrimSpace(second.Contents().Text())
			mk, ok := getMappedKey(key)
			if !ok {
				return
			}
			switch mk {
			case keyPlayer:
				curBan.setPlayer(scraperName, value)
			case keySteam3ID:
				curBan.setSteam(scraperName, value)
			case keyCommunityLinks:
				if curBan.SteamID.Valid() {
					return
				}
				nv, foundKey := second.Children().First().Attr("href")
				if !foundKey {
					return
				}
				pcs := strings.Split(nv, "/")
				curBan.setSteam(scraperName, pcs[4])
			case keySteamCommunity:
				pts := strings.Split(value, " ")
				curBan.setSteam(scraperName, pts[0])
			case keyInvokedOn:
				curBan.setInvokedOn(scraperName, parseTime, value)
			case keyBanLength:
				curBan.setBanLength(value)
			case keyExpiredOn:
				curBan.setExpiredOn(scraperName, parseTime, value)
			case keyReason:
				curBan.setReason(value)
				curBan.Reason = value
				if curBan.SteamID.Valid() {
					bans = append(bans, curBan)
				} else {
					skipCount++
				}
				curBan = sbRecord{}
			}
		})

	})
	return bans, skipCount, nil
}

// https://github.com/brhndursun/SourceBans-StarTheme
func parseStar(doc *goquery.Selection, parseTime parseTimeFunc, scraperName string) ([]sbRecord, int, error) {
	var (
		bans      []sbRecord
		curBan    sbRecord
		skipCount int
	)
	doc.Find("div").Each(func(_ int, selection *goquery.Selection) {
		idAttr, ok := selection.Attr("id")
		if !ok {
			return
		}
		if !strings.HasPrefix(idAttr, "expand_") {
			return
		}
		selection.Find("tbody tr").Each(func(i int, selection *goquery.Selection) {
			if i == 0 {
				return
			}
			children := selection.Children()
			if len(children.Nodes) == 3 {
				curBan.Name = strings.TrimSpace(children.First().Next().Text())
				return
			}
			first := children.First()
			second := children.Last()
			key := normKey(first.Contents().Text())
			value := strings.TrimSpace(second.Contents().Text())
			mk, found := getMappedKey(key)
			if !found {
				return
			}
			switch mk {
			case keyPlayer:
				curBan.setPlayer(scraperName, value)
			case keyCommunityLinks:
				nv, foundHREF := second.Children().First().Attr("href")
				if !foundHREF {
					return
				}
				pcs := strings.Split(nv, "/")
				curBan.setSteam(scraperName, pcs[4])
			case keySteam3ID:
				curBan.setSteam(scraperName, value)
			case keySteamCommunity:
				if curBan.SteamID.Valid() {
					return
				}
				pts := strings.Split(value, " ")
				curBan.setSteam(scraperName, pts[0])
			case keyInvokedOn:
				curBan.setInvokedOn(scraperName, parseTime, value)
			case keyBanLength:
				curBan.setBanLength(value)
			case keyExpiredOn:
				curBan.setExpiredOn(scraperName, parseTime, value)
			case keyReason:
				curBan.setReason(value)
				if curBan.SteamID.Valid() {
					bans = append(bans, curBan)
				} else {
					skipCount++
				}
				curBan = sbRecord{}
			}
		})

	})
	return bans, skipCount, nil
}

// https://github.com/aXenDeveloper/sourcebans-web-theme-fluent
func parseFluent(doc *goquery.Selection, parseTime parseTimeFunc, scraperName string) ([]sbRecord, int, error) {
	var (
		bans      []sbRecord
		curBan    sbRecord
		skipCount int
	)
	doc.Find("ul.ban_list_detal li").Each(func(i int, selection *goquery.Selection) {
		child := selection.Children()
		key := normKey(child.First().Contents().Text())
		value := strings.TrimSpace(child.Last().Contents().Text())
		mk, found := getMappedKey(key)
		if !found {
			return
		}
		switch mk {
		case keyPlayer:
			curBan.setPlayer(scraperName, value)
		case keySteam3ID:
			curBan.setSteam(scraperName, value)
		case keySteamCommunity:
			pts := strings.Split(value, " ")
			curBan.setSteam(scraperName, pts[0])
		case keyInvokedOn:
			curBan.setInvokedOn(scraperName, parseTime, value)
		case keyBanLength:
			curBan.setBanLength(value)
		case keyExpiredOn:
			curBan.setExpiredOn(scraperName, parseTime, value)
		case keyReason:
			curBan.setReason(value)
			if curBan.SteamID.Valid() {
				bans = append(bans, curBan)
			} else {
				skipCount++
			}
			curBan = sbRecord{}
		}
	})
	return bans, skipCount, nil
}

func parseDefault(doc *goquery.Selection, parseTime parseTimeFunc, scraperName string) ([]sbRecord, int, error) {
	var (
		bans     []sbRecord
		curBan   sbRecord
		curState mappedKey
		isValue  bool
		skipped  int
	)
	doc.Find("#banlist .listtable table tr td").Each(func(i int, selection *goquery.Selection) {
		value := strings.TrimSpace(selection.Text())
		if !isValue {
			switch strings.ToLower(value) {
			case "player":
				curState = keyPlayer
				isValue = true
			case "steam community":
				curState = keySteamCommunity
				isValue = true
			case "invoked on":
				curState = keyInvokedOn
				isValue = true
			case "banlength":
				curState = keyBanLength
				isValue = true
			case "expires on":
				curState = keyExpiredOn
				isValue = true
			case "reason":
				curState = keyReason
				isValue = true
			}
		} else {
			isValue = false
			switch curState {
			case keyPlayer:
				curBan.setPlayer(scraperName, value)
			case keySteamCommunity:
				pts := strings.Split(value, " ")
				curBan.setSteam(scraperName, pts[0])
			case keyInvokedOn:
				curBan.setInvokedOn(scraperName, parseTime, value)
			case keyBanLength:
				curBan.setBanLength(value)
			case keyExpiredOn:
				curBan.setExpiredOn(scraperName, parseTime, value)
			case keyReason:
				curBan.setReason(value)
				if curBan.SteamID.Valid() {
					bans = append(bans, curBan)
				} else {
					skipped++
				}
				curBan = sbRecord{}
			}
		}
	})
	return bans, skipped, nil
}

//
//type megaScatterNode struct {
//	ID                  string `json:"id"`
//	ID3                 string `json:"id3"`
//	ID1                 string `json:"id1"`
//	Label               string `json:"label"`
//	BorderWidthSelected int    `json:"borderWidthSelected"`
//	Shape               string `json:"shape"`
//	Color               struct {
//		Border     string `json:"border"`
//		Background string `json:"background"`
//		Highlight  struct {
//			Border     string `json:"border"`
//			Background string `json:"background"`
//		} `json:"highlight"`
//	} `json:"color"`
//	X       float64  `json:"x"`
//	Y       float64  `json:"y"`
//	Aliases []string `json:"-"`
//}

////nolint:golint,unused
//func parseMegaScatter(bodyReader io.Reader) ([]sbRecord, error) {
//	body, errBody := io.ReadAll(bodyReader)
//	if errBody != nil {
//		return nil, errBody
//	}
//	rx := regexp.MustCompile(`(?s)var nodes = new vis.DataSet\((\[.+?])\);`)
//	match := rx.FindStringSubmatch(string(body))
//	if len(match) == 0 {
//		return nil, errors.New("Failed to match data")
//	}
//	pass1 := strings.Replace(match[1], "'", "", -1)
//	replacer := regexp.MustCompile(`\s(\S+?):\s`)
//	pass2 := replacer.ReplaceAllString(pass1, "\"$1\": ")
//
//	replacer2 := regexp.MustCompile(`]},\s]$`)
//	pass3 := replacer2.ReplaceAllString(pass2, "]}]")
//
//	fmt.Println(pass3[0:1024])
//
//	fmt.Println(string(pass3[len(match[1])-2048]))
//
//	o, _ := os.Create("temp.json")
//	_, _ = io.WriteString(o, pass3)
//	_ = o.Close()
//	var msNodes []megaScatterNode
//	if errJSON := json.Unmarshal([]byte(pass3), &msNodes); errJSON != nil {
//		return nil, errJSON
//	}
//	return nil, nil
//}
