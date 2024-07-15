package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/leighmacdonald/bd-api/domain"
	"github.com/leighmacdonald/rgl"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/leighmacdonald/steamweb/v2"
)

var (
	errSteamBanFetch      = errors.New("failed to fetch steam ban state")
	errSteamBanDecode     = errors.New("failed to decode steam ban state")
	errSteamSummaryFetch  = errors.New("failed to fetch steam summary")
	errSteamSummaryDecode = errors.New("failed to decode steam summary")
)

func steamIDCollectionToInt64Slice(collection steamid.Collection) []int64 {
	ids := make([]int64, len(collection))
	for idx := range collection {
		ids[idx] = collection[idx].Int64()
	}

	return ids
}

type friendMap map[string][]steamweb.Friend

func getSteamFriends(ctx context.Context, cache cache, steamIDs steamid.Collection) friendMap {
	var (
		mutex     = sync.RWMutex{}
		output    = make(friendMap)
		waitGroup = &sync.WaitGroup{}
	)

	for _, sid := range steamIDs {
		output[sid.String()] = []steamweb.Friend{}
	}

	for _, currentID := range steamIDs {
		waitGroup.Add(1)

		go func(steamID steamid.SteamID) {
			defer waitGroup.Done()

			var (
				friends []steamweb.Friend
				key     = makeKey(KeyFriends, steamID)
			)

			friendsBody, errCache := cache.get(key)

			if errCache == nil {
				if err := json.Unmarshal(friendsBody, &friends); err != nil {
					slog.Error("Failed to unmarshal cached result", ErrAttr(err))
				}

				return
			}

			newFriends, errFriends := steamweb.GetFriendList(ctx, steamID)
			if errFriends != nil {
				// 401 = Friends list is not public
				if !strings.Contains(errFriends.Error(), "401") {
					slog.Warn("Failed to fetch friends", ErrAttr(errFriends))
				}

				return
			}

			if newFriends == nil {
				newFriends = []steamweb.Friend{}
			}

			body, errMarshal := json.Marshal(newFriends)
			if errMarshal != nil {
				slog.Error("Failed to unmarshal friends", ErrAttr(errMarshal))

				return
			}

			if errSet := cache.set(key, bytes.NewReader(body)); errSet != nil {
				slog.Error("Failed to update cache", ErrAttr(errSet), slog.String("site", "ugc"))
			}

			mutex.Lock()
			output[steamID.String()] = newFriends
			mutex.Unlock()
		}(currentID)
	}

	waitGroup.Wait()

	return output
}

func getSteamBans(ctx context.Context, cache cache, steamIDs steamid.Collection) ([]steamweb.PlayerBanState, error) {
	var (
		banStates []steamweb.PlayerBanState
		missed    steamid.Collection
	)

	for _, steamID := range steamIDs {
		var banState steamweb.PlayerBanState
		summaryBody, errCache := cache.get(makeKey(KeyBans, steamID))
		if errCache == nil {
			if err := json.Unmarshal(summaryBody, &banState); err != nil {
				slog.Error("Failed to unmarshal cached result", ErrAttr(err))
			}
		}

		if banState.SteamID.Valid() {
			banStates = append(banStates, banState)
		} else {
			missed = append(missed, steamID)
		}
	}

	if len(missed) > 0 {
		newBans, errBans := steamweb.GetPlayerBans(ctx, missed)
		if errBans != nil {
			return nil, errors.Join(errBans, errSteamBanFetch)
		}

		for _, ban := range newBans {
			body, errMarshal := json.Marshal(ban)
			if errMarshal != nil {
				return nil, errors.Join(errMarshal, errSteamBanDecode)
			}

			if errSet := cache.set(makeKey(KeyBans, ban.SteamID), bytes.NewReader(body)); errSet != nil {
				slog.Error("Failed to update cache", ErrAttr(errSet))
			}
		}

		banStates = append(banStates, newBans...)
	}

	return banStates, nil
}

