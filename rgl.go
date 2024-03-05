package main

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/leighmacdonald/rgl"
	"github.com/leighmacdonald/steamid/v3/steamid"
	"github.com/pkg/errors"
)

// Current issues:
// - Sometime api just fails on the first attempt
// - Empty value.
func getRGL(ctx context.Context, log *slog.Logger, sid64 steamid.SID64) ([]Season, error) {
	startTime := time.Now()
	client := NewHTTPClient()
	_, errProfile := rgl.Profile(ctx, client, sid64)
	if errProfile != nil {
		return nil, errors.Wrap(errProfile, "Failed tp fetch profile")
	}

	teams, errTeams := rgl.ProfileTeams(ctx, client, sid64)
	if errTeams != nil {
		return nil, errors.Wrap(errTeams, "Failed to fetch teams")
	}

	seasons := make([]Season, len(teams))

	for index, team := range teams {
		seasonStartTime := time.Now()

		var season Season

		seasonInfo, errSeason := rgl.Season(ctx, client, team.SeasonID)
		if errSeason != nil {
			return nil, errors.Wrap(errSeason, "Failed to fetch seasons")
		}

		season.League = "rgl"
		season.Division = seasonInfo.Name
		season.DivisionInt = parseRGLDivision(team.DivisionName)
		season.TeamName = team.TeamName

		if seasonInfo.FormatName == "" {
			switch {
			case strings.Contains(strings.ToLower(seasonInfo.Name), "sixes"):
				seasonInfo.FormatName = "Sixes"
			case strings.Contains(strings.ToLower(seasonInfo.Name), "prolander"):
				seasonInfo.FormatName = "Prolander"
			case strings.Contains(strings.ToLower(seasonInfo.Name), "hl season"):
				seasonInfo.FormatName = "HL"
			case strings.Contains(strings.ToLower(seasonInfo.Name), "p7 season"):
				seasonInfo.FormatName = "Prolander"
			}
		}

		season.Format = seasonInfo.FormatName
		seasons[index] = season

		log.Info("RGL season fetched", slog.Duration("duration", time.Since(seasonStartTime)))
	}

	log.Info("RGL Completed", slog.Duration("duration", time.Since(startTime)))

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
