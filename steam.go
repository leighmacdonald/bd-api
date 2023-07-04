package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/leighmacdonald/steamid/v3/steamid"
	"github.com/leighmacdonald/steamweb/v2"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

func (a *App) getSteamFriends(ctx context.Context, steamID steamid.SID64) ([]steamweb.Friend, error) {
	var friends []steamweb.Friend

	key := fmt.Sprintf("steam-friends-%d", steamID.Int64())

	friendsBody, errCache := a.cache.get(key)
	if errCache == nil {
		if err := json.Unmarshal(friendsBody, &friends); err != nil {
			a.log.Error("Failed to unmarshal cached result", zap.Error(err))
		}

		return friends, nil
	}

	newFriends, errFriends := steamweb.GetFriendList(ctx, steamID)
	if errFriends != nil {
		return nil, errors.Wrap(errFriends, "Failed to fetch friends")
	}

	body, errMarshal := json.Marshal(newFriends)
	if errMarshal != nil {
		return nil, errors.Wrap(errFriends, "Failed to unmarshal friends")
	}

	if errSet := a.cache.set(key, bytes.NewReader(body)); errSet != nil {
		a.log.Error("Failed to update cache", zap.Error(errSet), zap.String("site", "ugc"))
	}

	return newFriends, nil
}

func (a *App) getSteamSummary(ctx context.Context, steamID steamid.SID64) ([]steamweb.PlayerSummary, error) {
	var summaries []steamweb.PlayerSummary

	key := fmt.Sprintf("steam-summary-%d", steamID.Int64())

	summaryBody, errCache := a.cache.get(key)
	if errCache == nil {
		if err := json.Unmarshal(summaryBody, &summaries); err != nil {
			a.log.Error("Failed to unmarshal cached result", zap.Error(err))
		}

		return summaries, nil
	}

	newSummaries, errSummaries := steamweb.PlayerSummaries(ctx, steamid.Collection{steamID})
	if errSummaries != nil {
		return nil, errors.Wrap(errSummaries, "Failed to fetch summaries")
	}

	body, errMarshal := json.Marshal(newSummaries)
	if errMarshal != nil {
		return nil, errors.Wrap(errMarshal, "Failed to marshal friends")
	}

	if errSet := a.cache.set(key, bytes.NewReader(body)); errSet != nil {
		a.log.Error("Failed to update cache", zap.Error(errSet), zap.String("site", "ugc"))
	}

	return newSummaries, nil
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
					var pr playerRecord
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
