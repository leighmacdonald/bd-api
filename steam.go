package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/leighmacdonald/steamweb/v2"
	"github.com/pkg/errors"

	"github.com/leighmacdonald/steamid/v2/steamid"
	"go.uber.org/zap"
)

func getSteamFriends(ctx context.Context, steamID steamid.SID64) ([]steamweb.Friend, error) {
	var friends []steamweb.Friend

	key := fmt.Sprintf("steam-friends-%d", steamID.Int64())

	friendsBody, errCache := cacheGet(key)
	if errCache == nil {
		if err := json.Unmarshal(friendsBody, &friends); err != nil {
			logger.Error("Failed to unmarshal cached result", zap.Error(err))
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

	if errSet := cacheSet(key, bytes.NewReader(body)); errSet != nil {
		logger.Error("Failed to update cache", zap.Error(errSet), zap.String("site", "ugc"))
	}

	return newFriends, nil
}

func getSteamSummary(ctx context.Context, steamID steamid.SID64) ([]steamweb.PlayerSummary, error) {
	var summaries []steamweb.PlayerSummary

	key := fmt.Sprintf("steam-summary-%d", steamID.Int64())

	summaryBody, errCache := cacheGet(key)
	if errCache == nil {
		if err := json.Unmarshal(summaryBody, &summaries); err != nil {
			logger.Error("Failed to unmarshal cached result", zap.Error(err))
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

	if errSet := cacheSet(key, bytes.NewReader(body)); errSet != nil {
		logger.Error("Failed to update cache", zap.Error(errSet), zap.String("site", "ugc"))
	}

	return newSummaries, nil
}

func profileUpdater(ctx context.Context, database *pgStore, inChan <-chan steamid.SID64) {
	const (
		maxQueuedCount = 100
		updateInterval = time.Second * 5
	)

	var updateQueue steamid.Collection

	updateTicker := time.NewTicker(updateInterval)
	triggerUpdate := make(chan any)

	for {
		select {
		case <-updateTicker.C:
			triggerUpdate <- true
		case updateSid := <-inChan:
			updateQueue = append(updateQueue, updateSid)
			if len(updateQueue) >= maxQueuedCount {
				triggerUpdate <- true
			}
		case <-triggerUpdate:
			var expiredIds steamid.Collection
			expiredIds = append(expiredIds, updateQueue...)

			expiredProfiles, errProfiles := database.playerGetExpiredProfiles(ctx, maxQueuedCount-len(expiredIds))
			if errProfiles != nil {
				logger.Error("Failed to fetch expired profiles", zap.Error(errProfiles))
			}

			for _, sid64 := range updateQueue {
				var pr playerRecord
				if errQueued := database.playerGetOrCreate(ctx, sid64, &pr); errQueued != nil {
					continue
				}

				expiredProfiles = append(expiredProfiles, pr)
			}

			if len(expiredProfiles) == 0 {
				continue
			}

			updateQueue = nil

			for _, profile := range expiredProfiles {
				expiredIds = append(expiredIds, profile.SteamID)
			}

			summaries, errSum := steamweb.PlayerSummaries(ctx, expiredIds)
			if errSum != nil {
				logger.Error("Failed to fetch summaries", zap.Error(errSum))

				continue
			}

			bans, errBans := steamweb.GetPlayerBans(ctx, expiredIds)
			if errBans != nil {
				logger.Error("Failed to fetch bans", zap.Error(errSum))

				continue
			}

			for _, profile := range expiredProfiles {
				for _, sum := range summaries {
					if sum.SteamID == profile.SteamID {
						profile.applySummary(sum)

						break
					}
				}

				for _, ban := range bans {
					if ban.SteamID == profile.SteamID {
						profile.applyBans(ban)

						break
					}
				}

				if errSave := database.playerRecordSave(ctx, &profile); errSave != nil {
					logger.Error("Failed to update profile", zap.Int64("sid", profile.SteamID.Int64()), zap.Error(errSave))
				}
			}

		case <-ctx.Done():
			return
		}
	}
}
