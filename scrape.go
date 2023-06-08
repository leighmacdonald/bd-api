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

type nextURLFunc func(doc *goquery.Selection) string

type parseTimeFunc func(s string) (time.Time, error)

type parserFunc func(doc *goquery.Selection, nextUrl nextURLFunc, timeParser parseTimeFunc) (string, []sbRecord, error)

func startScraper(config *appConfig) {
	scrapers := []*sbScraper{
		new7MauScraper(),
		newApeModeScraper(),
		newAstraManiaScraper(),
		newBouncyBallScraper(),
		newCSIServersScraper(),
		newCutiePieScraper(),
		newDarkPyroScraper(),
		newDefuseRoScraper(),
		newDiscFFScraper(),
		newDreamFireScraper(),
		newECJScraper(),
		newElectricScraper(),
		newFirePoweredScraper(),
		newFluxTFScraper(),
		newFurryPoundScraper(),
		newGFLScraper(),
		newGhostCapScraper(),
		newGlobalParadiseScraper(),
		newGunServerScraper(),
		newHarpoonScraper(),
		newHellClanScraper(),
		newJumpAcademyScraper(),
		newLBGamingScraper(),
		newLOOSScraper(),
		newLazyNeerScraper(),
		newLazyPurpleScraper(),
		newMaxDBScraper(),
		newNeonHeightsScraper(),
		newNideScraper(),
		newOpstOnlineScraper(),
		newOreonScraper(),
		newOwlTFScraper(),
		newPancakesScraper(),
		newPandaScraper(),
		newPowerFPSScraper(),
		newPubsTFScraper(),
		newRetroServersScraper(),
		newSGGamingScraper(),
		newSameTeemScraper(),
		newSavageServidoresScraper(),
		newScrapTFScraper(),
		newServiliveClScraper(),
		newSettiScraper(),
		newSirPleaseScraper(),
		newSkialScraper(),
		newSneaksScraper(),
		newSpaceShipScraper(),
		newSpectreScraper(),
		newSvdosBrothersScraper(),
		newSwapShopScraper(),
		newTF2MapsScraper(),
		newTF2ROScraper(),
		newTawernaScraper(),
		newTheVilleScraper(),
		newTitanScraper(),
		newTriggerHappyScraper(),
		newUGCScraper(),
		newVaticanCityScraper(),
		newVidyaGaemsScraper(),
		newWonderlandTFScraper(),
		newZMBrasilScraper(),
	}
	startProxies(config)
	defer stopProxies()

	for _, scraper := range scrapers {
		if errProxies := setupProxies(scraper.Collector, config); errProxies != nil {
			logger.Panic("Failed to setup proxies", zap.Error(errProxies))
		}
		go func(s *sbScraper) {
			if errScrape := s.start(); errScrape != nil {
				logger.Error("sbScraper returned error", zap.Error(errScrape))
			}
		}(scraper)
	}
}

type metaKey int

const (
	unknown metaKey = iota
	player
	steamID
	steam2
	steamComm
	invokedOn
	banLength // "Permanent"
	expiresOn // "Not applicable."
	reason
	last
)

type sbRecord struct {
	Name      string
	SteamID   steamid.SID64
	Reason    string
	CreatedOn time.Time
	Length    time.Duration
	Permanent bool
}

type sbScraper struct {
	*colly.Collector
	name      string
	log       *zap.Logger
	results   []sbRecord
	resultsMu sync.RWMutex
	baseURL   string
	startPath string
	parser    parserFunc
	nextURL   nextURLFunc
	parseTIme parseTimeFunc
}

