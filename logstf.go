package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/leighmacdonald/bd-api/domain"
	"io"
	"log/slog"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/gocolly/colly"
	"github.com/gocolly/colly/extensions"
	"github.com/gocolly/colly/queue"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

var (
	reLOGSResults   = regexp.MustCompile(`<p>(\d+|\d+,\d+)\sresults</p>`)
	errParseLogsRow = errors.New("failed to parse title")
	errParseAttrs   = errors.New("failed to parse valid attrs")
)

func getLogsTF(ctx context.Context, steamid steamid.SteamID) (int64, error) {
	const expectedMatches = 2

	resp, err := get(ctx, fmt.Sprintf("https://logs.tf/profile/%d", steamid.Int64()), nil)
	if err != nil {
		return 0, err
	}

	defer logCloser(resp.Body)

	body, errRead := io.ReadAll(resp.Body)
	if errRead != nil {
		return 0, errors.Join(errRead, errResponseRead)
	}

	bStr := string(body)
	if strings.Contains(bStr, "No logs found.") {
		return 0, nil
	}

	match := reLOGSResults.FindStringSubmatch(bStr)
	if len(match) != expectedMatches {
		return 0, errResponseInvalid
	}

	value := strings.ReplaceAll(match[1], ",", "")

	count, errParse := strconv.ParseInt(value, 10, 64)
	if errParse != nil || count <= 0 {
		return 0, errors.Join(errParse, fmt.Errorf("%w: %s", errResponseDecode, match[1]))
	}

	return count, nil
}

type logTableRow struct {
	logID   int64
	title   string
	mapName string
	format  string
	views   int64
	date    time.Time
}

type logsTFScraper struct {
	*colly.Collector
	log     *slog.Logger
	curPage int
	queue   *queue.Queue
}

func newLogsTFScraper(cacheDir string) (*logsTFScraper, error) {
	logger := slog.With("name", "logstf")
	debugLogger := scrapeLogger{logger: logger} //nolint:exhaustruct

	reqQueue, errQueue := queue.New(1, &queue.InMemoryQueueStorage{MaxSize: maxQueueSize})
	if errQueue != nil {
		return nil, errors.Join(errQueue, errScrapeQueueInit)
	}

	collector := colly.NewCollector(
		colly.UserAgent("bd-api"),
		colly.CacheDir(filepath.Join(cacheDir, "logstf")),
		colly.Debugger(&debugLogger),
		colly.AllowedDomains("logs.tf"),
	)

	scraper := logsTFScraper{
		Collector: collector,
		curPage:   1,
		log:       logger,
		queue:     reqQueue,
	}

	scraper.SetRequestTimeout(requestTimeout)
	scraper.OnRequest(func(r *colly.Request) {
		slog.Debug("Visiting", slog.String("url", r.URL.String()))
	})
	extensions.RandomUserAgent(scraper.Collector)

	if errLimit := scraper.Limit(&colly.LimitRule{ //nolint:exhaustruct
		DomainGlob:  "*logs.tf",
		RandomDelay: randomDelay,
	}); errLimit != nil {
		return nil, errors.Join(errLimit, errScrapeLimit)
	}

	scraper.OnError(func(r *colly.Response, err error) {
		logger.Error("Request error", slog.String("url", r.Request.URL.String()), ErrAttr(err))
	})

	return &scraper, nil
}

func (s logsTFScraper) start(ctx context.Context) {
	scraperInterval := time.Hour
	scraperTimer := time.NewTimer(scraperInterval)

	s.scrape()

	for {
		select {
		case <-scraperTimer.C:
			s.scrape()
			scraperTimer.Reset(scraperInterval)
		case <-ctx.Done():
			return
		}
	}
}

