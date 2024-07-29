package main

import (
	"context"
	"errors"
	"github.com/leighmacdonald/bd-api/domain"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/leighmacdonald/steamweb/v2"
	"log/slog"
	"sort"
	"sync"
)

// loadProfiles concurrently loads data from all the tracked data source tables and assembles them into
// a slice of domain.Profile.
//
//nolint:cyclop
func loadProfiles(ctx context.Context, database *pgStore, cache cache, steamIDs steamid.Collection) ([]domain.Profile, error) { //nolint:funlen
	var ( //nolint:prealloc
		waitGroup   = &sync.WaitGroup{}
		summaries   []steamweb.PlayerSummary
		bans        []steamweb.PlayerBanState
		profiles    []domain.Profile
		logs        map[steamid.SteamID]int
		friends     friendMap
		bdEntries   []domain.BDSearchResult
		servemeBans []*domain.ServeMeRecord
		sourceBans  BanRecordMap
	)

	if len(steamIDs) > maxResults {
		return nil, errTooMany
	}

	localCtx, cancel := context.WithTimeout(ctx, apiTimeout)
	defer cancel()

	waitGroup.Add(1)

	go func() {
		defer waitGroup.Done()

		foundBDEntries, errBDSearch := database.botDetectorListSearch(localCtx, steamIDs, nil)
		if errBDSearch != nil {
			slog.Error("Failed to get bot detector records", ErrAttr(errBDSearch))
		}

		bdEntries = foundBDEntries
	}()

	waitGroup.Add(1)

	go func() {
		defer waitGroup.Done()

		sbRecords, errSB := database.sourcebansRecordBySID(localCtx, steamIDs)
		if errSB != nil {
			slog.Error("Failed to load sourcebans records", ErrAttr(errSB))
		}

		sourceBans = sbRecords
	}()

	waitGroup.Add(1)

	go func() {
		defer waitGroup.Done()

		sum, errSum := getSteamSummaries(localCtx, cache, steamIDs)
		if errSum != nil || len(sum) == 0 {
			slog.Error("Failed to load player summaries", ErrAttr(errSum))
		}

		summaries = sum
	}()

	waitGroup.Add(1)

	go func() {
		defer waitGroup.Done()

		banState, errBanState := getSteamBans(localCtx, cache, steamIDs)
		if errBanState != nil || len(banState) == 0 {
			slog.Error("Failed to load player ban states", ErrAttr(errBanState))
		}

		bans = banState
	}()

	waitGroup.Add(1)

	go func() {
		defer waitGroup.Done()

		friends = getSteamFriends(localCtx, cache, steamIDs)
	}()

	waitGroup.Add(1)

	go func() {
		defer waitGroup.Done()

		serveme, errs := database.servemeRecordsSearch(localCtx, steamIDs)
		if errs != nil && !errors.Is(errs, errDatabaseNoResults) {
			slog.Error("Failed to get serveme records")

			return
		}

		servemeBans = serveme
	}()

	waitGroup.Add(1)

	go func() {
		defer waitGroup.Done()

		logsVal, err := database.logsTFLogCount(localCtx, steamIDs)
		if err != nil {
			slog.Error("failed to query log counts", ErrAttr(err))

			return
		}

		logs = logsVal
	}()

	waitGroup.Wait()

	if len(steamIDs) == 0 || len(summaries) == 0 {
		return nil, errDatabaseNoResults
	}

	for _, sid := range steamIDs {
		profile := domain.Profile{
			SourceBans:  make([]domain.SbBanRecord, 0),
			ServeMe:     nil,
			LogsCount:   0,
			BotDetector: make([]domain.BDSearchResult, 0),
			Seasons:     make([]domain.Season, 0),
			Friends:     make([]steamweb.Friend, 0),
		}

		for _, entry := range bdEntries {
			if entry.Match.Steamid == sid.String() {
				profile.BotDetector = append(profile.BotDetector, entry)
			}
		}

		for _, summary := range summaries {
			if summary.SteamID == sid {
				profile.Summary = summary

				break
			}
		}

		for _, ban := range bans {
			if ban.SteamID == sid {
				profile.BanState = domain.PlayerBanState{
					SteamID:          ban.SteamID,
					CommunityBanned:  ban.CommunityBanned,
					VACBanned:        ban.VACBanned,
					NumberOfVACBans:  ban.NumberOfVACBans,
					DaysSinceLastBan: ban.DaysSinceLastBan,
					NumberOfGameBans: ban.NumberOfGameBans,
					EconomyBan:       ban.EconomyBan,
				}

				break
			}
		}

		for _, serveme := range servemeBans {
			if serveme.SteamID == sid {
				profile.ServeMe = serveme

				break
			}
		}

		for logSID, count := range logs {
			if logSID == sid {
				profile.LogsCount = count

				break
			}
		}

		if records, ok := sourceBans[sid.String()]; ok {
			profile.SourceBans = records
		} else {
			// Dont return null json values
			profile.SourceBans = []domain.SbBanRecord{}
		}

		if friendsList, ok := friends[sid.String()]; ok {
			profile.Friends = friendsList
		} else {
			profile.Friends = []steamweb.Friend{}
		}

		profile.Seasons = []domain.Season{}
		sort.Slice(profile.Seasons, func(i, j int) bool {
			return profile.Seasons[i].DivisionInt < profile.Seasons[j].DivisionInt
		})
		profiles = append(profiles, profile)
	}

	return profiles, nil
}