func (scraper *sbScraper) start() error {
	scraper.Collector.OnHTML("*", func(e *colly.HTMLElement) {
		nextURL, results, parseErr := parseDefault(e.DOM, scraper.nextURL, scraper.parseTIme)
		if parseErr != nil {
			logger.Error("Parser returned error", zap.Error(parseErr))
			return
		}
		scraper.resultsMu.Lock()
		scraper.results = append(scraper.results, results...)
		scraper.resultsMu.Unlock()
		if nextURL != "" {
			if errVisit := scraper.Visit(scraper.url(nextURL)); errVisit != nil {
				logger.Error("Failed to visit sub url", zap.Error(errVisit), zap.String("url", nextURL))
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

func newScraper(name string, baseURL string, startPath string, parser parserFunc, nextURL nextURLFunc, parseTime parseTimeFunc) *sbScraper {
	u, errURL := url.Parse(baseURL)
	if errURL != nil {
		logger.Panic("Failed to parse base url", zap.Error(errURL))
	}

	scraper := sbScraper{
		baseURL:   baseURL,
		name:      name,
		startPath: startPath,
		parser:    parser,
		nextURL:   nextURL,
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

	scraper.OnError(func(r *colly.Response, err error) {
		logger.Error("Request error", zap.String("url", r.Request.URL.String()), zap.Error(err))
	})

	return &scraper
}

func (scraper *sbScraper) url(path string) string {
	return scraper.baseURL + path
}

func newSkialScraper() *sbScraper {
	return newScraper(
		"skial",
		"https://www.skial.com/sourcebans/",
		"index.php?p=banlist",
		parseDefault,
		nextURLFirst,
		parseSkialTime,
	)
}

func newGFLScraper() *sbScraper {
	return newScraper("gfl", "https://sourcebans.gflclan.com/", "index.php?p=banlist",
		parseDefault, nextURLLast, parseDefaultTime)
}

func newSpaceShipScraper() *sbScraper {
	return newScraper("spaceship", "https://sappho.io/bans/", "index.php?p=banlist",
		parseDefault, nextURLLast, parseDefaultTime)
}

func newUGCScraper() *sbScraper {
	return newScraper("ugc", "https://sb.ugc-gaming.net/", "index.php?p=banlist",
		parseFluent, nextURLFluent, parseDefaultTime)
}

func newSirPleaseScraper() *sbScraper {
	return newScraper("sirplease", "https://sirplease.gg/", "index.php?p=banlist",
		parseFluent, nextURLFluent, parseDefaultTime)
}

func newVidyaGaemsScraper() *sbScraper {
	return newScraper("vidyagaems", "https://www.vidyagaems.net/", "index.php?p=banlist",
		parseFluent, nextURLFluent, parseTrailYear)
}

func newOwlTFScraper() *sbScraper {
	return newScraper("owl", "https://kingpandagamer.xyz/sb/", "index.php?p=banlist",
		parseDefault, nextURLLast, parseDefaultTime)
}

func newZMBrasilScraper() *sbScraper {
	return newScraper("zmbrasil", "http://bans.zmbrasil.com.br/", "index.php?p=banlist",
		parseDefault, nextURLLast, parseSkialTime)
}

func newScrapTFScraper() *sbScraper {
	return newScraper("scraptf", "https://bans.scrap.tf/", "index.php?p=banlist",
		parseDefault, nextURLLast, parseDefaultTime)
}

func newWonderlandTFScraper() *sbScraper {
	return newScraper("wonderland", "https://bans.wonderland.tf/", "index.php?p=banlist",
		parseDefault, nextURLLast, parseWonderlandTime)
}

func newLazyPurpleScraper() *sbScraper {
	return newScraper("lazypurple", "https://www.lazypurple.com/sourcebans/", "index.php?p=banlist",
		parseDefault, nextURLLast, parseDefaultTime)
}

func newFirePoweredScraper() *sbScraper {
	return newScraper("firepowered", "https://firepoweredgaming.com/sourcebanspp/", "index.php?p=banlist",
		parseDefault, nextURLLast, parseSkialTime)
}

func newHarpoonScraper() *sbScraper {
	return newScraper("harpoongaming", "https://bans.harpoongaming.com/", "index.php?p=banlist",
		parseDefault, nextURLLast, parseDefaultTime)
}

func newPandaScraper() *sbScraper {
	return newScraper("panda", "https://bans.panda-community.com/", "index.php?p=banlist",
		parseFluent, nextURLFluent, parseDefaultTime)
}

func newNeonHeightsScraper() *sbScraper {
	return newScraper("neonheights", "https://neonheights.xyz/bans/", "index.php?p=banlist",
		parseDefault, nextURLLast, parseSkialTime)
}

func newPancakesScraper() *sbScraper {
	return newScraper("pancakes", "https://pancakes.tf/", "index.php?p=banlist",
		parseDefault, nextURLLast, parsePancakesTime)
}

func newLOOSScraper() *sbScraper {
	return newScraper("loos", "https://looscommunity.com/bans/", "index.php?p=banlist",
		parseDefault, nextURLLast, parseDefaultTime)
}

func newPubsTFScraper() *sbScraper {
	return newScraper("pubstf", "https://bans.pubs.tf/", "index.php?p=banlist",
		parseDefault, nextURLLast, parseSkialTime)
}

func newServiliveClScraper() *sbScraper {
	return newScraper("servilivecl", "https://sourcebans.servilive.cl/", "index.php?p=banlist",
		parseFluent, nextURLFluent, parseDefaultTimeMonthFirst)
}

func newCutiePieScraper() *sbScraper {
	return newScraper("cutiepie", "https://bans.cutiepie.tf/", "index.php?p=banlist",
		parseDefault, nextURLLast, parseDefaultTime)
}
func newSGGamingScraper() *sbScraper {
	return newScraper("sggaming", "https://sg-gaming.net/bans/", "index.php?p=banlist",
		parseDefault, nextURLLast, parseSGGamingTime)
}

func newApeModeScraper() *sbScraper {
	return newScraper("apemode", "https://sourcebans.apemode.tf/", "index.php?p=banlist",
		parseDefault, nextURLLast, parseSkialTime)
}

func newMaxDBScraper() *sbScraper {
	return newScraper("maxdb", "https://bans.maxdb.net/", "index.php?p=banlist",
		parseDefault, nextURLLast, parseDefaultTime)
}

func newSvdosBrothersScraper() *sbScraper {
	return newScraper("svdosbrothers", "https://bans.svdosbrothers.com/", "index.php?p=banlist",
		parseFluent, nextURLFluent, parseSVDos)
}

func newElectricScraper() *sbScraper {
	return newScraper("electric", "http://168.181.184.179/", "index.php?p=banlist",
		parseFluent, nextURLFluent, parseDefaultTime)
}

func newGlobalParadiseScraper() *sbScraper {
	return newScraper("globalparadise", "https://bans.theglobalparadise.org/", "index.php?p=banlist",
		parseDefault, nextURLLast, parseDefaultTime)
}

func newSavageServidoresScraper() *sbScraper {
	return newScraper("savageservidores", "https://bans.savageservidores.com/", "index.php?p=banlist",
		parseFluent, nextURLFluent, parseDefaultTime)
}

func newCSIServersScraper() *sbScraper {
	return newScraper("csiservers", "https://bans.csiservers.com/", "index.php?p=banlist",
		parseDefault, nextURLLast, parseDefaultTime)
}

func newLBGamingScraper() *sbScraper {
	return newScraper("lbgaming", "https://bans.lbgaming.co/", "index.php?p=banlist",
		parseDefault, nextURLLast, parseSkialTime)
}

func newFluxTFScraper() *sbScraper {
	return newScraper("fluxtf", "https://bans.flux.tf/", "index.php?p=banlist",
		parseDefault, nextURLLast, parseFluxTime)
}

func newDarkPyroScraper() *sbScraper {
	return newScraper("darkpyro", "https://bans.darkpyrogaming.com/", "index.php?p=banlist",
		parseDefault, nextURLLast, parseDarkPyroTime)
}

func newOpstOnlineScraper() *sbScraper {
	return newScraper("opstonline", "https://www.opstonline.com/bans/", "index.php?p=banlist",
		parseDefault, nextURLLast, parseSkialTime)
}

func newBouncyBallScraper() *sbScraper {
	return newScraper("bouncyball", "https://www.bouncyball.eu/bans2/", "index.php?p=banlist",
		parseDefault, nextURLLast, parseSkialTime)
}

func newFurryPoundScraper() *sbScraper {
	return newScraper("furrypound", "http://sourcebans.thefurrypound.org/", "index.php?p=banlist",
		parseDefault, nextURLLast, parseFurryPoundTime)
}

func newRetroServersScraper() *sbScraper {
	return newScraper("retroservers", "https://bans.retroservers.net/", "index.php?p=banlist",
		parseDefault, nextURLLast, parseDefaultTime)
}

func newSwapShopScraper() *sbScraper {
	return newScraper("swapshop", "http://tf2swapshop.com/sourcebans/", "index.php?p=banlist",
		parseDefault, nextURLLast, parseSkialTime)
}

func newECJScraper() *sbScraper {
	return newScraper("ecj", "https://ecj.tf/sourcebans/", "index.php?p=banlist",
		parseDefault, nextURLLast, parseSkialTime)
}

func newJumpAcademyScraper() *sbScraper {
	return newScraper("jumpacademy", "https://bans.jumpacademy.tf/", "index.php?p=banlist",
		parseDefault, nextURLLast, parseDefaultTime)
}

func newTF2ROScraper() *sbScraper {
	// Not enough values to page yet...
	return newScraper("tf2ro", "https://bans.tf2ro.com/", "index.php?p=banlist",
		parseDefault, nextURLLast, parseDefaultTime)
}

func newSameTeemScraper() *sbScraper {
	return newScraper("sameteem", "https://sameteem.com/sourcebans/", "index.php?p=banlist",
		parseDefault, nextURLLast, parseDefaultTime)
}

func newPowerFPSScraper() *sbScraper {
	return newScraper("powerfps", "https://bans.powerfps.com/", "index.php?p=banlist",
		parseDefault, nextURLLast, parseSkialTime)
}

func new7MauScraper() *sbScraper {
	return newScraper("7mau", "https://7-mau.com/server/", "index.php?p=banlist",
		parseFluent, nextURLFluent, parseDefaultTime)
}

func newGhostCapScraper() *sbScraper {
	return newScraper("ghostcap", "https://sourcebans.ghostcap.com/", "index.php?p=banlist",
		parseDefault, nextURLLast, parseDefaultTime)
}

func newSpectreScraper() *sbScraper {
	return newScraper("spectre", "https://spectre.gg/bans/", "index.php?p=banlist",
		parseDefault, nextURLLast, parseDefaultTime)
}

func newDreamFireScraper() *sbScraper {
	return newScraper("dreamfire", "https://sourcebans.dreamfire.fr/", "index.php?p=banlist",
		parseDefault, nextURLLast, parseDefaultTime)
}

func newSettiScraper() *sbScraper {
	return newScraper("setti", "https://pong.setti.info/sourcebans/", "index.php?p=banlist",
		parseDefault, nextURLLast, parseDefaultTime)
}

func newGunServerScraper() *sbScraper {
	return newScraper("gunserver", "https://gunserver.ru/sourcebans/", "index.php?p=banlist",
		parseDefault, nextURLFirst, parseGunServer)
}

func newHellClanScraper() *sbScraper {
	return newScraper("hellclan", "https://hellclan.co.uk/sourcebans/", "index.php?p=banlist",
		parseDefault, nextURLLast, parseHellClanTime)
}

func newSneaksScraper() *sbScraper {
	return newScraper("sneaks", "https://bans.snksrv.com/", "index.php?p=banlist",
		parseDefault, nextURLLast, parseSneakTime)
}

func newNideScraper() *sbScraper {
	return newScraper("nide", "https://bans.nide.gg/", "index.php?p=banlist",
		parseFluent, nextURLFluent, parseDefaultTime)
}

func newAstraManiaScraper() *sbScraper {
	return newScraper("astramania", "https://astramania.ro/sban2/", "index.php?p=banlist",
		parseDefault, nextURLLast, parseTrailYear)
}

func newTF2MapsScraper() *sbScraper {
	return newScraper("tf2maps", "https://bans.tf2maps.net/", "index.php?p=banlist",
		parseDefault, nextURLLast, parseDefaultTime)
}

func newVaticanCityScraper() *sbScraper {
	return newScraper("vaticancity", "https://www.the-vaticancity.com/sourcebans/", "index.php?p=banlist",
		parseDefault, nextURLLast, parseSkialTime)
}

func newLazyNeerScraper() *sbScraper {
	return newScraper("lazyneer", "https://www.lazyneer.com/SourceBans/", "index.php?p=banlist",
		parseDefault, nextURLLast, parseSkialAltTime)
}

func newTheVilleScraper() *sbScraper {
	return newScraper("theville", "https://www.theville.org/sourcebans/", "index.php?p=banlist",
		parseDefault, nextURLLast, parseSkialTime)
}

func newOreonScraper() *sbScraper {
	return newScraper("oreon", "https://www.tf2-oreon.fr/sourceban/", "index.php?p=banlist",
		parseDefault, nextURLLast, parseSkialTime)
}

func newTriggerHappyScraper() *sbScraper {
	return newScraper("triggerhappy", "https://triggerhappygamers.com/sourcebans/", "index.php?p=banlist",
		parseDefault, nextURLLast, parseTriggerHappyTime)
}

func newDefuseRoScraper() *sbScraper {
	return newScraper("defusero", "https://bans.defusero.org/", "index.php?p=banlist",
		parseFluent, nextURLFluent, parseDefaultTime)
}

func newTawernaScraper() *sbScraper {
	return newScraper("tawerna", "https://sb.tawerna.tf/", "index.php?p=banlist",
		parseDefault, nextURLLast, parseSkialTime)
}

func newTitanScraper() *sbScraper {
	return newScraper("titan", "https://bans.titan.tf/", "index.php?p=banlist",
		parseDefault, nextURLLast, parseTitanTime)
}

func newDiscFFScraper() *sbScraper {
	return newScraper("discff", "http://disc-ff.site.nfoservers.com/sourcebanstf2/", "index.php?p=banlist",
		parseDefault, nextURLLast, parseDefaultTime)
}

//func NewOtakuScraper() *sbScraper {
//	return newScraper("otaku", "https://bans.otaku.tf/bans", "",
//		parseDefault, nextURLLast, parseOtakuTime)
//}

// 05-17-23 03:07
func parseSkialTime(s string) (time.Time, error) {
	if s == "Not applicable." || s == "Permanent" {
		return time.Time{}, nil
	}
	return time.Parse("01-02-06 15:04", s)
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
	return time.Parse("02/01/06 15:04 PM", s)
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

// 17-05-2023 03:07:05
func parseSneakTime(s string) (time.Time, error) {
	if s == "Not applicable." || s == "Permanent" {
		return time.Time{}, nil
	}
	return time.Parse("02-01-2006 15:04 PM MST", s)
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
	return time.Parse("2006-01-02 15:04:05", s)
}

// Thu, May 11, 2023 7:14 PM
func parsePancakesTime(s string) (time.Time, error) {
	if s == "Not applicable." || s == "never, this is permanent" {
		return time.Time{}, nil
	}
	return time.Parse("Mon, Jan 02, 2006 15:04 PM", s)
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

func nextURLFluent(doc *goquery.Selection) string {
	nextPage, _ := doc.Find(".pagination a[href]").First().Attr("href")
	return nextPage
}

func nextURLFirst(doc *goquery.Selection) string {
	nextPage, _ := doc.Find("#banlist-nav a[href]").First().Attr("href")
	return nextPage
}

func nextURLLast(doc *goquery.Selection) string {
	nextPage, _ := doc.Find("#banlist-nav a[href]").Last().Attr("href")
	return nextPage
}

// https://github.com/brhndursun/SourceBans-StarTheme
//func parseStar(doc *goquery.Selection, urlFunc nextURLFunc, parseTime parseTimeFunc) (string, []sbRecord, error) {
//	panic("todo")
//}

// https://github.com/aXenDeveloper/sourcebans-web-theme-fluent
func parseFluent(doc *goquery.Selection, urlFunc nextURLFunc, parseTime parseTimeFunc) (string, []sbRecord, error) {
	var (
		bans   []sbRecord
		curBan sbRecord
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
			curBan.SteamID = sid64
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
				curBan.SteamID = 0 // invalidate it
			} else if lowerVal == "permanent" {
				curBan.Permanent = true
			}
			curBan.Length = 0
		case "expires on":
			if curBan.Permanent || !curBan.SteamID.Valid() {
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
			if curBan.SteamID.Valid() {
				bans = append(bans, curBan)
			}
			curBan = sbRecord{}
		}
	})
	return urlFunc(doc), bans, nil
}

func parseDefault(doc *goquery.Selection, urlFunc nextURLFunc, parseTime parseTimeFunc) (string, []sbRecord, error) {
	var (
		bans     []sbRecord
		curBan   sbRecord
		curState = unknown
		isValue  bool
	)
	doc.Find("#banlist .listtable table tr td").Each(func(i int, selection *goquery.Selection) {
		txt := strings.TrimSpace(selection.Text())
		if !isValue {
			switch strings.ToLower(txt) {
			case "player":
				curState = player
				isValue = true
			case "steam id":
				curState = steamID
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
				curBan.SteamID = sid64
			case invokedOn:
				t, errTime := parseTime(txt)
				if errTime != nil {
					logger.Error("Failed to parse invoke time", zap.Error(errTime))
					return
				}
				curBan.CreatedOn = t
			case banLength:
				lowerVal := strings.ToLower(txt)
				if strings.Contains(lowerVal, "unbanned") {
					curBan.SteamID = 0 // invalidate it
				} else if lowerVal == "permanent" {
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
				if curBan.SteamID.Valid() {
					bans = append(bans, curBan)
				}
				curBan = sbRecord{}
				curState = last
			}
			curState = unknown
		}

	})
	return urlFunc(doc), bans, nil
}

type megaScatterNode struct {
	ID                  string `json:"id"`
	ID3                 string `json:"id3"`
	ID1                 string `json:"id1"`
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

func parseMegaScatter(bodyReader io.Reader) ([]sbRecord, error) {
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

	fmt.Println(pass3[0:1024])

	fmt.Println(string(pass3[len(match[1])-2048]))

	o, _ := os.Create("temp.json")
	_, _ = io.WriteString(o, pass3)
	_ = o.Close()
	var msNodes []megaScatterNode
	if errJSON := json.Unmarshal([]byte(pass3), &msNodes); errJSON != nil {
		return nil, errJSON
	}
	return nil, nil
}
