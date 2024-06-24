package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"

	"github.com/leighmacdonald/bd-api/domain"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

type comp struct {
	Category    string `json:"category"`
	Competition string `json:"competition"`
	Division    struct {
		Name string      `json:"name"`
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

func sortSeasons(seasons []domain.Season) []domain.Season {
	sort.Slice(seasons, func(i, j int) bool {
		return seasons[i].DivisionInt < seasons[j].DivisionInt
	})

	return seasons
}

func getETF2L(ctx context.Context, sid steamid.SteamID) ([]domain.Season, error) {
	url := fmt.Sprintf("https://api.etf2l.org/player/%d.json", sid.Int64())

	var player etf2lPlayer

	resp, errGet := get(ctx, url, nil)
	if errGet != nil {
		return nil, errGet
	}

	defer logCloser(resp.Body)

	if errUnmarshal := json.NewDecoder(resp.Body).Decode(&player); errUnmarshal != nil {
		return nil, errors.Join(errUnmarshal, errResponseDecode)
	}

	return parseETF2L(player), nil
}

func parseETF2L(player etf2lPlayer) []domain.Season {
	var seasons []domain.Season

	for _, team := range player.Player.Teams {
		for _, competition := range team.Competitions {
			var (
				div    = domain.UnknownDivision
				divStr = competition.Competition
				format = "N/A"
			)

			if competition.Division.Name != "" {
				switch competition.Division.Name {
				case "Open":
					div = domain.ETF2LOpen
					divStr = "Open"
				case "Mid":
					div = domain.ETF2LMid
					divStr = "Mid"
				case "Division 4":
					div = domain.ETF2LLow
					divStr = "Low"
				case "Division 3":
					div = domain.ETF2LMid
					divStr = "Div 3"
				case "Division 2":
					div = domain.ETF2LDiv2
					divStr = "Div 2"
				case "Division 1":
					div = domain.ETF2LDiv1
					divStr = "Div 1"
				case "Premiership":
					div = domain.ETF2LPremiership
					divStr = "Premiership"
				}
			}

			switch team.Type {
			case "Highlander":

				format = "Highlander"
			case "6on6":

				format = "6s"
			}

			seasons = append(seasons, domain.Season{
				League:      "ETF2L",
				Division:    divStr,
				DivisionInt: div,
				Format:      format,
				TeamName:    team.Name,
			})
		}
	}

	return sortSeasons(seasons)
}
