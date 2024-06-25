package main

import (
	"context"
	"errors"
	"fmt"
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
	created, errCreated := time.Parse("02-Jan-2006 15:04:05", date)
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
