package main

import (
	"context"
	"fmt"
	"net/url"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/gocolly/colly"
	"github.com/gocolly/colly/debug"
	"github.com/gocolly/colly/extensions"
	"github.com/gocolly/colly/queue"
	"github.com/leighmacdonald/bd-api/models"
	"github.com/leighmacdonald/steamid/v3/steamid"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

type nextURLFunc func(scraper *sbScraper, doc *goquery.Selection) string

type parseTimeFunc func(s string) (time.Time, error)

type parserFunc func(doc *goquery.Selection, logger *zap.Logger, timeParser parseTimeFunc) ([]sbRecord, int, error)

func (a *App) runScrapers(ctx context.Context) {
	if a.config.ProxiesEnabled {
		a.pm.start(&a.config)

		defer a.pm.stop()

		for _, scraper := range a.scrapers {
			if errProxies := a.pm.setup(scraper.Collector, &a.config); errProxies != nil {
				a.log.Panic("Failed to setup proxies", zap.Error(errProxies))
			}
		}
	}

	waitGroup := &sync.WaitGroup{}

	for _, scraper := range a.scrapers {
		waitGroup.Add(1)

		go func(s *sbScraper) {
			defer waitGroup.Done()

			s.start(ctx, a.db)
		}(scraper)
	}

	waitGroup.Wait()
}

func (a *App) startScrapers(ctx context.Context) {
	const scraperInterval = time.Hour * 24
	scraperTicker := time.NewTicker(scraperInterval)
	trigger := make(chan any)

	go func() {
		trigger <- true
	}()

	for {
		select {
		case <-trigger:
			a.runScrapers(ctx)
		case <-scraperTicker.C:
			trigger <- true
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

func (r *sbRecord) setPlayer(name string) {
	if name == "" {
		return
	}

	r.Name = name
}

func (r *sbRecord) setInvokedOn(parseTime parseTimeFunc, value string) error {
	parsedTime, errTime := parseTime(value)
	if errTime != nil {
		return errTime
	}

	r.CreatedOn = parsedTime

	return nil
}

func (r *sbRecord) setBanLength(value string) {
	lowerVal := strings.ToLower(value)
	if strings.Contains(lowerVal, "unbanned") {
		r.SteamID = "" // invalidate it
	} else if lowerVal == "permanent" {
		r.Permanent = true
	}

	r.Length = 0
}

func (r *sbRecord) setExpiredOn(parseTime parseTimeFunc, value string) error {
	if r.Permanent || !r.SteamID.Valid() {
		// Ignore when
		return nil
	}

	parsedTime, errTime := parseTime(value)
	if errTime != nil {
		return errTime
	}

	r.Length = parsedTime.Sub(r.CreatedOn)

	if r.Length < 0 {
		// Some temp ban/actions use a negative duration?, just invalidate these
		r.SteamID = ""
	}

	return nil
}

func (r *sbRecord) setReason(value string) {
	if value == "" {
		return
	}

	r.Reason = value
}

func (r *sbRecord) setSteam(value string) {
	if r.SteamID.Valid() {
		return
	}

	r.SteamID = steamid.New(value)
}

type sbScraper struct {
	*colly.Collector
	name      string
	theme     string
	log       *zap.Logger
	curPage   int
	results   []sbRecord
	queue     *queue.Queue
	resultsMu sync.RWMutex
	baseURL   string
	sleepTime time.Duration
	startPath string
	parser    parserFunc
	nextURL   nextURLFunc
	parseTIme parseTimeFunc
}

func createScrapers(logger *zap.Logger, cacheDir string) ([]*sbScraper, error) {
	// scraperConstructors:= []func(logger *zap.Logger, cacheDir string) (*sbScraper, error){newProGamesZetScraper}.
	scraperConstructors := []func(logger *zap.Logger, cacheDir string) (*sbScraper, error){
		new7MauScraper, newAceKillScraper, newAMSGamingScraper, newApeModeScraper, newAstraManiaScraper,
		newBachuruServasScraper, newBaitedCommunityScraper, newBierwieseScraper, newBigBangGamersScraper,
		newBioCraftingScraper, newBouncyBallScraper, newCasualFunScraper, newCedaPugScraper, newCSIServersScraper,
		newCuteProjectScraper, newCutiePieScraper, newDarkPyroScraper, newDefuseRoScraper, newDiscFFScraper,
		newDreamFireScraper, newECJScraper, newElectricScraper, newEOTLGamingScraper, newEpicZoneScraper,
		newFirePoweredScraper, newFluxTFScraper, newFurryPoundScraper, newG44Scraper, newGameSitesScraper,
		newGamesTownScraper, newGetSomeScraper, newGFLScraper, newGhostCapScraper, newGlobalParadiseScraper,
		newGunServerScraper, newHarpoonScraper, newHellClanScraper, newJumpAcademyScraper, newLBGamingScraper,
		newLOOSScraper, newLazyNeerScraper, newLazyPurpleScraper, newLunarioScraper, newMagyarhnsScraper,
		newMaxDBScraper, newMoevsMachineScraper, newNeonHeightsScraper, newNideScraper, newOpstOnlineScraper,
		newOreonScraper, newOwlTFScraper, newPancakesScraper, newPandaScraper, newPetrolTFScraper,
		newPhoenixSourceScraper, newPlayesROScraper, newPowerFPSScraper, newProGamesZetScraper, newPRWHScraper,
		newPubsTFScraper, newRandomTF2Scraper, newRushyScraper, newRetroServersScraper, newSGGamingScraper,
		newSameTeemScraper, newSavageServidoresScraper, newScrapTFScraper, newServiliveClScraper, newSettiScraper,
		newSirPleaseScraper, newSkialScraper, newSlavonServerScraper, newSneaksScraper, newSpaceShipScraper,
		newSpectreScraper, newSvdosBrothersScraper, newSwapShopScraper, newTF2MapsScraper, newTF2ROScraper,
		/*newTawernaScraper,*/ newTheVilleScraper, newTitanScraper, newTriggerHappyScraper, newUGCScraper,
		newVaticanCityScraper, newVidyaGaemsScraper, newVortexScraper, /* newWonderlandTFGOOGScraper,*/
		newZMBrasilScraper, newZubatScraper,
	}

	var (
		scrapers  []*sbScraper
		scraperMu = &sync.RWMutex{}
		errGroup  = errgroup.Group{}
	)

	for _, scraperSetupFunc := range scraperConstructors {
		setupFn := scraperSetupFunc

		errGroup.Go(func() error {
			scraper, errScraper := setupFn(logger, cacheDir)
			if errScraper != nil {
				return errScraper
			}

			scraperMu.Lock()
			scrapers = append(scrapers, scraper)
			scraperMu.Unlock()

			return nil
		})
	}

	if errWait := errGroup.Wait(); errWait != nil {
		return nil, errors.Wrap(errWait, "Could not initialize all scrapers")
	}

	return scrapers, nil
}

func (scraper *sbScraper) start(ctx context.Context, database *pgStore) {
	scraper.log.Info("Starting scrape job", zap.String("name", scraper.name), zap.String("theme", scraper.theme))

	lastURL := ""
	startTime := time.Now()
	totalErrorCount := 0

	scraper.Collector.OnHTML("body", func(element *colly.HTMLElement) {
		results, errorCount, parseErr := scraper.parser(element.DOM, scraper.log, scraper.parseTIme)
		if parseErr != nil {
			scraper.log.Error("Parser returned error", zap.Error(parseErr))

			return
		}
		nextURL := scraper.nextURL(scraper, element.DOM)
		totalErrorCount += errorCount
		scraper.resultsMu.Lock()
		scraper.results = append(scraper.results, results...)
		scraper.resultsMu.Unlock()
		for _, result := range results {
			pRecord := newPlayerRecord(result.SteamID)
			if errPlayer := database.playerGetOrCreate(ctx, result.SteamID, &pRecord); errPlayer != nil {
				scraper.log.Error("failed to get player record", zap.Int64("sid64", result.SteamID.Int64()), zap.Error(errPlayer))

				continue
			}

			bRecord := models.SbBanRecord{
				BanID:       0,
				SiteName:    "",
				SiteID:      int(scraper.ID),
				PersonaName: result.Name,
				SteamID:     pRecord.SteamID,
				Reason:      result.Reason,
				Duration:    result.Length,
				Permanent:   result.Permanent,
				TimeStamped: models.TimeStamped{
					UpdatedOn: time.Now(),
					CreatedOn: result.CreatedOn,
				},
			}

			if errBanSave := database.sbBanSave(ctx, &bRecord); errBanSave != nil {
				if errors.Is(errBanSave, errDuplicate) {
					scraper.log.Debug("Failed to save ban record (duplicate)",
						zap.Int64("sid64", pRecord.SteamID.Int64()), zap.Error(errBanSave))

					continue
				}
				scraper.log.Error("Failed to save ban record",
					zap.Int64("sid64", pRecord.SteamID.Int64()), zap.Error(errBanSave))
			}
		}
		if nextURL != "" && nextURL != lastURL {
			lastURL = nextURL
			if scraper.sleepTime > 0 {
				time.Sleep(scraper.sleepTime)
			}
			scraper.log.Debug("Visiting next url", zap.String("url", nextURL))
			if errAdd := scraper.queue.AddURL(nextURL); errAdd != nil {
				scraper.log.Panic("Failed to add queue error", zap.Error(errAdd))
			}
		}
	})

	if errAdd := scraper.queue.AddURL(scraper.url(scraper.startPath)); errAdd != nil {
		scraper.log.Panic("Failed to add queue error", zap.Error(errAdd))
	}

	if errRun := scraper.queue.Run(scraper.Collector); errRun != nil {
		scraper.log.Error("Queue returned error", zap.Error(errRun))
	}

	scraper.log.Info("Completed scrape job", zap.String("name", scraper.name),
		zap.Int("valid", len(scraper.results)), zap.Int("skipped", totalErrorCount),
		zap.Duration("duration", time.Since(startTime)))
}

type scrapeLogger struct {
	logger *zap.Logger
	start  time.Time
}

func (log *scrapeLogger) Init() error {
	log.start = time.Now()

	return nil
}

func (log *scrapeLogger) Event(event *debug.Event) {
	args := []zap.Field{
		zap.Uint32("col_id", event.CollectorID),
		zap.Uint32("req_id", event.RequestID), zap.Duration("duration", time.Since(log.start)),
	}

	u, ok := event.Values["url"]
	if ok {
		args = append(args, zap.String("url", u))
	}

	switch event.Type {
	case "error":
		log.logger.Error("Error scraping url", args...)
	default:
		args = append(args, zap.String("type", event.Type))
		log.logger.Debug("Scraped url", args...)
	}
}

const defaultStartPath = "index.php?p=banlist"

//nolint:unparam
func newScraper(logger *zap.Logger, cacheDir string, name string, baseURL string, startPath string, parser parserFunc,
	nextURL nextURLFunc, parseTime parseTimeFunc,
) (*sbScraper, error) {
	const (
		randomDelay    = 5 * time.Second
		maxQueueSize   = 10000
		requestTimeout = time.Second * 30
	)

	parsedURL, errURL := url.Parse(baseURL)
	if errURL != nil {
		return nil, errors.Wrap(errURL, "Failed to parse base url")
	}

	log := logger.Named(name)
	debugLogger := scrapeLogger{logger: log} //nolint:exhaustruct

	reqQueue, errQueue := queue.New(1, &queue.InMemoryQueueStorage{MaxSize: maxQueueSize})
	if errQueue != nil {
		return nil, errors.Wrap(errQueue, "Filed to create queue")
	}

	if startPath == "" {
		startPath = defaultStartPath
	}

	scraper := sbScraper{ //nolint:exhaustruct
		baseURL:   baseURL,
		name:      name,
		theme:     "default",
		startPath: startPath,
		queue:     reqQueue,
		curPage:   1,
		parser:    parser,
		nextURL:   nextURL,
		parseTIme: parseTime,
		log:       log,
		Collector: colly.NewCollector(
			colly.UserAgent("bd"),
			colly.CacheDir(filepath.Join(cacheDir, "scrapers")),
			colly.Debugger(&debugLogger),
			colly.AllowedDomains(parsedURL.Hostname()),
		),
	}

	scraper.SetRequestTimeout(requestTimeout)
	scraper.OnRequest(func(r *colly.Request) {
		scraper.log.Debug("Visiting", zap.String("url", r.URL.String()))
	})
	extensions.RandomUserAgent(scraper.Collector)

	if errLimit := scraper.Limit(&colly.LimitRule{ //nolint:exhaustruct
		DomainGlob:  "*" + parsedURL.Hostname(),
		RandomDelay: randomDelay,
	}); errLimit != nil {
		scraper.log.Panic("Failed to set limit", zap.Error(errLimit))
	}

	scraper.OnError(func(r *colly.Response, err error) {
		scraper.log.Error("Request error", zap.String("url", r.Request.URL.String()), zap.Error(err))
	})

	return &scraper, nil
}

func (scraper *sbScraper) url(path string) string {
	return scraper.baseURL + path
}

func doTimeParse(layout string, timeStr string) (time.Time, error) {
	parsedTime, errParse := time.Parse(layout, timeStr)
	if errParse != nil {
		return time.Time{}, errors.Wrap(errParse, "Failed to parse time value")
	}

	return parsedTime, nil
}

// 05-17-23 03:07.
func parseSkialTime(timeStr string) (time.Time, error) {
	if timeStr == "Not applicable." || timeStr == "Permanent" {
		return time.Time{}, nil
	}

	return doTimeParse("01-02-06 15:04", timeStr)
}

// 05-17-23 03:07.
func parseRushyTime(timeStr string) (time.Time, error) {
	if timeStr == "Not applicable." || timeStr == "Permanent" {
		return time.Time{}, nil
	}

	return doTimeParse("15:04 pm 01/02/2006", timeStr)
}

// 17-05-23 03:07.
func parseBachuruServasTime(timeStr string) (time.Time, error) {
	if timeStr == "Not applicable." || timeStr == "Permanent" {
		return time.Time{}, nil
	}

	return doTimeParse("02-01-2006, 15:04", timeStr)
}

// 05-17-23 03:07.
func parseBaitedTime(timeStr string) (time.Time, error) {
	if timeStr == "Not applicable." || timeStr == "Permanent" {
		return time.Time{}, nil
	}

	return doTimeParse("02-01-2006 15:04", timeStr)
}

// 05-17-23 03:07.
func parseSkialAltTime(timeStr string) (time.Time, error) {
	if timeStr == "Not applicable." || timeStr == "Permanent" {
		return time.Time{}, nil
	}

	return doTimeParse("06-01-02 15:04", timeStr)
}

// 05-17-23 03:07.
func parseGunServer(timeStr string) (time.Time, error) {
	if timeStr == "Not applicable." || timeStr == "Permanent" {
		return time.Time{}, nil
	}

	return doTimeParse("02.01.2006 15:04", timeStr)
}

// 08.06.2023 в 21:21.
func parseProGamesZetTime(timeStr string) (time.Time, error) {
	if timeStr == "Not applicable." || timeStr == "Permanent" || timeStr == "Никогда." {
		return time.Time{}, nil
	}

	return doTimeParse("02.01.2006 в 15:04", timeStr)
}

func parsePRWHTime(timeStr string) (time.Time, error) {
	if timeStr == "Not applicable." || timeStr == "Permanent" || timeStr == "Никогда." {
		return time.Time{}, nil
	}

	return doTimeParse("02.01.06 15:04:05", timeStr)
}

// 17/05/23 - 03:07:05.
func parseSVDos(timeStr string) (time.Time, error) {
	if timeStr == "Not applicable." || timeStr == "Permanent" {
		return time.Time{}, nil
	}

	return doTimeParse("02/01/06 - 15:04:05", timeStr)
}

// 17/05/23 - 03:07:05.
func parseTriggerHappyTime(timeStr string) (time.Time, error) {
	if timeStr == "Not applicable." || timeStr == "Permanent" {
		return time.Time{}, nil
	}

	return doTimeParse("02/01/2006 15:04:05", timeStr)
}

// 17/05/23 03:07 PM.
func parseDarkPyroTime(timeStr string) (time.Time, error) {
	if timeStr == "Not applicable." || timeStr == "Permanent" {
		return time.Time{}, nil
	}

	return doTimeParse("01/02/06 15:04 PM", timeStr)
}

// 17-05-2023 03:07:05.
func parseTrailYear(timeStr string) (time.Time, error) {
	if timeStr == "Not applicable." || timeStr == "Permanent" {
		return time.Time{}, nil
	}

	return doTimeParse("02-01-2006 15:04:05", timeStr)
}

// 17-05-2023 03:07:05.
func parseHellClanTime(timeStr string) (time.Time, error) {
	if timeStr == "Not applicable." || timeStr == "Permanent" {
		return time.Time{}, nil
	}

	return doTimeParse("02-01-2006 15:04 MST", timeStr)
}

// 05-31-2023 9:57 PM CDT.
func parseSneakTime(timeStr string) (time.Time, error) {
	if timeStr == "Not applicable." || timeStr == "Permanent" {
		return time.Time{}, nil
	}

	return doTimeParse("01-02-2006 15:04 PM MST", timeStr)
}

// 24-06-2023 11:15:11 IST.
func parseAMSGamingTime(timeStr string) (time.Time, error) {
	if timeStr == "Not applicable." || timeStr == "Permanent" {
		return time.Time{}, nil
	}

	return doTimeParse("02-01-2006 15:04:05 MST", timeStr)
}

// 2023-05-17 03:07:05.
func parseDefaultTime(timeStr string) (time.Time, error) {
	if timeStr == "Not applicable." {
		return time.Time{}, nil
	}

	return doTimeParse("2006-01-02 15:04:05", timeStr)
}

// 2023-17-05 03:07:05  / 2023-26-05 10:56:53.
func parseDefaultTimeMonthFirst(timeStr string) (time.Time, error) {
	if timeStr == "Not applicable." {
		return time.Time{}, nil
	}

	return doTimeParse("2006-02-01 15:04:05", timeStr)
}

// Thu, May 11, 2023 7:14 PM    / Fri, Jun 2, 2023 6:40 PM.
func parsePancakesTime(timeStr string) (time.Time, error) {
	if timeStr == "Not applicable." || timeStr == "never, this is permanent" {
		return time.Time{}, nil
	}

	return doTimeParse("Mon, Jan 2, 2006 15:04 PM", timeStr)
}

// Thu, May 11, 2023 7:14 PM
// func parseOtakuTime(s string) (time.Time, error) {
//	if s == "Not applicable." || s == "never, this is permanent" {
//		return time.Time{}, nil
//	}
//	return time.Parse("Jan-2-2006 15:04:05", s)
//}

// Thu, May 11, 2023 7:14 PM.
func parseTitanTime(timeStr string) (time.Time, error) {
	if timeStr == "Not applicable." || timeStr == "never, this is permanent" {
		return time.Time{}, nil
	}

	return doTimeParse("Monday, 2 Jan 2006 15:04:05 PM", timeStr)
}

// May 11, 2023 7:14 PM.
func parseSGGamingTime(timeStr string) (time.Time, error) {
	if timeStr == "Not applicable." || timeStr == "never, this is permanent" {
		return time.Time{}, nil
	}

	return doTimeParse("Jan 02, 2006 15:04 PM", timeStr)
}

// May 11, 2023 7:14 PM   / June 7, 2022, 1:15 am.
func parseFurryPoundTime(timeStr string) (time.Time, error) {
	if timeStr == "Not applicable." || timeStr == "never, this is permanent" {
		return time.Time{}, nil
	}

	return doTimeParse("January _2, 2006, 15:04 pm", timeStr)
}

// Sunday 11th of May 2023 7:14:05 PM.
func parseFluxTime(timeStr string) (time.Time, error) {
	rx := regexp.MustCompile(`\s(\d+)(st|nd|rd|th)\s`)
	normTimeStr := rx.ReplaceAllString(timeStr, " $1 ")

	if timeStr == "Not applicable." || timeStr == "never, this is permanent" {
		return time.Time{}, nil
	}

	return doTimeParse("Monday _2 of January 2006 15:04:05 PM", normTimeStr)
}

// May 17th, 2023 (6:56).
func parseWonderlandTime(timeStr string) (time.Time, error) {
	if timeStr == "Not applicable." {
		return time.Time{}, nil
	}

	for _, k := range []string{"st", "nd", "rd", "th"} {
		timeStr = strings.ReplaceAll(timeStr, k, "")
	}

	return doTimeParse("January 2, 2006 (15:04)", timeStr)
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

	keyReason   mappedKey = "reason"
	keySteam3ID mappedKey = "steam3"
	keyPlayer   mappedKey = "player"
)

type normalizer struct {
	keyMap  map[string]mappedKey
	spaceRm *regexp.Regexp
}

func newNormalizer() *normalizer {
	return &normalizer{
		spaceRm: regexp.MustCompile(`\s+`),
		keyMap: map[string]mappedKey{
			"community links":      "community links",
			"banlanma tarihi":      "invoked on",
			"был выдан":            "invoked on",
			"datum a čas udělení":  "invoked on",
			"invoked on":           "invoked on",
			"steam community":      "steam community",
			"steam komunitní":      "steam community",
			"ban uzunluğu":         "ban length",
			"délka":                "ban length",
			"banlength":            "ban length",
			"ban length":           "ban length",
			"длительность":         "ban length",
			"şu zaman sona eriyor": "expires on",
			"vyprší":               "expires on",
			"будет снят":           "expires on",
			"expires on":           "expires on",
			"причина разбана":      "reason unbanned",
			"sebep":                "reason",
			"důvod":                "reason",
			"reason":               "reason",
			"разбанен админом":     "unbanned by",
			"причина бана":         "reason",
			"oyuncu":               "player",
			"игрок":                "player",
			"player":               "player",
			"steam3 id":            "steam3",
			"steam id":             "steam",
		},
	}
}

func (n normalizer) key(s string) string {
	return strings.ReplaceAll(n.spaceRm.ReplaceAllString(strings.TrimSpace(strings.ToLower(s)), " "), "\n", "")
}

func (n normalizer) getMappedKey(s string) (mappedKey, bool) {
	mk, found := n.keyMap[s]

	return mk, found
}

// https://github.com/SB-MaterialAdmin/Web/tree/stable-dev
func parseMaterial(doc *goquery.Selection, log *zap.Logger, parseTime parseTimeFunc) ([]sbRecord, int, error) {
	var (
		bans      []sbRecord
		curBan    sbRecord
		skipCount int
	)

	norm := newNormalizer()

	doc.Find("div.opener .card-body").Each(func(_ int, selection *goquery.Selection) {
		selection.First().Children().Children().Each(func(i int, selection *goquery.Selection) {
			children := selection.First().Children()
			first := children.First()
			second := children.Last()
			key := norm.key(first.Contents().Text())
			value := strings.TrimSpace(second.Contents().Text())
			mk, ok := norm.getMappedKey(key)
			if !ok {
				return
			}
			switch mk {
			case keyPlayer:
				curBan.setPlayer(value)
			case keySteam3ID:
				curBan.setSteam(value)
			case keyCommunityLinks:
				nv, foundKey := second.Children().First().Attr("href")
				if !foundKey {
					return
				}
				pcs := strings.Split(nv, "/")
				curBan.setSteam(pcs[4])
			case keySteamCommunity:
				pts := strings.Split(value, " ")
				curBan.setSteam(pts[0])
			case keyInvokedOn:
				if errInvoke := curBan.setInvokedOn(parseTime, value); errInvoke != nil {
					log.Error("failed to set invoke time", zap.String("input", value), zap.Error(errInvoke))
				}
			case keyBanLength:
				curBan.setBanLength(value)
			case keyExpiredOn:
				if errExpiration := curBan.setExpiredOn(parseTime, value); errExpiration != nil {
					log.Error("failed to set expiration time", zap.String("input", value), zap.Error(errExpiration))
				}
			case keyReason:
				curBan.setReason(value)
				curBan.Reason = value
				if curBan.SteamID.Valid() && curBan.Name != "" {
					bans = append(bans, curBan)
				} else {
					skipCount++
				}
				curBan = sbRecord{} //nolint:exhaustruct
			}
		})
	})

	return bans, skipCount, nil
}

// https://github.com/brhndursun/SourceBans-StarTheme
func parseStar(doc *goquery.Selection, log *zap.Logger, parseTime parseTimeFunc) ([]sbRecord, int, error) {
	const expectedNodes = 3

	var (
		bans      []sbRecord
		curBan    sbRecord
		skipCount int
	)

	norm := newNormalizer()

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
			if len(children.Nodes) == expectedNodes {
				curBan.Name = strings.TrimSpace(children.First().Next().Text())

				return
			}
			first := children.First()
			second := children.Last()
			key := norm.key(first.Contents().Text())
			value := strings.TrimSpace(second.Contents().Text())
			mk, found := norm.getMappedKey(key)
			if !found {
				return
			}
			switch mk {
			case keyPlayer:
				curBan.setPlayer(value)
			case keyCommunityLinks:
				nv, foundHREF := second.Children().First().Attr("href")
				if !foundHREF {
					return
				}
				pcs := strings.Split(nv, "/")
				curBan.setSteam(pcs[4])
			case keySteam3ID:
				curBan.setSteam(value)
			case keySteamCommunity:
				pts := strings.Split(value, " ")
				curBan.setSteam(pts[0])
			case keyInvokedOn:
				if errInvoke := curBan.setInvokedOn(parseTime, value); errInvoke != nil {
					log.Error("failed to set invoke time", zap.String("input", value), zap.Error(errInvoke))
				}
			case keyBanLength:
				curBan.setBanLength(value)
			case keyExpiredOn:
				if errExpiration := curBan.setExpiredOn(parseTime, value); errExpiration != nil {
					log.Error("failed to set expiration time", zap.String("input", value), zap.Error(errExpiration))
				}

			case keyReason:
				curBan.setReason(value)
				if curBan.SteamID.Valid() && curBan.Name != "" {
					bans = append(bans, curBan)
				} else {
					skipCount++
				}
				curBan = sbRecord{} //nolint:exhaustruct
			}
		})
	})

	return bans, skipCount, nil
}