func (s logsTFScraper) scrape() {
	startTime := time.Now()

	s.log.Info("Starting scrape job")
	s.OnHTML("body", func(element *colly.HTMLElement) {
		results := s.parse(element.DOM)
		if results == nil {
			s.log.Warn("No results parsed")

			return
		}

		for _, res := range results {
			s.log.Info(res.date.String())
		}

		nextPage := s.nextURL(element.DOM)
		s.log.Info(nextPage)
		if nextPage != "" {
			if errNext := s.queue.AddURL(nextPage); errNext != nil {
				s.log.Error("failed to add url to queue", ErrAttr(errNext))
			}
		}
	})

	if errAdd := s.queue.AddURL("https://logs.tf/?p=1"); errAdd != nil {
		s.log.Error("Failed to add queue error", ErrAttr(errAdd))

		return
	}

	if errRun := s.queue.Run(s.Collector); errRun != nil {
		s.log.Error("Queue returned error", ErrAttr(errRun))

		return
	}

	s.log.Info("Completed scrape job",
		slog.Duration("duration", time.Since(startTime)))
}

func (s logsTFScraper) nextURL(doc *goquery.Selection) string {
	node := doc.Find(".pagination ul li span strong").First()
	nextPage, err := strconv.ParseInt(node.Text(), 10, 64)
	if err != nil {
		return ""
	}

	return fmt.Sprintf("https://logs.tf/?p=%d", nextPage+1)
}

func (s logsTFScraper) parse(doc *goquery.Selection) []logTableRow {
	var results []logTableRow

	doc.Find(".loglist tbody tr").Each(func(_ int, selection *goquery.Selection) {
		row, errRow := logsTFSelectionToRow(selection)
		if errRow != nil {
			s.log.Error("Failed to parse log row", ErrAttr(errRow))

			return
		}
		results = append(results, row)
	})

	return results
}

func logsTFSelectionToRow(doc *goquery.Selection) (logTableRow, error) {
	children := doc.Children()
	title := children.Get(0).FirstChild.Data
	titleAttrs := children.Get(0).FirstChild.Attr

	if len(titleAttrs) != 1 {
		return logTableRow{}, errParseAttrs
	}

	logID, errID := strconv.ParseInt(strings.Replace(titleAttrs[0].Val, "/", "", 1), 10, 64)
	if errID != nil {
		return logTableRow{}, errors.Join(errID, errParseLogsRow)
	}

	var mapName string
	mapNameChild := children.Get(1)
	if mapNameChild.FirstChild != nil {
		mapName = mapNameChild.FirstChild.Data
	} else {
		slog.Warn("map name missing")
	}
	format := children.Get(2).FirstChild.Data

	views, errViews := strconv.ParseInt(children.Get(3).FirstChild.Data, 10, 64)
	if errViews != nil {
		return logTableRow{}, errors.Join(errViews, errParseLogsRow)
	}

	date := children.Get(4).FirstChild.Data

	// 24-Jun-2024 23:11:13
	created, errCreated := parseLogsTFDate(date)
	if errCreated != nil {
		return logTableRow{}, errors.Join(errCreated, errScrapeParseTime)
	}

	return logTableRow{
		logID:   logID,
		title:   title,
		mapName: mapName,
		format:  format,
		views:   views,
		date:    created,
	}, nil
}

// newDetailsFromDoc will parse the logstf match details page HTML into a domain.LogsTFMatch.
//
// Does not currently parse, and probably won't ever parse these because they are not valuable for our use case:
// - Individual player class stats weapon details
// - Notable round events
// - Player kills vs class table
func newDetailsFromDoc(doc *goquery.Document) (*domain.LogsTFMatch, error) {
	var match domain.LogsTFMatch

	if err := parseHeader(doc, &match); err != nil {
		return nil, err
	}

	if err := parseScores(doc, &match); err != nil {
		return nil, err
	}

	if err := parsePlayers(doc, &match); err != nil {
		return nil, err
	}

	if err := parseRounds(doc, &match); err != nil {
		return nil, err
	}

	if err := parseMedics(doc, &match); err != nil {
		return nil, err
	}

	return &match, nil
}

