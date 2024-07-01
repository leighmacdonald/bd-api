package main

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net/http"
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
	"github.com/leighmacdonald/bd-api/domain"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"golang.org/x/sync/errgroup"
)

var (
	errScrapInit          = errors.New("failed to initialize all scrapers")
	errScrapeURL          = errors.New("failed to parse scraper URL")
	errScrapeQueueInit    = errors.New("failed to initialize scraper queue")
	errScrapeLimit        = errors.New("failed to set scraper limit")
	errScrapeLauncherInit = errors.New("failed to setup browser launcher")
	errScrapeWait         = errors.New("failed to wait for content load")
	errScrapeCFOpen       = errors.New("could not open cloudflare transport")
	errScrapeParseTime    = errors.New("failed to parse time value")
)

type nextURLFunc func(scraper *sbScraper, doc *goquery.Selection) string

type parseTimeFunc func(s string) (time.Time, error)

type parserFunc func(doc *goquery.Selection, log *slog.Logger, timeParser parseTimeFunc) ([]sbRecord, int, error)

func initScrapers(ctx context.Context, database *pgStore, cacheDir string) ([]*sbScraper, error) {
	scrapers, errScrapers := createScrapers(cacheDir)
	if errScrapers != nil {
		return nil, errScrapers
	}

	for _, scraper := range scrapers {
		// Attach a site_id to the scraper, so we can keep track of the scrape source
		var s domain.SbSite
		if errSave := database.sbSiteGetOrCreate(ctx, scraper.name, &s); errSave != nil {
			return nil, errSave
		}

		scraper.ID = uint32(s.SiteID)
	}

	return scrapers, nil
}

func runScrapers(ctx context.Context, database *pgStore, scrapers []*sbScraper) {
	waitGroup := &sync.WaitGroup{}

	for _, scraper := range scrapers {
		waitGroup.Add(1)

		go func(s *sbScraper) {
			defer waitGroup.Done()

			s.start(ctx, database)
		}(scraper)
	}

	waitGroup.Wait()
}

func startScrapers(ctx context.Context, database *pgStore, scrapers []*sbScraper) {
	const scraperInterval = time.Hour * 24
	scraperTicker := time.NewTicker(scraperInterval)

	sync.OnceFunc(func() {
		runScrapers(ctx, database, scrapers)
	})()

	for {
		select {
		case <-scraperTicker.C:
			runScrapers(ctx, database, scrapers)
		case <-ctx.Done():
			return
		}
	}
}