func getCompHistory(ctx context.Context, cache cache, steamIDs steamid.Collection) domain.CompMap {
	var (
		results   = domain.CompMap{}
		missed    steamid.Collection
		startTime = time.Now()
	)

	missed = append(missed, steamIDs...)

	// for _, steamID := range steamIDs { //nolint:gosimple
	//	// var seasons []Season
	//	missed = append(missed, steamID)
	//
	//	// seasonsBody, errCache := a.cache.get(makeKey(KeyRGL, steamID))
	//	// if errCache != nil {
	//	//	if errors.Is(errCache, errCacheExpired) {
	//	//		missed = append(missed, steamID)
	//	//
	//	//		continue
	//	//	}
	//	// }
	//	//
	//	// if errUnmarshal := json.Unmarshal(seasonsBody, &seasons); errUnmarshal != nil {
	//	//	slog.Error("Failed to unmarshal cached result", ErrAttr(errUnmarshal))
	//	//
	//	//	missed = append(missed, steamID)
	//	//
	//	//	continue
	//	// }
	//	//
	//	// results[steamID] = append(results[steamID], seasons...)
	// }

	for _, steamID := range missed {
		logger := slog.With(slog.String("site", "rgl"), slog.String("steam_id", steamID.String()))
		rglSeasons, errRGL := getRGL(ctx, logger, steamID)

		if errRGL != nil {
			if errors.Is(errRGL, rgl.ErrRateLimit) {
				logger.Warn("API Rate limited")
			} else {
				logger.Error("Failed to fetch rgl data", ErrAttr(errRGL))
			}

			continue
		}

		body, errMarshal := json.Marshal(rglSeasons)
		if errMarshal != nil {
			logger.Error("Failed to marshal rgl data", ErrAttr(errRGL))

			continue
		}

		if errSet := cache.set(makeKey(KeyRGL, steamID), bytes.NewReader(body)); errSet != nil {
			logger.Error("Failed to update cache", ErrAttr(errSet))
		}

		results[steamID] = append(results[steamID], rglSeasons...)
	}

	slog.Debug("RGL Query time", slog.Duration("duration", time.Since(startTime)))

	return results
}

func getSteamSummaries(ctx context.Context, cache cache, steamIDs steamid.Collection) ([]steamweb.PlayerSummary, error) {
	var (
		summaries []steamweb.PlayerSummary
		missed    steamid.Collection
	)

	for _, steamID := range steamIDs {
		var summary steamweb.PlayerSummary
		summaryBody, errCache := cache.get(makeKey(KeySummary, steamID))

		if errCache == nil {
			if err := json.Unmarshal(summaryBody, &summary); err != nil {
				slog.Error("Failed to unmarshal cached result", ErrAttr(err))
			}
		}

		if summary.SteamID.Valid() {
			summaries = append(summaries, summary)
		} else {
			missed = append(missed, steamID)
		}
	}

	if len(missed) > 0 {
		newSummaries, errSummaries := steamweb.PlayerSummaries(ctx, missed)
		if errSummaries != nil {
			return nil, errors.Join(errSummaries, errSteamSummaryFetch)
		}

		for _, summary := range newSummaries {
			body, errMarshal := json.Marshal(summary)
			if errMarshal != nil {
				return nil, errors.Join(errMarshal, errSteamSummaryDecode)
			}

			if errSet := cache.set(makeKey(KeySummary, summary.SteamID), bytes.NewReader(body)); errSet != nil {
				slog.Error("Failed to update cache", ErrAttr(errSet))
			}
		}

		summaries = append(summaries, newSummaries...)
	}

	return summaries, nil
}

func profileUpdater(ctx context.Context, database *pgStore) {
	const (
		maxQueuedCount = 100
		updateInterval = time.Second
	)

	var (
		updateQueue   steamid.Collection
		updateTicker  = time.NewTicker(updateInterval)
		triggerUpdate = make(chan any)
	)

	for {
		select {
		case <-updateTicker.C:
			triggerUpdate <- true
		case <-triggerUpdate:
			var expiredIDs steamid.Collection
			expiredProfiles, errProfiles := database.playerGetExpiredProfiles(ctx, maxQueuedCount)
			if errProfiles != nil {
				slog.Error("Failed to fetch expired profiles", ErrAttr(errProfiles))
			}

			additional := 0

			for len(expiredProfiles) < maxQueuedCount {
				for _, sid64 := range updateQueue {
					var pr PlayerRecord
					if errQueued := database.playerGetOrCreate(ctx, sid64, &pr); errQueued != nil {
						continue
					}

					expiredProfiles = append(expiredProfiles, pr)
					additional++
				}
			}

			updateQueue = updateQueue[additional:]

			if len(expiredProfiles) == 0 {
				continue
			}

			for _, profile := range expiredProfiles {
				expiredIDs = append(expiredIDs, profile.SteamID)
			}

			summaries, errSum := steamweb.PlayerSummaries(ctx, expiredIDs)
			if errSum != nil {
				slog.Error("Failed to fetch summaries", ErrAttr(errSum))

				continue
			}

			bans, errBans := steamweb.GetPlayerBans(ctx, expiredIDs)
			if errBans != nil {
				slog.Error("Failed to fetch bans", ErrAttr(errSum))

				continue
			}

			for _, profile := range expiredProfiles {
				prof := profile
				for _, sum := range summaries {
					if sum.SteamID.Int64() == prof.SteamID.Int64() {
						prof.applySummary(sum)

						break
					}
				}

				for _, ban := range bans {
					if ban.SteamID.Int64() == prof.SteamID.Int64() {
						prof.applyBans(ban)

						break
					}
				}

				if errSave := database.playerRecordSave(ctx, &prof); errSave != nil {
					slog.Error("Failed to update profile", slog.Int64("sid", prof.SteamID.Int64()), ErrAttr(errSave))
				}
			}
		case <-ctx.Done():
			return
		}
	}
}
