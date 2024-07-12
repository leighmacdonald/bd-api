package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/gocolly/colly"
	"github.com/gocolly/colly/extensions"
	"github.com/gocolly/colly/queue"
	"github.com/leighmacdonald/bd-api/domain"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

var (
	errFindStartID    = errors.New("failed to find start id")
	errParseStartID   = errors.New("failed to parse start id")
	errParseDuration  = errors.New("failed to parse duration")
	errCreateDocument = errors.New("failed to create goquery document")
	errParseScores    = errors.New("failed to parse scores")
	errParseDate      = errors.New("failed to parse logs creation date")
	errLogID          = errors.New("failed to parse log id")
)

type logsTFScraper struct {
	*colly.Collector
	log   *slog.Logger
	queue *queue.Queue
	db    *pgStore
}

func newLogsTFScraper(database *pgStore, config appConfig) (*logsTFScraper, error) {
	logger := slog.With("name", "logstf")
	debugLogger := scrapeLogger{logger: logger} //nolint:exhaustruct

	reqQueue, errQueue := queue.New(6, &queue.InMemoryQueueStorage{})
	if errQueue != nil {
		return nil, errors.Join(errQueue, errScrapeQueueInit)
	}

	collector := colly.NewCollector(
		colly.UserAgent("bd-api"),
		colly.CacheDir(filepath.Join(config.CacheDir, "logstf")),
		colly.Debugger(&debugLogger),
		colly.AllowedDomains("logs.tf"),
	)

	extensions.RandomUserAgent(collector)

	scraper := logsTFScraper{
		Collector: collector,
		log:       logger,
		queue:     reqQueue,
		db:        database,
	}

	scraper.SetRequestTimeout(requestTimeout)
	scraper.OnRequest(func(r *colly.Request) {
		slog.Debug("Visiting", slog.String("url", r.URL.String()))
	})

	initialDelay := time.Millisecond * 750

	parallelism := 1
	if config.ProxiesEnabled {
		parallelism = len(config.Proxies)
	}

	if errLimit := scraper.Limit(&colly.LimitRule{ //nolint:exhaustruct
		DomainGlob:  "*logs.tf",
		Delay:       initialDelay,
		Parallelism: parallelism,
	}); errLimit != nil {
		return nil, errors.Join(errLimit, errScrapeLimit)
	}

	// Keep track of which log ids
	var retries []int

	scraper.OnError(func(response *colly.Response, err error) {
		if response.StatusCode != http.StatusTooManyRequests {
			logger.Error("Request error", slog.String("url", response.Request.URL.String()), ErrAttr(err))

			return
		}

		// initialDelay += time.Millisecond * 100
		slog.Info("Too many requests...", slog.String("delay", initialDelay.String()))

		time.Sleep(time.Second * 2)

		if errLimit := scraper.Limit(&colly.LimitRule{ //nolint:exhaustruct
			DomainGlob: "*logs.tf",
			Delay:      initialDelay,
		}); errLimit != nil {
			panic(errScrapeLimit)
		}
		idStr := strings.TrimPrefix(response.Request.URL.Path, "/")
		logID, errID := strconv.Atoi(idStr)
		if errID != nil {
			panic(errID)
		}
		if slices.Contains(retries, logID) {
			logger.Error("Failed retry", slog.String("url", response.Request.URL.String()), ErrAttr(err))

			return
		}

		retries = append(retries, logID)

		if errRetry := response.Request.Retry(); errRetry != nil {
			logger.Error("Retry error", slog.String("url", response.Request.URL.String()), ErrAttr(err))
		}
	})

	return &scraper, nil
}

func (s logsTFScraper) start(ctx context.Context) {
	scraperInterval := time.Hour
	scraperTimer := time.NewTimer(scraperInterval)

	s.scrape(ctx)

	for {
		select {
		case <-scraperTimer.C:
			s.scrape(ctx)
			scraperTimer.Reset(scraperInterval)
		case <-ctx.Done():
			return
		}
	}
}