type sbRecord struct {
	Name      string
	SteamID   steamid.SteamID
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
		r.SteamID = steamid.SteamID{} // invalidate it
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
		r.SteamID = steamid.SteamID{}
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
	name      domain.Site
	theme     string
	log       *slog.Logger
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

func createScrapers(cacheDir string) ([]*sbScraper, error) {
	// scraperConstructors:= []func(cacheDir string) (*sbScraper, error){newProGamesZetScraper}.
	scraperConstructors := []func(cacheDir string) (*sbScraper, error){
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
		newVaticanCityScraper, newVidyaGaemsScraper, newVortexScraper, /*newWonderlandTFScraper*/
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
			scraper, errScraper := setupFn(cacheDir)
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
		return nil, errors.Join(errWait, errScrapInit)
	}

	return scrapers, nil
}

func (scraper *sbScraper) start(ctx context.Context, database *pgStore) {
	slog.Info("Starting scrape job",
		slog.String("name", string(scraper.name)), slog.String("theme", scraper.theme))

	lastURL := ""
	startTime := time.Now()
	totalErrorCount := 0

	scraper.Collector.OnHTML("body", func(element *colly.HTMLElement) {
		results, errorCount, parseErr := scraper.parser(element.DOM, scraper.log, scraper.parseTIme)
		if parseErr != nil {
			slog.Error("Parser returned error", ErrAttr(parseErr))

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
				slog.Error("failed to get player record", slog.String("sid64", result.SteamID.String()), ErrAttr(errPlayer))

				continue
			}

			bRecord := domain.SbBanRecord{
				BanID:       0,
				SiteName:    "",
				SiteID:      int(scraper.ID),
				PersonaName: result.Name,
				SteamID:     pRecord.SteamID,
				Reason:      result.Reason,
				Duration:    result.Length,
				Permanent:   result.Permanent,
				TimeStamped: domain.TimeStamped{
					UpdatedOn: time.Now(),
					CreatedOn: result.CreatedOn,
				},
			}

			if errBanSave := database.sbBanSave(ctx, &bRecord); errBanSave != nil {
				if errors.Is(errBanSave, errDatabaseUnique) {
					// slog.Debug("Failed to save ban record (duplicate)",
					//	slog.String("sid64", pRecord.SteamID.String()), ErrAttr(errBanSave))

					continue
				}
				slog.Error("Failed to save ban record",
					slog.String("sid64", pRecord.SteamID.String()), ErrAttr(errBanSave))
			}
		}
		if nextURL != "" && nextURL != lastURL {
			lastURL = nextURL
			if scraper.sleepTime > 0 {
				time.Sleep(scraper.sleepTime)
			}
			slog.Debug("Visiting next url", slog.String("url", nextURL))
			if errAdd := scraper.queue.AddURL(nextURL); errAdd != nil {
				slog.Error("Failed to add queue error", ErrAttr(errAdd))

				return
			}
		}
	})

	if errAdd := scraper.queue.AddURL(scraper.url(scraper.startPath)); errAdd != nil {
		slog.Error("Failed to add queue error", ErrAttr(errAdd))

		return
	}

	if errRun := scraper.queue.Run(scraper.Collector); errRun != nil {
		slog.Error("Queue returned error", ErrAttr(errRun))

		return
	}

	slog.Info("Completed scrape job", slog.String("name", string(scraper.name)),
		slog.Int("valid", len(scraper.results)), slog.Int("skipped", totalErrorCount),
		slog.Duration("duration", time.Since(startTime)))
}

type scrapeLogger struct {
	logger *slog.Logger
	start  time.Time
}

func (log *scrapeLogger) Init() error {
	log.start = time.Now()

	return nil
}

func (log *scrapeLogger) Event(event *debug.Event) {
	args := []any{
		slog.Uint64("col_id", uint64(event.CollectorID)),
		slog.Uint64("req_id", uint64(event.RequestID)),
		slog.Duration("duration", time.Since(log.start)),
	}

	u, ok := event.Values["url"]
	if ok {
		args = append(args, slog.String("url", u))
	}

	switch event.Type {
	case "error":
		log.logger.Error("Error scraping url", args...)
	default:
		args = append(args, slog.String("type", event.Type))
		log.logger.Debug("Scraped url", args...)
	}
}

const defaultStartPath = "index.php?p=banlist"

func newScraperWithTransport(cacheDir string, name domain.Site,
	baseURL string, startPath string, parser parserFunc, nextURL nextURLFunc, parseTime parseTimeFunc,
	transport http.RoundTripper,
) (*sbScraper, error) {
	scraper, errScraper := newScraper(cacheDir, name, baseURL, startPath, parser, nextURL, parseTime)
	if errScraper != nil {
		return nil, errScraper
	}

	scraper.Collector.WithTransport(transport)

	return scraper, nil
}

const (
	randomDelay    = 5 * time.Second
	maxQueueSize   = 10000000
	requestTimeout = time.Second * 30
)

