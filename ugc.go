package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/leighmacdonald/bd-api/domain"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

const (
	ugcHLHeader = "TF2 Highlander Medals"

	ugc6sHeader = "TF2 6vs6 Medals"

	ugc4sHeader = "TF2 4vs4 Medals"
)

var reUGCRank = regexp.MustCompile(`Season (\d+) (\D+) (\S+)`)

func getUGC(ctx context.Context, steam steamid.SteamID) ([]domain.Season, error) {
	resp, err := get(ctx,
		fmt.Sprintf("https://www.ugcleague.com/players_page.cfm?player_id=%d", steam.Int64()), nil)
	if err != nil {
		return nil, err
	}

	body, errRead := io.ReadAll(resp.Body)
	if errRead != nil {
		return nil, errors.Join(errRead, domain.ErrResponseRead)
	}

	defer logCloser(resp.Body)

	seasons, errSeasons := parseUGCRank(string(body))
	if errSeasons != nil {
		return seasons, errors.Join(errSeasons, domain.ErrResponseDecode)
	}

	return seasons, nil
}

func parseUGCRank(body string) ([]domain.Season, error) {
	dom, errReader := goquery.NewDocumentFromReader(strings.NewReader(body))
	if errReader != nil {
		return nil, errors.Join(errReader, domain.ErrGoQueryDocument)
	}

	var seasons []domain.Season

	dom.Find("h5").Each(func(_ int, selection *goquery.Selection) {
		text := selection.Text()

		if text == ugcHLHeader || text == ugc6sHeader || text == ugc4sHeader {
			selection.Next().ChildrenFiltered("li").Each(func(_ int, selection *goquery.Selection) {
				curRank, curRankStr := parseRankField(selection.Text())

				var format string

				switch text {
				case ugcHLHeader:
					format = "highlander"
				case ugc6sHeader:
					format = "6s"
				case ugc4sHeader:
					format = "4s"
				}

				seasons = append(seasons, domain.Season{
					League:      "UGC",
					Division:    curRankStr,
					DivisionInt: curRank,
					Format:      format,
					TeamName:    "",
				})
			})
		}
	})

	return seasons, nil
}

func parseRankField(field string) (domain.Division, string) {
	const expectedFieldCount = 4

	info := strings.Split(strings.ReplaceAll(field, "\n\n", ""), "\n")

	results := reUGCRank.FindStringSubmatch(info[0])

	if len(results) == expectedFieldCount {
		switch results[3] {
		case "Platinum":
			return domain.UGCRankPlatinum, "platinum"
		case "Gold":
			return domain.UGCRankGold, "gold"
		case "Silver":
			return domain.UGCRankSilver, "silver"
		case "Steel":
			return domain.UGCRankSteel, "steel"
		case "Iron":
			return domain.UGCRankIron, "iron"
		}
	}

	return domain.UGCRankNone, ""
}