func parseHeader(doc *goquery.Document, match *domain.LogsTFMatch) error {
	var err error

	doc.Find("#log-header h3").EachWithBreak(func(i int, selection *goquery.Selection) bool {
		slog.Info(selection.Text())
		attrID, found := selection.Attr("id")
		if !found {
			return false
		}
		switch attrID {
		case "log-name":
			match.Title = selection.Text()
		case "log-map":
			match.Map = selection.Text()
		case "log-length":
			dur, errDur := parseLogsTFDuration(selection.Text())
			if errDur != nil {
				err = errDur
				return false
			}

			match.Duration = dur
		case "log-date":
			co, errDate := parseLogsTFDate(selection.Text())
			if errDate != nil {
				err = errDate

				return false
			}
			match.CreatedOn = co
		}

		return true
	})

	return err
}

func parseScores(doc *goquery.Document, match *domain.LogsTFMatch) error {
	var err error
	doc.Find("#log-score h1").EachWithBreak(func(i int, selection *goquery.Selection) bool {
		if i == 1 {
			score, errScore := strconv.Atoi(selection.Text())
			if errScore != nil {
				err = errScore

				return false
			}

			match.ScoreBLU = score
		} else if i == 2 {
			score, errScore := strconv.Atoi(selection.Text())
			if errScore != nil {
				err = errScore

				return false
			}
			match.ScoreRED = score
		}

		return true
	})

	return err
}

func parsePlayers(doc *goquery.Document, match *domain.LogsTFMatch) error {
	var err error

	doc.Find("#players tbody tr").EachWithBreak(func(i int, selection *goquery.Selection) bool {
		var player domain.LogsTFPlayer
		playerID, found := selection.Attr("id")
		if !found {
			slog.Warn("Failed to find player row")

			return false
		}

		parts := strings.SplitN(playerID, "_", 2)
		if len(parts) != 2 {
			slog.Warn("Could not parse player steamid", slog.String("attr", playerID))

			return false
		}

		playerSID := steamid.New(parts[1])
		if !playerSID.Valid() {
			slog.Warn("Parsed invalid steam id", slog.String("attr", playerID))

			return false
		}

		player.SteamID = playerSID

		selection.Find("td").EachWithBreak(func(i int, innerSelection *goquery.Selection) bool {
			switch i {
			case 0:
				if strings.ToLower(innerSelection.Text()) == "blu" {
					player.Team = domain.BLU
				} else {
					player.Team = domain.RED
				}
			case 1:
				player.Name = innerSelection.Find(".dropdown-toggle").First().Text()
			case 2:
				parseClassStats(innerSelection, &player)
			case 3:
				player.Kills = stringToIntWithDefault(innerSelection.Text())
			case 4:
				player.Assists = stringToIntWithDefault(innerSelection.Text())
			case 5:
				player.Deaths = stringToIntWithDefault(innerSelection.Text())
			case 6:
				player.Damage = stringToInt64WithDefault(innerSelection.Text())
			case 7:
				player.DPM = stringToIntWithDefault(innerSelection.Text())
			case 8:
				player.KAD = stringToFloatWithDefault(innerSelection.Text())
			case 9:
				player.KD = stringToFloatWithDefault(innerSelection.Text())
			case 10:
				player.DamageTaken = stringToIntWithDefault(innerSelection.Text())
			case 11:
				player.DTM = stringToIntWithDefault(innerSelection.Text())
			case 12:
				player.HealthPacks = stringToIntWithDefault(innerSelection.Text())
			case 13:
				player.Backstabs = stringToIntWithDefault(innerSelection.Text())
			case 14:
				player.Headshots = stringToIntWithDefault(innerSelection.Text())
			case 15:
				player.Airshots = stringToIntWithDefault(innerSelection.Text())
			case 16:
				player.Caps = stringToIntWithDefault(innerSelection.Text())
			}

			return true
		})

		match.Players = append(match.Players, player)

		return true
	})

	return err
}

func stringToClass(name string) domain.PlayerClass {
	value := strings.ToLower(name)
	switch value {
	case "scout":
		return domain.Scout
	case "soldier":
		return domain.Soldier
	case "pyro":
		return domain.Pyro
	case "demoman":
		return domain.Demo
	case "heavyweapons":
		return domain.Heavy
	case "engineer":
		return domain.Engineer
	case "medic":
		return domain.Medic
	case "sniper":
		return domain.Sniper
	case "spy":
		return domain.Spy
	default:
		return domain.Spectator
	}
}

