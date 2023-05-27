package main

import (
	"context"
	"github.com/leighmacdonald/rgl"
	"github.com/leighmacdonald/steamid/v2/steamid"
)

func getRGL(ctx context.Context, sid64 steamid.SID64) ([]Season, error) {
	_, errProfile := rgl.Profile(ctx, sid64)
	if errProfile != nil {
		return nil, errProfile
	}
	teams, errTeams := rgl.ProfileTeams(ctx, sid64)
	if errTeams != nil {
		return nil, errTeams
	}
	var seasons []Season
	for _, team := range teams {
		var season Season
		seasonInfo, errSeason := rgl.Season(ctx, team.SeasonId)
		if errSeason != nil {
			return nil, errSeason
		}
		season.League = "rgl"
		season.Division = seasonInfo.Name
		season.DivisionInt = parseRGLDivision(team.DivisionName)
		season.TeamName = team.TeamName
		season.Format = seasonInfo.FormatName
		seasons = append(seasons, season)
	}
	return seasons, nil
}

func parseRGLDivision(div string) Division {

	switch div {
	case "RGL-Invite":
		fallthrough
	case "Invite":
		return RGLRankInvite
	case "RGL-Div-1":
		return RGLRankDiv1
	case "RGL-Div-2":
		return RGLRankDiv2
	case "RGL-Main":
		fallthrough
	case "Main":
		return RGLRankMain
	case "RGL-Advanced":
		fallthrough
	case "Advanced-1":
		fallthrough
	case "Advanced":
		return RGLRankAdvanced
	case "RGL-Intermediate":
		fallthrough
	case "Intermediate":
		return RGLRankIntermediate
	case "RGL-Challenger":
		return RGLRankIntermediate
	case "Open":
		return RGLRankOpen
	case "Amateur":
		return RGLRankAmateur
	case "Fresh Meat":
		return RGLRankFreshMeat
	default:
		return RGLRankNone
	}

}