package main

import (
	"bytes"
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/leighmacdonald/steamid/v3/steamid"
	"github.com/leighmacdonald/steamweb/v2"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

type friendMap map[steamid.SID64][]steamweb.Friend

func (a *App) getSteamFriends(ctx context.Context, steamIDs steamid.Collection) friendMap {
	var (
		mutex     = sync.RWMutex{}
		output    = make(friendMap)
		waitGroup = &sync.WaitGroup{}
	)

	for _, sid := range steamIDs {
		output[sid] = []steamweb.Friend{}
	}

	for _, currentID := range steamIDs {
		waitGroup.Add(1)

		go func(steamID steamid.SID64) {
			defer waitGroup.Done()

			var (
				friends []steamweb.Friend
				key     = makeKey(KeyFriends, steamID)
			)

			friendsBody, errCache := a.cache.get(key)
			if errCache == nil {
				if err := json.Unmarshal(friendsBody, &friends); err != nil {
					a.log.Error("Failed to unmarshal cached result", zap.Error(err))
				}

				return
			}

			newFriends, errFriends := steamweb.GetFriendList(ctx, steamID)
			if errFriends != nil {
				a.log.Warn("Failed to fetch friends", zap.Error(errFriends))

				return
			}

			body, errMarshal := json.Marshal(newFriends)
			if errMarshal != nil {
				a.log.Error("Failed to unmarshal friends", zap.Error(errMarshal))

				return
			}

			if errSet := a.cache.set(key, bytes.NewReader(body)); errSet != nil {
				a.log.Error("Failed to update cache", zap.Error(errSet), zap.String("site", "ugc"))
			}

			mutex.Lock()
			output[steamID] = newFriends
			mutex.Unlock()
		}(currentID)
	}

	waitGroup.Wait()

	return output
}

func (a *App) getSteamBans(ctx context.Context, steamIDs steamid.Collection) ([]steamweb.PlayerBanState, error) {
	var (
		banStates []steamweb.PlayerBanState
		missed    steamid.Collection
	)

	for _, steamID := range steamIDs {
		var banState steamweb.PlayerBanState

		summaryBody, errCache := a.cache.get(makeKey(KeyBans, steamID))
		if errCache == nil {
			if err := json.Unmarshal(summaryBody, &banState); err != nil {
				a.log.Error("Failed to unmarshal cached result", zap.Error(err))
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
			return nil, errors.Wrap(errBans, "Failed to fetch ban state")
		}

		for _, ban := range newBans {
			body, errMarshal := json.Marshal(ban)
			if errMarshal != nil {
				return nil, errors.Wrap(errMarshal, "Failed to marshal ban state")
			}

			if errSet := a.cache.set(makeKey(KeyBans, ban.SteamID), bytes.NewReader(body)); errSet != nil {
				a.log.Error("Failed to update cache", zap.Error(errSet))
			}
		}

		banStates = append(banStates, newBans...)
	}

	return banStates, nil
}

func (a *App) getSteamSummaries(ctx context.Context, steamIDs steamid.Collection) ([]steamweb.PlayerSummary, error) {
	var (
		summaries []steamweb.PlayerSummary
		missed    steamid.Collection
	)

	for _, steamID := range steamIDs {
		var summary steamweb.PlayerSummary

		summaryBody, errCache := a.cache.get(makeKey(KeySummary, steamID))
		if errCache == nil {
			if err := json.Unmarshal(summaryBody, &summary); err != nil {
				a.log.Error("Failed to unmarshal cached result", zap.Error(err))
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
			return nil, errors.Wrap(errSummaries, "Failed to fetch summaries")
		}

		for _, summary := range newSummaries {
			body, errMarshal := json.Marshal(summary)
			if errMarshal != nil {
				return nil, errors.Wrap(errMarshal, "Failed to marshal friends")
			}

			if errSet := a.cache.set(makeKey(KeySummary, summary.SteamID), bytes.NewReader(body)); errSet != nil {
				a.log.Error("Failed to update cache", zap.Error(errSet))
			}
		}

		summaries = append(summaries, newSummaries...)
	}

	return summaries, nil
}

func (a *App) profileUpdater(ctx context.Context) {
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
			var expiredIds steamid.Collection

			expiredProfiles, errProfiles := a.db.playerGetExpiredProfiles(ctx, maxQueuedCount)
			if errProfiles != nil {
				a.log.Error("Failed to fetch expired profiles", zap.Error(errProfiles))
			}

			additional := 0

			for len(expiredProfiles) < maxQueuedCount {
				for _, sid64 := range updateQueue {
					var pr PlayerRecord
					if errQueued := a.db.playerGetOrCreate(ctx, sid64, &pr); errQueued != nil {
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
				expiredIds = append(expiredIds, profile.SteamID)
			}

			summaries, errSum := steamweb.PlayerSummaries(ctx, expiredIds)
			if errSum != nil {
				a.log.Error("Failed to fetch summaries", zap.Error(errSum))

				continue
			}

			bans, errBans := steamweb.GetPlayerBans(ctx, expiredIds)
			if errBans != nil {
				a.log.Error("Failed to fetch bans", zap.Error(errSum))

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

				if errSave := a.db.playerRecordSave(ctx, &prof); errSave != nil {
					a.log.Error("Failed to update profile", zap.Int64("sid", prof.SteamID.Int64()), zap.Error(errSave))
				}
			}

		case <-ctx.Done():
			return
		}
	}
}
