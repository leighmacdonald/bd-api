package main

import (
	"context"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/pkg/errors"
	"io"
	"regexp"
	"strings"
)

const (
	ugcHLHeader = "TF2 Highlander Medals"
	ugc6sHeader = "TF2 6vs6 Medals"
	ugc4sHeader = "TF2 4vs4 Medals"
)

var reUGCRank *regexp.Regexp

func getUGC(ctx context.Context, steam steamid.SID64) ([]Season, error) {
	resp, err := get(ctx,
		fmt.Sprintf("https://www.ugcleague.com/players_page.cfm?player_id=%d", steam.Int64()), nil)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to get ugc response: %v", err)
	}
	b, errRead := io.ReadAll(resp.Body)
	if errRead != nil {
		return nil, errors.Wrapf(errRead, "Failed to read response body: %v", errRead)
	}
	seasons, errSeasons := parseUGCRank(string(b))
	if errSeasons != nil {
		return seasons, errors.Wrapf(errSeasons, "Failed to parse ugc response: %v", errSeasons)
	}
	return seasons, nil
}

func parseUGCRank(body string) ([]Season, error) {
	var seasons []Season
	dom, _ := goquery.NewDocumentFromReader(strings.NewReader(body))
	dom.Find("h5").Each(func(i int, selection *goquery.Selection) {
		text := selection.Text()
		if text == ugcHLHeader || text == ugc6sHeader || text == ugc4sHeader {
			//ugcRank := UGCRankNone
			selection.Next().ChildrenFiltered("li").Each(func(i int, selection *goquery.Selection) {
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
				seasons = append(seasons, Season{
					League:      "UGC",
					Division:    curRankStr,
					DivisionInt: curRank,
					Format:      format,
				})
			})
		}
	})
	return seasons, nil
}

func parseRankField(field string) (Division, string) {
	info := strings.Split(strings.Replace(field, "\n\n", "", -1), "\n")
	results := reUGCRank.FindStringSubmatch(info[0])
	if len(results) == 4 {
		switch results[3] {
		case "Platinum":
			return UGCRankPlatinum, "platinum"
		case "Gold":
			return UGCRankGold, "gold"
		case "Silver":
			return UGCRankSilver, "silver"
		case "Steel":
			return UGCRankSteel, "steel"
		case "Iron":
			return UGCRankIron, "iron"
		}
	}
	return UGCRankNone, ""
}