func newScraper(cacheDir string, name domain.Site, baseURL string,
	startPath string, parser parserFunc, nextURL nextURLFunc, parseTime parseTimeFunc,
) (*sbScraper, error) {
	parsedURL, errURL := url.Parse(baseURL)
	if errURL != nil {
		return nil, errors.Join(errURL, errScrapeURL)
	}

	logger := slog.With("name", string(name))
	debugLogger := scrapeLogger{logger: logger} //nolint:exhaustruct

	reqQueue, errQueue := queue.New(1, &queue.InMemoryQueueStorage{MaxSize: maxQueueSize})
	if errQueue != nil {
		return nil, errors.Join(errQueue, errScrapeQueueInit)
	}

	if startPath == "" {
		startPath = defaultStartPath
	}

	collector := colly.NewCollector(
		colly.UserAgent("bd-api"),
		colly.CacheDir(filepath.Join(cacheDir, "scrapers")),
		colly.Debugger(&debugLogger),
		colly.AllowedDomains(parsedURL.Hostname()),
	)

	scraper := sbScraper{ //nolint:exhaustruct
		baseURL:   baseURL,
		name:      name,
		theme:     "default",
		startPath: startPath,
		queue:     reqQueue,
		curPage:   1,
		log:       logger,
		parser:    parser,
		nextURL:   nextURL,
		parseTIme: parseTime,
		Collector: collector,
	}

	scraper.SetRequestTimeout(requestTimeout)
	scraper.OnRequest(func(r *colly.Request) {
		slog.Debug("Visiting", slog.String("url", r.URL.String()))
	})
	extensions.RandomUserAgent(scraper.Collector)

	if errLimit := scraper.Limit(&colly.LimitRule{ //nolint:exhaustruct
		DomainGlob:  "*" + parsedURL.Hostname(),
		RandomDelay: randomDelay,
	}); errLimit != nil {
		return nil, errors.Join(errLimit, errScrapeLimit)
	}

	scraper.OnError(func(r *colly.Response, err error) {
		slog.Error("Request error", slog.String("url", r.Request.URL.String()), ErrAttr(err))
	})

	return &scraper, nil
}

func (scraper *sbScraper) url(path string) string {
	return scraper.baseURL + path
}

