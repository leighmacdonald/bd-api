package main

import (
	"github.com/leighmacdonald/steamid/v3/steamid"
	"sort"
	"time"
)

type TimeStamped struct {
	CreatedOn time.Time `json:"created_on"`
	UpdatedOn time.Time `json:"updated_on"`
}

func newTimeStamped() TimeStamped {
	t0 := time.Now()
	return TimeStamped{UpdatedOn: t0, CreatedOn: t0}
}

type ETF2LPlayer struct {
	SteamID steamid.SID64 `json:"steam_id"`
	ID      int           `json:"id"`
	Name    string        `json:"name"`
	Country string        `json:"country"`
	TimeStamped
}

type ETF2LBan struct {
	SteamID   steamid.SID64 `json:"steam_id"`
	StartDate time.Time     `json:"start_date"`
	EndDate   time.Time     `json:"end_date"`
	Reason    string        `json:"reason"`
	TimeStamped
}

type ETF2LTeam struct {
	TeamID   int           `json:"team_id"`
	SteamID  steamid.SID64 `json:"steam_id"`
	ID       int           `json:"id"`
	Name     string        `json:"name"`
	Title    string        `json:"title"`
	Country  string        `json:"country"`
	TeamType string        `json:"teamType"`
	TimeStamped
}

type ETF2LCompetition struct {
	CompetitionID int    `json:"competition_id"`
	TeamID        int    `json:"team_id"`
	Category      string `json:"category"`
	Competition   string `json:"competition"`
	DivisionName  string `json:"division_name"`
	DivisionTier  int    `json:"division_tier"`
	TimeStamped
}

func sortSeasons(seasons []Season) []Season {
	sort.Slice(seasons, func(i, j int) bool {
		return seasons[i].DivisionInt < seasons[j].DivisionInt
	})

	return seasons
}