func (s logsTFScraper) scrape(ctx context.Context) {
	s.log.Info("Starting scrape job")

	var (
		startTime    = time.Now()
		start        = true
		totalCount   = 0
		curCount     = 0
		successCount = 0
		errorCount   = 0
		maxID        = 3680000
		skipCount    = 0
		lastCount    = time.Now()
	)

	minID, errID := s.db.getNewestLogID(ctx)
	if errID != nil {
		if errors.Is(errID, errDatabaseNoResults) {
			minID = 1
		} else {
			slog.Error("Failed to get int id", ErrAttr(errID))
		}

		return
	}

	s.OnHTML("html", func(element *colly.HTMLElement) {
		if start {
			start = false
			// Setup the queue
			maxIDValue, errMaxID := getLogsTFMaxID(element.DOM)
			if errMaxID != nil {
				s.log.Error("No log id parsed, using default", ErrAttr(errMaxID))

				return
			}

			maxID = maxIDValue

			for i := range maxID {
				if i <= minID {
					continue
				}
				if errNext := s.queue.AddURL(fmt.Sprintf("https://logs.tf/%d", i)); errNext != nil {
					s.log.Error("failed to add url to queue", ErrAttr(errNext))
				}
			}

			return
		}

		totalCount++
		curCount++

		match, errMatch := parseMatchFromDoc(element.DOM)
		if errMatch != nil {
			if errors.Is(errMatch, errLogID) || errors.Is(errMatch, errMissingExtended) {
				skipCount++

				return
			}

			errorCount++
			slog.Error("failed to parse document", slog.String("log", fmt.Sprintf("https://logs.tf/%d", match.LogID)), ErrAttr(errMatch))

			return
		}

		if err := s.db.insertLogsTF(ctx, match); err != nil {
			slog.Error("Failed to insert match", ErrAttr(err))
		}

		successCount++

		if time.Since(lastCount) > time.Minute {
			slog.Info("Scrape stats",
				slog.Int("total", totalCount), slog.Int("per_min", curCount),
				slog.Int("success", successCount), slog.Int("error", errorCount), slog.Int("skip", skipCount),
				slog.String("current", fmt.Sprintf("%d/%d", match.LogID, maxID)))
			curCount = 0
			lastCount = time.Now()
		}

		slog.Debug("Parsed page", slog.Int("log_id", match.LogID))
	})

	// The index is checked first so that we can get the max ID
	if errAdd := s.queue.AddURL("https://logs.tf"); errAdd != nil {
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

func getLogsTFMaxID(doc *goquery.Selection) (int, error) {
	idStr, found := doc.Find("table.loglist tbody tr").First().Attr("id")
	if !found {
		return 0, errFindStartID
	}

	id, err := strconv.Atoi(strings.TrimPrefix(idStr, "log_"))
	if err != nil {
		return 0, errors.Join(err, errParseStartID)
	}

	return id, nil
}

// parseMatchFromDoc will parse the logstf match details page HTML into a domain.LogsTFMatch.
//
// Does not currently parse, and probably won't ever parse these because they are not valuable for our use case:
// - Individual player class stats weapon details
// - Notable round events
// - Player kills vs class table.
func parseMatchFromDoc(doc *goquery.Selection) (*domain.LogsTFMatch, error) {
	var match domain.LogsTFMatch

	if err := parseLogID(doc, &match); err != nil {
		return &match, err
	}

	logger := slog.With(slog.String("log", fmt.Sprintf("https://logs.tf/%d", match.LogID)))

	if err := parseHeader(doc, &match); err != nil {
		return &match, err
	}

	if err := parseScores(doc, &match); err != nil {
		return &match, err
	}

	parsePlayers(doc, logger, &match)

	if err := parseRounds(doc, &match); err != nil {
		return nil, err
	}

	parseMedics(doc, logger, &match)

	return &match, nil
}

func parseLogID(doc *goquery.Selection, match *domain.LogsTFMatch) error {
	attr, ok := doc.Find("meta[property='og:url']").Attr("content")
	if !ok {
		return errLogID
	}

	parts := strings.SplitAfter(attr, "logs.tf/")

	logID, err := strconv.Atoi(parts[1])
	if err != nil {
		return errors.Join(err, errLogID)
	}

	match.LogID = logID

	return nil
}

var errMissingExtended = errors.New("missing extended stats")

func parseHeader(doc *goquery.Selection, match *domain.LogsTFMatch) error {
	var err error

	found := doc.Find(".log-notification")
	if found.Nodes != nil {
		return errMissingExtended
	}

	doc.Find("#log-header h3").EachWithBreak(func(_ int, selection *goquery.Selection) bool {
		attrID, foundAttr := selection.Attr("id")
		if !foundAttr {
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

			match.Duration = domain.JSONDuration{Duration: dur}
		case "log-date":
			created, errDate := parseLogsTFDate(selection.Text())
			if errDate != nil {
				err = errDate

				return false
			}
			match.CreatedOn = created
		}

		return true
	})

	return err
}

func parseScores(doc *goquery.Selection, match *domain.LogsTFMatch) error {
	var err error
	doc.Find("#log-score h1").EachWithBreak(func(idx int, selection *goquery.Selection) bool {
		if idx == 1 {
			score, errScore := strconv.Atoi(selection.Text())
			if errScore != nil {
				err = errScore

				return false
			}

			match.ScoreBLU = score
		} else if idx == 2 {
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

func parsePlayers(doc *goquery.Selection, logger *slog.Logger, match *domain.LogsTFMatch) {
	match.LogFormatOld = doc.Find("#players thead th span").Get(6).FirstChild.Data == "KS"

	doc.Find("#players tbody tr").EachWithBreak(func(_ int, selection *goquery.Selection) bool {
		var player domain.LogsTFPlayer
		playerID, found := selection.Attr("id")
		if !found {
			logger.Warn("Failed to find player row")

			return false
		}

		parts := strings.SplitN(playerID, "_", 2)
		if len(parts) != 2 {
			logger.Warn("Could not parse player steamid", slog.String("attr", playerID))

			return false
		}

		playerSID := steamid.New(parts[1])
		if !playerSID.Valid() {
			logger.Warn("Parsed invalid steam id", slog.String("attr", playerID))

			return false
		}

		player.LogID = match.LogID
		player.SteamID = playerSID

		selection.Find("td").EachWithBreak(func(i int, innerSelection *goquery.Selection) bool {
			if match.LogFormatOld {
				return parseOldFormat(i, innerSelection, &player)
			}

			return parseNewFormat(i, innerSelection, &player)
		})

		match.Players = append(match.Players, player)

		return true
	})
}

func parseOldFormat(i int, innerSelection *goquery.Selection, player *domain.LogsTFPlayer) bool {
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
		parseClassStats(innerSelection, player)
	case 3:
		player.Kills = stringToIntWithDefault(innerSelection.Text())
	case 4:
		player.Assists = stringToIntWithDefault(innerSelection.Text())
	case 5:
		player.Deaths = stringToIntWithDefault(innerSelection.Text())
	case 7:
		player.Damage = stringToInt64WithDefault(innerSelection.Text())
	case 8:
		player.DPM = stringToIntWithDefault(innerSelection.Text())
	case 9:
		player.KAD = stringToFloatWithDefault(innerSelection.Text())
	case 10:
		player.KD = stringToFloatWithDefault(innerSelection.Text())
	case 11:
		player.HealthPacks = stringToIntWithDefault(innerSelection.Text())
	case 13:
		player.Backstabs = stringToIntWithDefault(innerSelection.Text())
	case 14:
		player.Headshots = stringToIntWithDefault(innerSelection.Text())
	case 16:
		player.Caps = stringToIntWithDefault(innerSelection.Text())
	}

	return true
}

func parseNewFormat(i int, innerSelection *goquery.Selection, player *domain.LogsTFPlayer) bool {
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
		parseClassStats(innerSelection, player)
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
	sel.Find("i").EachWithBreak(func(_ int, selection *goquery.Selection) bool {
		content, found := selection.Attr("data-content")
		if !found {
			return false
		}

		classTitle, foundTitle := selection.Attr("data-title")
		if !foundTitle {
			return false
		}

		class := domain.LogsTFPlayerClass{Class: stringToClass(classTitle), LogID: player.LogID, SteamID: player.SteamID}
		if err := parseClass(content, &class); err != nil {
			return false
		}

		player.Classes = append(player.Classes, class)

		return true
	})
}

func stringToIntWithDefault(value string) int {
	intVal, err := strconv.Atoi(value)
	if err != nil {
		slog.Warn("Failed to parse int string", ErrAttr(err))

		return 0
	}

	return intVal
}

func stringToInt64WithDefault(value string) int64 {
	intVal, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		slog.Warn("Failed to parse int64 string", ErrAttr(err))

		return 0
	}

	return intVal
}

func stringToFloatWithDefault(value string) float32 {
	floatVal, err := strconv.ParseFloat(value, 32)
	if err != nil {
		slog.Warn("Failed to parse float string", ErrAttr(err))

		return 0
	}

	return float32(floatVal)
}

func parseClass(body string, class *domain.LogsTFPlayerClass) error {
	selection, err := goquery.NewDocumentFromReader(strings.NewReader(body))
	if err != nil {
		return errors.Join(err, errCreateDocument)
	}

	selection.Find("table tbody").EachWithBreak(func(tableIdx int, selection *goquery.Selection) bool {
		selection.Find("tr").Each(func(_ int, selection *goquery.Selection) {
			selection.Find("td").EachWithBreak(func(i int, selection *goquery.Selection) bool {
				value := selection.Text()
				// Parse the first overall table
				if tableIdx == 0 {
					switch i {
					case 0:
						duration, errDur := parseLogsTFDuration(value)
						if errDur != nil {
							err = errDur

							return false
						}
						class.Played.Duration = duration
					case 1:
						class.Kills = stringToIntWithDefault(value)
					case 2:
						class.Assists = stringToIntWithDefault(value)
					case 3:
						class.Deaths = stringToIntWithDefault(value)
					case 4:
						class.Damage = stringToIntWithDefault(value)
					}
				}
				// else {
				//	// Parse the weapons
				// }

				return true
			})
		})

		return true
	})

	return err
}

func parseRounds(doc *goquery.Selection, match *domain.LogsTFMatch) error {
	var err error
	doc.Find("#log-section-rounds .round_row").EachWithBreak(func(_ int, selection *goquery.Selection) bool {
		round := domain.LogsTFRound{
			LogID: match.LogID,
		}

		selection.Find("td").EachWithBreak(func(i int, selection *goquery.Selection) bool {
			switch i {
			case 0:
				round.Round = stringToIntWithDefault(selection.Text())
			case 1:
				dur, errDur := parseLogsTFDuration(selection.Text())
				if errDur != nil {
					err = errors.Join(errDur, errParseDuration)

					return false
				}
				round.Length.Duration = dur
			case 2:
				parts := strings.SplitN(selection.Text(), " - ", 2)
				if len(parts) != 2 {
					err = errParseScores

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
			case 9:
				if strings.ToLower(selection.Text()) == "red" {
					round.MidFight = domain.RED
				} else {
					round.MidFight = domain.BLU
				}
				match.Rounds = append(match.Rounds, round)
				round = domain.LogsTFRound{}
			}

			return true
		})

		return true
	})

	return err
}

var (
	healingRx = regexp.MustCompile(`(\d+)\s\((\d+)/m\)`)
	medigunRx = regexp.MustCompile(`(.+?):\s(\d+)\s`)
)

func parseMedics(doc *goquery.Selection, logger *slog.Logger, match *domain.LogsTFMatch) {
	parent := doc.Find("#log-section-healspread")
	parent.Find(".healtable").Each(func(_ int, selection *goquery.Selection) {
		medic := domain.LogsTFMedic{LogID: match.LogID}

		playerName := selection.Find("h6").Text()
		for _, player := range match.Players {
			if player.Name == playerName {
				medic.SteamID = player.SteamID
			}
		}

		var curField string

		selection.Find(".medstats td").EachWithBreak(func(idx int, selection *goquery.Selection) bool {
			if idx%2 == 0 {
				curField = strings.ToLower(selection.Text())

				return true
			}

			switch curField {
			case "healing":
				parts := healingRx.FindStringSubmatch(selection.Text())
				if len(parts) != 3 {
					logger.Error("failed to get healing values")

					return false
				}
				medic.Healing = stringToInt64WithDefault(parts[1])
				medic.HealingPerMin = stringToIntWithDefault(parts[2])
			case "charges":
				if match.LogFormatOld {
					value := strings.TrimSpace(selection.Text())
					medic.ChargesMedigun = stringToIntWithDefault(value)

					return true
				}
				selection.Find("li").EachWithBreak(func(_ int, selection *goquery.Selection) bool {
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
						if parts[1] == "Unknown" {
							return false
						}
						panic(parts)
					}

					return true
				})
			case "drops":
				medic.Drops = stringToIntWithDefault(selection.Text())
			case "avg time to build":
				medic.AvgTimeBuild.Duration = parseLogsTFDurationMedicInt(selection.Text())
			case "avg time before using":
				medic.AvgTimeUse.Duration = parseLogsTFDurationMedicInt(selection.Text())
			case "near full charge deaths":
				medic.NearFullDeath = stringToIntWithDefault(selection.Text())
			case "avg uber length":
				medic.AvgUberLen.Duration = parseLogsTFDurationMedicFloat(selection.Text())
			case "deaths after charge":
				medic.DeathAfterCharge = stringToIntWithDefault(selection.Text())
			case "major advantages lost":
				medic.MajorAdvLost = stringToIntWithDefault(selection.Text())
			case "biggest advantage lost":
				medic.BiggestAdvLost.Duration = parseLogsTFDurationMedicInt(selection.Text())
			}

			return true
		})

		//
		// selection.Find(".healsort").EachWithBreak(func(i int, selection *goquery.Selection) bool {
		//	// TODO... maybe?
		//	return true
		// })

		match.Medics = append(match.Medics, medic)
	})
}

// 16:56
func parseLogsTFDuration(d string) (time.Duration, error) {
	durString := strings.Replace(d, ":", "m", 1) + "s"

	dur, err := time.ParseDuration(durString)
	if err != nil {
		return 0, errors.Join(err, errParseDuration)
	}

	return dur, nil
}

// 05-Feb-2022 06:39:42.
func parseLogsTFDate(d string) (time.Time, error) {
	date, err := time.Parse("02-Jan-2006 15:04:05", d)
	if err != nil {
		return time.Time{}, errors.Join(err, errParseDate)
	}

	return date, nil
}

func parseLogsTFDurationMedicInt(value string) time.Duration {
	seconds, err := strconv.Atoi(strings.ReplaceAll(value, " s", ""))
	if err != nil {
		slog.Error("Failed to parse medic duration", slog.String("value", value))

		return 0
	}

	return time.Second * time.Duration(seconds)
}

func parseLogsTFDurationMedicFloat(value string) time.Duration {
	seconds, err := strconv.ParseFloat(strings.ReplaceAll(value, " s", ""), 32)
	if err != nil {
		slog.Error("Failed to parse medic duration", slog.String("value", value))

		return 0
	}

	return time.Second * time.Duration(seconds)
}