func parseClassStats(sel *goquery.Selection, player *domain.LogsTFPlayer) {
	sel.Find("i").EachWithBreak(func(i int, selection *goquery.Selection) bool {
		content, found := selection.Attr("data-content")
		if !found {
			return false
		}

		classTitle, foundTitle := selection.Attr("data-title")
		if !foundTitle {
			return false
		}

		class := domain.LogsTFPlayerClass{Class: stringToClass(classTitle)}
		if err := parseClass(content, &class); err != nil {
			return false
		}

		player.Classes = append(player.Classes, class)

		return true
	})
}

func stringToIntWithDefault(value string) int {
	v, err := strconv.Atoi(value)
	if err != nil {
		slog.Warn("Failed to parse int string", ErrAttr(err))
		return 0
	}

	return v
}

func stringToInt64WithDefault(value string) int64 {
	v, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		slog.Warn("Failed to parse int string", ErrAttr(err))
		return 0
	}

	return v
}

func stringToFloatWithDefault(value string) float32 {
	v, err := strconv.ParseFloat(value, 32)
	if err != nil {
		slog.Warn("Failed to parse int string", ErrAttr(err))
		return 0
	}

	return float32(v)
}

func parseClass(body string, class *domain.LogsTFPlayerClass) error {
	selection, err := goquery.NewDocumentFromReader(strings.NewReader(body))
	if err != nil {
		return err
	}

	selection.Find("table tbody").EachWithBreak(func(tableIdx int, selection *goquery.Selection) bool {
		selection.Find("tr").Each(func(i int, selection *goquery.Selection) {
			selection.Find("td").EachWithBreak(func(i int, selection *goquery.Selection) bool {
				value := selection.Text()
				// Parse the first overall table
				if tableIdx == 0 {
					switch i {
					case 0:
						d, errDur := parseLogsTFDuration(value)
						if errDur != nil {
							err = errDur
							return false
						}
						class.Played = d
					case 1:
						class.Kills = stringToIntWithDefault(value)
					case 2:
						class.Assists = stringToIntWithDefault(value)
					case 3:
						class.Deaths = stringToIntWithDefault(value)
					case 4:
						class.Damage = stringToIntWithDefault(value)
					}
				} else {
					// Parse the weapons
				}

				return true
			})
		})

		return true
	})

	return err
}

func parseRounds(selection *goquery.Document, match *domain.LogsTFMatch) error {

	selection.Find("#log-section-rounds .round_row").EachWithBreak(func(i int, selection *goquery.Selection) bool {
		var round domain.LogsTFRound

		selection.Find("td").EachWithBreak(func(i int, selection *goquery.Selection) bool {
			switch i {
			case 0:
				round.Round = stringToIntWithDefault(selection.Text())
			case 1:
				dur, errDur := parseLogsTFDuration(selection.Text())
				if errDur != nil {
					slog.Error("Failed to parse round time", ErrAttr(errDur))
					return false
				}
				round.Length = dur
			case 2:
				parts := strings.SplitN(selection.Text(), " - ", 2)
				if len(parts) != 2 {
					slog.Error("Failed to parse round scores")
					return false
				}

				round.ScoreBLU = stringToIntWithDefault(parts[0])
				round.ScoreRED = stringToIntWithDefault(parts[1])
			case 3:
				round.KillsBLU = stringToIntWithDefault(selection.Text())
			case 4:
				round.KillsRED = stringToIntWithDefault(selection.Text())
			case 5:
				round.UbersBLU = stringToIntWithDefault(selection.Text())
			case 6:
				round.UbersRED = stringToIntWithDefault(selection.Text())
			case 7:
				round.DamageBLU = stringToIntWithDefault(selection.Text())
			case 8:
				round.DamageRED = stringToIntWithDefault(selection.Text())
				match.Rounds = append(match.Rounds, round)
				round = domain.LogsTFRound{}
			}
			return true
		})
		return true
	})
	return nil
}