func doTimeParse(layout string, timeStr string) (time.Time, error) {
	parsedTime, errParse := time.Parse(layout, timeStr)
	if errParse != nil {
		return time.Time{}, errors.Join(errParse, errScrapeParseTime)
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
	nodes.EachWithBreak(func(_ int, selection *goquery.Selection) bool {
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
func parseMaterial(doc *goquery.Selection, log *slog.Logger, parseTime parseTimeFunc) ([]sbRecord, int, error) {
	var (
		bans      []sbRecord
		curBan    sbRecord
		skipCount int
	)

	norm := newNormalizer()

	doc.Find("div.opener .card-body").Each(func(_ int, selection *goquery.Selection) {
		selection.First().Children().Children().Each(func(_ int, selection *goquery.Selection) {
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
					log.Error("failed to set invoke time", slog.String("input", value), ErrAttr(errInvoke))
				}
			case keyBanLength:
				curBan.setBanLength(value)
			case keyExpiredOn:
				if errExpiration := curBan.setExpiredOn(parseTime, value); errExpiration != nil {
					log.Error("failed to set expiration time", slog.String("input", value), ErrAttr(errExpiration))
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
func parseStar(doc *goquery.Selection, log *slog.Logger, parseTime parseTimeFunc) ([]sbRecord, int, error) {
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
					log.Error("failed to set invoke time", slog.String("input", value), ErrAttr(errInvoke))
				}
			case keyBanLength:
				curBan.setBanLength(value)
			case keyExpiredOn:
				if errExpiration := curBan.setExpiredOn(parseTime, value); errExpiration != nil {
					log.Error("failed to set expiration time", slog.String("input", value), ErrAttr(errExpiration))
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
func parseFluent(doc *goquery.Selection, log *slog.Logger, parseTime parseTimeFunc) ([]sbRecord, int, error) {
	var (
		bans      []sbRecord
		curBan    sbRecord
		skipCount int
	)

	norm := newNormalizer()

	doc.Find("ul.ban_list_detal li").Each(func(_ int, selection *goquery.Selection) {
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
				log.Error("failed to set invoke time", slog.String("input", value), ErrAttr(errInvoke))
			}
		case keyBanLength:
			curBan.setBanLength(value)
		case keyExpiredOn:
			if errExpiration := curBan.setExpiredOn(parseTime, value); errExpiration != nil {
				log.Error("failed to set expiration time", slog.String("input", value), ErrAttr(errExpiration))
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

func parseDefault(doc *goquery.Selection, log *slog.Logger, parseTime parseTimeFunc) ([]sbRecord, int, error) {
	var (
		bans     []sbRecord
		curBan   sbRecord
		curState mappedKey
		isValue  bool
		skipped  int
	)

	norm := newNormalizer()

	doc.Find("#banlist .listtable table tr td").Each(func(_ int, selection *goquery.Selection) {
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
				log.Error("failed to set invoke time", slog.String("input", value), ErrAttr(errInvoke))
			}
		case keyBanLength:
			curBan.setBanLength(value)
		case keyExpiredOn:
			if errExpiration := curBan.setExpiredOn(parseTime, value); errExpiration != nil {
				log.Error("failed to set expiration time", slog.String("input", value), ErrAttr(errExpiration))
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
	page string
	body string
}

type cfTransport struct {
	launchURL      string
	browser        *rod.Browser
	page           *rod.Page
	waitStableTime time.Duration
	similarity     float32
}

func newCFTransport() *cfTransport {
	const (
		stableWaitTimeout = time.Second * 5
		pageSimilarity    = 0.5
	)

	return &cfTransport{waitStableTime: stableWaitTimeout, similarity: pageSimilarity} //nolint:exhaustruct
}

func (t *cfTransport) Open(ctx context.Context) error {
	const slowTimeout = time.Second * 5

	launchURL, errLauncher := launcher.
		NewUserMode().
		Leakless(true).
		Headless(false).
		UserDataDir("cache/t"). // *must* be this?
		Set("disable-default-apps").
		Set("no-first-run").
		Launch()
	if errLauncher != nil {
		return errors.Join(errLauncher, errScrapeLauncherInit)
	}

	t.launchURL = launchURL
	t.browser = rod.
		New().
		SlowMotion(slowTimeout).
		Context(ctx).
		ControlURL(t.launchURL).
		MustConnect().
		NoDefaultDevice()

	return nil
}

// NopCloser exists to satisfy io.ReadCloser interface for our fake http.Response.
type NopCloser struct {
	*bytes.Reader
}

func (b NopCloser) Close() error {
	return nil
}

// RoundTrip implements the minimal http.Transport interface so that the underlying browser used for
// scraping cloudflare protected sites can be integrated into the colly.Collector pipeline.
func (t *cfTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	reqURL := req.URL.String()

	res, errReq := t.fetch(reqURL)
	if errReq != nil {
		return nil, errReq
	}

	body := NopCloser{Reader: bytes.NewReader([]byte(res.body))}
	resp := &http.Response{ //nolint:exhaustruct
		Request:    req,
		Body:       body,
		StatusCode: http.StatusOK,
		Status:     http.StatusText(http.StatusOK),
		Proto:      "HTTP/1.0",
	}

	return resp, nil
}

func (t *cfTransport) fetch(url string) (*cfResult, error) {
	if t.page == nil {
		t.page = t.browser.MustPage(url)
	} else {
		t.page = t.page.MustNavigate(url)
	}

	if errWait := t.page.WaitStable(t.waitStableTime); errWait != nil {
		return nil, errors.Join(errWait, errScrapeWait)
	}

	body, errBody := t.page.HTML()
	if errBody != nil {
		return nil, errors.Join(errBody, errResponseRead)
	}

	return &cfResult{page: url, body: body}, nil
}
