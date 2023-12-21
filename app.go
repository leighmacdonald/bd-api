package main

import (
	"context"
	"fmt"
	"github.com/leighmacdonald/etf2l"
	"github.com/leighmacdonald/rgl"
	"github.com/leighmacdonald/steamid/v3/steamid"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/bd-api/models"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

type App struct {
	config   appConfig
	db       *pgStore
	log      *zap.Logger
	cache    cache
	scrapers []*sbScraper
	pm       *proxyManager
	router   *gin.Engine
	etf2l    *etf2l.Client
}

func NewApp(logger *zap.Logger, config appConfig, database *pgStore, cache cache, proxyManager *proxyManager) *App {
	application := &App{
		config:   config,
		log:      logger,
		db:       database,
		cache:    cache,
		pm:       proxyManager,
		router:   nil,
		scrapers: []*sbScraper{},
		etf2l:    etf2l.New(),
	}

	router, errRouter := application.createRouter()
	if errRouter != nil {
		logger.Fatal("Failed to create router", zap.Error(errRouter))
	}

	application.router = router

	return application
}

func (a *App) Start(ctx context.Context) error {
	if a.config.SourcebansScraperEnabled {
		if errInitScrapers := a.initScrapers(ctx); errInitScrapers != nil {
			return errInitScrapers
		}

		go a.startScrapers(ctx)
	}

	go a.etf2lUpdater(ctx)
	go a.rglUpdater(ctx)

	go a.profileUpdater(ctx)

	return a.startAPI(ctx, a.config.ListenAddr)
}

func (a *App) initScrapers(ctx context.Context) error {
	scrapers, errScrapers := createScrapers(a.log, a.config.CacheDir)
	if errScrapers != nil {
		return errScrapers
	}

	for _, scraper := range scrapers {
		// Attach a site_id to the scraper, so we can keep track of the scrape source
		var s models.SbSite
		if errSave := sbSiteGetOrCreate(ctx, a.db, scraper.name, &s); errSave != nil {
			return errors.Wrap(errSave, "Database error")
		}

		scraper.ID = uint32(s.SiteID)
	}

	a.scrapers = scrapers

	return nil
}

func (a *App) etf2lUpdater(ctx context.Context) {
	ticker := time.NewTicker(time.Hour * 72)
	update := make(chan bool)

	go func() {
		update <- true
	}()
	for {
		select {
		case <-update:
			a.scrapeETF2L(ctx)
		case <-ticker.C:
			update <- true
		case <-ctx.Done():
			return
		}
	}
}

func (a *App) rglUpdater(ctx context.Context) {
	ticker := time.NewTicker(time.Hour * 72)
	update := make(chan bool)

	go func() {
		update <- true
	}()
	for {
		select {
		case <-update:
			a.scrapeRGL(ctx)
		case <-ticker.C:
			update <- true
		case <-ctx.Done():
			return
		}
	}
}
func wait() {
	time.Sleep(time.Second * 2)
}

func (a *App) nextHTTPClient() *http.Client {
	if !a.config.ProxiesEnabled {
		return &http.Client{Timeout: time.Second * 10}
	}

	if len(a.pm.proxies) == 0 {
		a.log.Warn("Not using proxy as none are configured")
		return &http.Client{Timeout: time.Second * 10}
	}

	return a.pm.next()
}

func (a *App) updateETF2LPlayers(ctx context.Context) {
	logger := a.log.Named("etf2l")
	playerErrLimit := 10000
	playerID := 0
	playerErr := 0
	for {
		playerID++

		var existingPlayer ETF2LPlayer
		if errExisting := etf2lPlayerByID(ctx, a.db, playerID, &existingPlayer); errExisting != nil {
			if !errors.Is(errExisting, ErrNoResult) {
				logger.Error("Failed to load player", zap.Error(errExisting))
			}
		} else {
			// Skip if we already know it
			continue
		}

		player, errPlayer := a.etf2l.Player(ctx, a.pm.next(), fmt.Sprintf("%d", playerID))
		if errPlayer != nil {
			logger.Debug("Player fetch error", zap.Error(errPlayer))
			playerErr++
			if playerErr >= playerErrLimit {
				logger.Info("Bailing on player update, reached error limit", zap.Int("limit", playerErrLimit))
				break
			}

			continue
		}

		if !player.Steam.ID64.Valid() {
			logger.Warn("Skipping invalid player")

			continue
		}

		newPlayer := newPlayerRecord(player.Steam.ID64)
		if err := playerGetOrCreate(ctx, a.db, player.Steam.ID64, &newPlayer); err != nil {
			logger.Error("Failed to create parent player record", zap.Error(err))

			return

		}

		if err := etf2lSavePlayer(ctx, a.db, player); err != nil {
			logger.Error("Failed to save player", zap.Error(err))

			return
		}

		logger.Info("New player created",
			zap.Int("player_id", player.ID), zap.Int64("steam_id", player.Steam.ID64.Int64()))
	}
}

func (a *App) updateRGLBans(ctx context.Context) {
	logger := a.log.Named("rgl.bans")
	offset := 0
	limit := 100
	inserted := 0

	for {
		bans, errBans := rgl.Bans(ctx, a.nextHTTPClient(), limit, offset)
		if errBans != nil {
			logger.Error("Failed to fetch next bans", zap.Error(errBans), zap.Int("limit", limit), zap.Int("offset", offset))

			return
		}

		if len(bans) == 0 {
			break
		}

		for _, ban := range bans {
			sid := steamid.New(ban.SteamID)
			if !sid.Valid() {
				sid = steamid.New(strings.Trim(ban.SteamID, ")"))
				if !sid.Valid() {
					logger.Warn("Received invalid steam_id", zap.String("steam_id", ban.SteamID))

					continue
				}

				ban.SteamID = sid.String()
			}

			newPlayer := newPlayerRecord(sid)
			if err := playerGetOrCreate(ctx, a.db, newPlayer.SteamID, &newPlayer); err != nil {
				logger.Error("Failed to create parent player record", zap.Error(err))

				return

			}

			if err := rglBanSave(ctx, a.db, ban); err != nil {
				logger.Error("Failed to save ban", zap.Error(err),
					zap.Int64("steam_id", steamid.SID64(ban.SteamID).Int64()))

				return
			}

			inserted++
		}

		logger.Info("Updated ban records", zap.Int("inserted", inserted))

		offset += limit

		wait()
	}
}

func (a *App) updateRGLTeams(ctx context.Context) {
	logger := a.log.Named("rgl.teams")

	teamErrLimit := 5000
	teamID := 0
	teamErr := 0
	for {
		teamID++
		var team *rgl.TeamOverview
		var existingTeam RGLTeam
		if errExisting := rglTeam(ctx, a.db, teamID, &existingTeam); errExisting != nil {
			if !errors.Is(errExisting, ErrNoResult) {
				logger.Error("Failed to load team", zap.Error(errExisting))
			}

			newTeam, errTeam := rgl.Team(ctx, a.pm.next(), int64(teamID))
			if errTeam != nil {
				a.log.Debug("Team fetch error", zap.Error(errTeam))
				teamErr++
				if teamErr >= teamErrLimit {
					logger.Info("Bailing on team update, reached error limit", zap.Int("limit", teamErrLimit))
					break
				}

				continue
			}

			if err := rglTeamSave(ctx, a.db, newTeam); err != nil {
				logger.Error("Failed to save new rgl team record", zap.Error(err))

				return
			}

			team = newTeam

			wait()
		}

		if team == nil {
			continue
		}

		for _, teamPlayer := range team.Players {
			newPlayer := newPlayerRecord(teamPlayer.SteamID)
			if err := playerGetOrCreate(ctx, a.db, newPlayer.SteamID, &newPlayer); err != nil {
				logger.Error("Failed to create parent team record", zap.Error(err))

				return

			}
			var player RGLPlayer
			if errRGLPlayer := rglPlayer(ctx, a.db, teamPlayer.SteamID, &player); errRGLPlayer != nil {
				if !errors.Is(errRGLPlayer, ErrNoResult) {
					logger.Error("Failed to fetch rgl player", zap.Error(errRGLPlayer))

					return
				}

				p, errP := rgl.Profile(ctx, a.nextHTTPClient(), teamPlayer.SteamID)
				if errP != nil {
					logger.Error("Failed to fetch rgl profile", zap.Error(errRGLPlayer))

					continue
				}

				if errSave := rglPlayerSave(ctx, a.db, p); errSave != nil {
					logger.Error("Failed to save rgl profile", zap.Error(errRGLPlayer))

					return
				}
				wait()
			}

			if err := rglTeamPlayerSave(ctx, a.db, team.TeamID, teamPlayer); err != nil {
				logger.Error("Failed to same team player", zap.Error(err))

				return
			}
		}

		logger.Info("New team created", zap.Int("team_id", team.TeamID))
	}
}

func (a *App) updateCompetitions(ctx context.Context) {
	compErrLimit := 100
	compID := 0
	compErr := 0
	for {
		compID++

		results, errResults := a.etf2l.CompetitionResults(ctx, a.pm.next(), compID, etf2l.BaseOpts{Recursive: true})
		if errResults != nil {
			compErr++
			if compErr == compErrLimit {
				break
			}

		}
		a.log.Info("Fetched competition results", zap.Int("competition_id", compID))

		wait()

		for _, result := range results {
			for _, teamID := range []int{result.Clan1.ID, result.Clan2.ID} {
				_, errExisting := etf2lTeam(ctx, a.db, teamID)
				if errExisting != nil {
					if !errors.Is(errExisting, ErrNoResult) {
						compErr++
						if compErr == compErrLimit {
							return
						}

						continue
					}

					a.log.Info("New team saved successfully", zap.Int("team_id", teamID))

					team, errTeam := a.etf2l.Team(ctx, a.pm.next(), teamID)
					if errTeam != nil {
						compErr++
						if compErr == compErrLimit {
							return
						}

						continue
					}

					if errSave := etf2lSaveTeam(ctx, a.db, team); errSave != nil {
						a.log.Error("Failed to save team", zap.Error(errSave))

						return
					}

					for _, p := range team.Players {
						if errTPS := etf2lTeamPlayerSave(ctx, a.db, p.Steam.ID64, p.ID); errTPS != nil {
							a.log.Error("Failed to save team player", zap.Error(errTPS))

							continue
						}
					}

					wait()
				} else {
					a.log.Info("Skipping existing team", zap.Int("team_id", teamID))
				}
			}

		}
	}
}

func (a *App) scrapeRGL(ctx context.Context) {
	a.updateRGLBans(ctx)
	a.updateRGLTeams(ctx)
	//a.updateCompetitions(ctx)
}

func (a *App) scrapeETF2L(ctx context.Context) {
	a.updateETF2LPlayers(ctx)
	a.updateCompetitions(ctx)
}

//func updateETF2LTeam(ctx context.Context, client *etf2l.Client, db *pgStore, teamID int) error {
//	_, errTeam := etf2lTeam(ctx, db, teamID)
//	if errTeam != nil {
//		if errors.Is(errTeam, ErrNoResult) {
//			newTeam, errNewTeam := client.Team(ctx, teamID)
//			if errNewTeam != nil {
//				return errNewTeam
//			}
//
//			if errBan := etf2lSaveTeam(ctx, db, newTeam); errBan != nil {
//				return errBan
//			}
//
//			for teamIDString, division := range newTeam.Competitions {
//				competitionID, errConv := strconv.ParseInt(teamIDString, 10, 32)
//				if errConv != nil {
//					return errConv
//				}
//
//				if errC := updateETF2LCompetition(ctx, client, db, int(competitionID), division); errC != nil {
//					return errC
//				}
//			}
//		} else {
//			return errTeam
//		}
//	}
//
//	return nil
//}
//
//func updateETF2LCompetition(ctx context.Context, client *etf2l.Client, db *pgStore, competitionID int, division etf2l.TeamCompetition) error {
//	var comp ETF2LCompetition
//	if errComp := etf2lCompetition(ctx, db, competitionID, &comp); errComp != nil {
//		if !errors.Is(errComp, ErrNoResult) {
//			return errors.Wrap(errComp, "failed to fetch competition")
//		}
//
//		teamComps, errTeamComps := client.CompetitionTables(ctx, competitionID)
//		if errTeamComps != nil {
//			return errTeamComps
//		}
//
//		for _, competition := range teamComps {
//			teamDetails, errDetails := client.Team(ctx, competition.TeamID)
//			if errDetails != nil {
//				return errDetails
//			}
//
//			etf2lSaveCompetition()
//		}
//
//	}
//
//	competition, errComp := etf2lCompetitions(ctx, db, teamID)
//	if errComp != nil {
//		if !errors.Is(errComp, ErrNoResult) {
//			return errors.Wrap(errComp, "failed to fetch competition")
//		}
//
//		if errCompSave := etf2lSaveCompetition(ctx, db, int(competitionID), team.TeamID, comp); errCompSave != nil {
//			return errCompSave
//		}
//	}
//
//	return nil
//}

//func updateETF2LPlayer(ctx context.Context, client *etf2l.Client, db *pgStore, steamID steamid.SID64) error {
//	player, err := client.Player(ctx, steamID.String())
//	if err != nil {
//		return err
//	}
//
//	// Make sure this exists first as its a foreign key parent to the rest
//	if errPlayer := etf2lSavePlayer(ctx, db, player); errPlayer != nil {
//		return errPlayer
//	}
//
//	for _, ban := range player.Bans {
//		if errBan := etf2lSaveBan(ctx, db, player.Steam.ID64, ban); errBan != nil {
//			return errBan
//		}
//	}
//
//	for _, playerTeam := range player.Teams {
//		//if errTeam := updateETF2LTeam(ctx, client, db, playerTeam.ID); errTeam != nil {
//		//	return errTeam
//		//}
//
//		if errTP := etf2lTeamPlayerSave(ctx, db, player.Steam.ID64, playerTeam.ID); errTP != nil {
//			return errTP
//		}
//	}
//
//	return nil
//}

func (a *App) updateRGLPlayer(ctx context.Context, steamID steamid.SID64) error {
	return nil
}