var (
	healingRx = regexp.MustCompile("(\\d+)\\s\\((\\d+)/m\\)")
	medigunRx = regexp.MustCompile("(.+?):\\s(\\d+)\\s")
)

func parseMedics(doc *goquery.Document, match *domain.LogsTFMatch) error {
	var err error

	parent := doc.Find("#log-section-healspread")
	parent.Find(".healtable").Each(func(i int, selection *goquery.Selection) {
		medic := domain.LogsTFMedic{LogID: match.LogID}

		playerName := selection.Find("h6").Text()
		for _, player := range match.Players {
			if player.Name == playerName {
				medic.SteamID = player.SteamID
			}
		}

		var curField string

		selection.Find(".medstats td").EachWithBreak(func(i int, selection *goquery.Selection) bool {
			if i%2 == 0 {
				curField = strings.ToLower(selection.Text())
			} else {
				switch curField {
				case "healing":
					parts := healingRx.FindStringSubmatch(selection.Text())
					if len(parts) != 3 {
						slog.Error("failed to get healing values")

						return false
					}
					medic.Healing = stringToInt64WithDefault(parts[1])
					medic.HealingPerMin = stringToIntWithDefault(parts[2])
				case "charges":
					selection.Find("li").EachWithBreak(func(i int, selection *goquery.Selection) bool {
						parts := medigunRx.FindStringSubmatch(selection.Text())
						if len(parts) != 3 {
							return false
						}
						switch strings.ToLower(parts[1]) {
						case "medigun":
							medic.ChargesMedigun = stringToIntWithDefault(parts[2])
						case "kritzkrieg":
							medic.ChargesKritz = stringToIntWithDefault(parts[2])
						case "quickfix":
							medic.ChargesKritz = stringToIntWithDefault(parts[2])
						case "vaccinator":
							medic.ChargesKritz = stringToIntWithDefault(parts[2])
						default:
							panic(parts)
						}

						return true
					})
				case "drops":
					medic.Drops = stringToIntWithDefault(selection.Text())
				case "avg time to build":
					medic.AvgTimeBuild = parseLogsTFDurationMedicInt(selection.Text())
				case "avg time before using":
					medic.AvgTimeUse = parseLogsTFDurationMedicInt(selection.Text())
				case "near full charge deaths":
					medic.NearFullDeath = stringToIntWithDefault(selection.Text())
				case "avg uber length":
					medic.AvgUberLen = parseLogsTFDurationMedicFloat(selection.Text())
				case "deaths after charge":
					medic.DeathAfterCharge = stringToIntWithDefault(selection.Text())
				case "major advantages lost":
					medic.MajorAdvLost = stringToIntWithDefault(selection.Text())
				case "biggest advantage lost":
					medic.BiggestAdvLost = parseLogsTFDurationMedicInt(selection.Text())
				}
			}

			return true
		})

		selection.Find(".healsort").EachWithBreak(func(i int, selection *goquery.Selection) bool {
			return true
		})

		match.Medics = append(match.Medics, medic)
	})

	return err
}

// 16:56
func parseLogsTFDuration(d string) (time.Duration, error) {
	durString := strings.Replace(d, ":", "m", 1) + "s"
	return time.ParseDuration(durString)
}

// 05-Feb-2022 06:39:42
func parseLogsTFDate(d string) (time.Time, error) {
	return time.Parse("02-Jan-2006 15:04:05", d)
}

func parseLogsTFDurationMedicInt(value string) time.Duration {
	seconds, err := strconv.Atoi(strings.Replace(value, " s", "", -1))
	if err != nil {
		slog.Error("Failed to parse medic duration", slog.String("value", value))

		return 0
	}

	return time.Second * time.Duration(seconds)
}

func parseLogsTFDurationMedicFloat(value string) time.Duration {
	seconds, err := strconv.ParseFloat(strings.Replace(value, " s", "", -1), 10)
	if err != nil {
		slog.Error("Failed to parse medic duration", slog.String("value", value))

		return 0
	}

	return time.Second * time.Duration(seconds)
}
