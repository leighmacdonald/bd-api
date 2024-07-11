package main

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/leighmacdonald/bd-api/domain"
	"github.com/leighmacdonald/rgl"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

// Current issues:

// - Sometime api just fails on the first attempt

// - Empty value.

func getRGL(ctx context.Context, log *slog.Logger, sid64 steamid.SteamID) ([]domain.Season, error) {
	startTime := time.Now()
	client := NewHTTPClient()

	_, errProfile := rgl.Profile(ctx, client, sid64)
	if errProfile != nil {
		return nil, errors.Join(errProfile, errRequestPerform)
	}

	teams, errTeams := rgl.ProfileTeams(ctx, client, sid64)
	if errTeams != nil {
		return nil, errRequestPerform
	}

	seasons := make([]domain.Season, len(teams))

	for index, team := range teams {
		seasonStartTime := time.Now()

		var season domain.Season
		seasonInfo, errSeason := rgl.Season(ctx, client, team.SeasonID)

		if errSeason != nil {
			return nil, errRequestPerform
		}

		season.League = "rgl"
		season.Division = seasonInfo.Name
		season.DivisionInt = parseRGLDivision(team.DivisionName)
		season.TeamName = team.TeamName

		lowerName := strings.ToLower(seasonInfo.Name)

		if seasonInfo.FormatName == "" {
			switch {
			case strings.Contains(lowerName, "sixes"):

				seasonInfo.FormatName = "Sixes"
			case strings.Contains(lowerName, "prolander"):

				seasonInfo.FormatName = "Prolander"
			case strings.Contains(lowerName, "hl season"):

				seasonInfo.FormatName = "HL"
			case strings.Contains(lowerName, "p7 season"):

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

func parseRGLDivision(div string) domain.Division {
	switch div {
	case "RGL-Invite":
		fallthrough
	case "Invite":
		return domain.RGLRankInvite
	case "RGL-Div-1":
		return domain.RGLRankDiv1
	case "RGL-Div-2":
		return domain.RGLRankDiv2
	case "RGL-Main":
		fallthrough
	case "Main":
		return domain.RGLRankMain
	case "RGL-Advanced":
		fallthrough
	case "Advanced-1":
		fallthrough
	case "Advanced":
		return domain.RGLRankAdvanced
	case "RGL-Intermediate":
		fallthrough
	case "Intermediate":
		return domain.RGLRankIntermediate
	case "RGL-Challenger":
		return domain.RGLRankIntermediate
	case "Open":
		return domain.RGLRankOpen
	case "Amateur":
		return domain.RGLRankAmateur
	case "Fresh Meat":
		return domain.RGLRankFreshMeat
	default:
		return domain.RGLRankNone
	}
}
