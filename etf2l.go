package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/pkg/errors"
	"io"
	"regexp"
	"sort"
	"strings"
)

type comp struct {
	Category    string `json:"category"`
	Competition string `json:"competition"`
	Division    struct {
		Name interface{} `json:"name"`
		Tier interface{} `json:"tier"`
	} `json:"division"`
	URL string `json:"url"`
}

type etf2lPlayer struct {
	Player struct {
		Bans       interface{} `json:"bans"`
		Classes    []string    `json:"classes"`
		Country    string      `json:"country"`
		ID         int         `json:"id"`
		Name       string      `json:"name"`
		Registered int         `json:"registered"`
		Steam      struct {
			Avatar string `json:"avatar"`
			ID     string `json:"id"`
			ID3    string `json:"id3"`
			ID64   string `json:"id64"`
		} `json:"steam"`
		Teams []struct {
			Competitions map[string]comp `json:"competitions,omitempty"`
			Country      string          `json:"country"`
			Homepage     string          `json:"homepage"`
			ID           int             `json:"id"`
			Irc          struct {
				Channel interface{} `json:"channel"`
				Network interface{} `json:"network"`
			} `json:"irc"`
			Name   string `json:"name"`
			Server string `json:"server"`
			Steam  struct {
				Avatar string `json:"avatar"`
				Group  string `json:"group"`
			} `json:"steam"`
			Tag  string `json:"tag"`
			Type string `json:"type"`
			Urls struct {
				Matches   string `json:"matches"`
				Results   string `json:"results"`
				Self      string `json:"self"`
				Transfers string `json:"transfers"`
			} `json:"urls"`
		} `json:"teams"`
		Title string `json:"title"`
		Urls  struct {
			Results   string `json:"results"`
			Self      string `json:"self"`
			Transfers string `json:"transfers"`
		} `json:"urls"`
	} `json:"player"`
	Status struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"status"`
}

func sortSeasons(s []Season) []Season {
	sort.Slice(s, func(i, j int) bool {
		return s[i].DivisionInt < s[j].DivisionInt
	})
	return s
}

func getETF2L(ctx context.Context, sid steamid.SID64) ([]Season, error) {
	url := fmt.Sprintf("https://api.etf2l.org/player/%d", sid.Int64())
	var player etf2lPlayer
	resp, errGet := get(ctx, url, nil)
	if errGet != nil {
		return nil, errGet
	}
	b, errRead := io.ReadAll(resp.Body)
	if errRead != nil {
		return nil, errRead
	}
	rx := regexp.MustCompile(`<div id="source" style="display:none;">(.+?)</div>`)
	m := rx.FindStringSubmatch(strings.ReplaceAll(string(b), "\n", ""))
	if len(m) != 2 {
		return nil, errors.New("Failed to find match")
	}
	b = []byte(m[1])
	if errUnmarshal := json.Unmarshal(b, &player); errUnmarshal != nil {
		return nil, errUnmarshal
	}
	seasons, errParse := parseETF2L(player)
	if errParse != nil {
		return nil, errParse
	}
	return seasons, nil
}

func parseETF2L(player etf2lPlayer) ([]Season, error) {
	var seasons []Season
	for _, team := range player.Player.Teams {
		for _, comp := range team.Competitions {
			if comp.Division.Tier == nil {
				continue
			}
			var (
				div    Division
				divStr string
				format string
			)
			switch comp.Division.Name {
			case "Open":
				div = ETF2LOpen
				divStr = "Open"
			case "Mid":
				div = ETF2LMid
				divStr = "Mid"
			case "Division 4":
				div = ETF2LLow
				divStr = "Low"
			case "Division 3":
				div = ETF2LMid
				divStr = "Div 3"
			case "Division 2":
				div = ETF2LDiv2
				divStr = "Div 2"
			case "Division 1":
				div = ETF2LDiv1
				divStr = "Div 1"
			case "Premiership":
				div = ETF2LPremiership
				divStr = "Premiership"
			default:
				fmt.Printf("Unknown etf2l div: %s\n", comp.Division.Name)
			}
			switch team.Type {
			case "Highlander":
				format = "Highlander"
			case "6on6":
				format = "6s"
			default:
				//log.Printf("Unknown etf2l format: %s\n", team.Type)
				continue
			}
			seasons = append(seasons, Season{
				League:      "ETF2L",
				Division:    divStr,
				DivisionInt: div,
				Format:      format,
			})
		}
	}
	return sortSeasons(seasons), nil
}