// https://github.com/aXenDeveloper/sourcebans-web-theme-fluent
func parseFluent(doc *goquery.Selection, log *zap.Logger, parseTime parseTimeFunc) ([]sbRecord, int, error) {
	var (
		bans      []sbRecord
		curBan    sbRecord
		skipCount int
	)

	norm := newNormalizer()

	doc.Find("ul.ban_list_detal li").Each(func(i int, selection *goquery.Selection) {
		child := selection.Children()
		key := norm.key(child.First().Contents().Text())
		value := strings.TrimSpace(child.Last().Contents().Text())
		mk, found := norm.getMappedKey(key)
		if !found {
			return
		}
		switch mk { //nolint:exhaustive
		case keyPlayer:
			curBan.setPlayer(value)
		case keySteam3ID:
			curBan.setSteam(value)
		case keySteamCommunity:
			pts := strings.Split(value, " ")
			curBan.setSteam(pts[0])
		case keyInvokedOn:
			if errInvoke := curBan.setInvokedOn(parseTime, value); errInvoke != nil {
				log.Error("failed to set invoke time", zap.String("input", value), zap.Error(errInvoke))
			}
		case keyBanLength:
			curBan.setBanLength(value)
		case keyExpiredOn:
			if errExpiration := curBan.setExpiredOn(parseTime, value); errExpiration != nil {
				log.Error("failed to set expiration time", zap.String("input", value), zap.Error(errExpiration))
			}
		case keyReason:
			curBan.setReason(value)
			if curBan.SteamID.Valid() && curBan.Name != "" {
				bans = append(bans, curBan)
			} else {
				skipCount++
			}
			curBan = sbRecord{} //nolint:exhaustruct
		}
	})

	return bans, skipCount, nil
}

func parseDefault(doc *goquery.Selection, log *zap.Logger, parseTime parseTimeFunc) ([]sbRecord, int, error) {
	var (
		bans     []sbRecord
		curBan   sbRecord
		curState mappedKey
		isValue  bool
		skipped  int
	)

	norm := newNormalizer()

	doc.Find("#banlist .listtable table tr td").Each(func(i int, selection *goquery.Selection) {
		value := strings.TrimSpace(selection.Text())

		if !isValue {
			key := norm.key(value)
			mk, found := norm.getMappedKey(key)
			if !found {
				return
			}
			switch mk { //nolint:exhaustive
			case keyPlayer:
				curState = keyPlayer
				isValue = true
			case keySteamCommunity:
				curState = keySteamCommunity
				isValue = true
			case keySteam3ID:
				curState = keySteam3ID
				isValue = true
			case keyInvokedOn:
				curState = keyInvokedOn
				isValue = true
			case keyBanLength:
				curState = keyBanLength
				isValue = true
			case keyExpiredOn:
				curState = keyExpiredOn
				isValue = true
			case keyReason:
				curState = keyReason
				isValue = true
			}

			return
		}

		isValue = false
		switch curState { //nolint:exhaustive
		case keyPlayer:
			curBan.setPlayer(value)
		// case keySteam3ID:
		//	if errSteam := curBan.setSteam(value); errSteam != nil {
		//		log.Debug("Failed to set steam (steam3 comm)", zap.String("input", value), zap.Error(errSteam))
		//	}
		// case keySteamID:
		//	if errSteam := curBan.setSteam(value); errSteam != nil {
		//		log.Debug("Failed to set steam (steam)", zap.String("input", value), zap.Error(errSteam))
		//	}
		case keySteamCommunity:
			pts := strings.Split(value, " ")
			curBan.setSteam(pts[0])
		case keyInvokedOn:
			if errInvoke := curBan.setInvokedOn(parseTime, value); errInvoke != nil {
				log.Error("failed to set invoke time", zap.String("input", value), zap.Error(errInvoke))
			}
		case keyBanLength:
			curBan.setBanLength(value)
		case keyExpiredOn:
			if errExpiration := curBan.setExpiredOn(parseTime, value); errExpiration != nil {
				log.Error("failed to set expiration time", zap.String("input", value), zap.Error(errExpiration))
			}
		case keyReason:
			curBan.setReason(value)
			if curBan.SteamID.Valid() && curBan.Name != "" {
				bans = append(bans, curBan)
			} else {
				skipped++
			}
			curBan = sbRecord{} //nolint:exhaustruct
		}
	})

	return bans, skipped, nil
}

//
// type megaScatterNode struct {
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
// func parseMegaScatter(bodyReader io.Reader) ([]sbRecord, error) {
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

type cfResult struct {
	page int
	body string
}

func crawlCloudflare(ctx context.Context, baseURL string, pages int, results chan cfResult) error {
	const (
		slowTimeout = time.Second * 5
	)

	var (
		userMode = launcher.
				NewUserMode().
				Leakless(true).
				Headless(true).
				UserDataDir("cache/t"). // *must* be this?
				Set("disable-default-apps").
				Set("no-first-run").
				MustLaunch()
		browser = rod.
			New().
			SlowMotion(slowTimeout).
			Context(ctx).
			ControlURL(userMode).
			MustConnect().
			NoDefaultDevice()
		page    *rod.Page
		curPage = 1
	)

	for curPage <= pages {
		URL := fmt.Sprintf(baseURL, curPage)
		if page == nil {
			page = browser.MustPage(URL)
		} else {
			page = page.MustNavigate(URL)
		}

		page.MustWaitLoad()

		body, errBody := page.HTML()
		if errBody != nil {
			return errors.Wrap(errBody, "Failed to read HTML body")
		}

		results <- cfResult{page: curPage, body: body}

		curPage++
	}

	return nil
}
