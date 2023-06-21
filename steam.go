package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/leighmacdonald/steamweb/v2"
	"go.uber.org/zap"
	"time"
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
		logger.Error("Failed to fetch friends",
			zap.Error(errFriends), zap.Int64("steam_id", steamID.Int64()))
		return nil, errFriends
	}
	body, errMarshal := json.Marshal(newFriends)
	if errMarshal != nil {
		logger.Error("Failed to marshal friends", zap.Error(errMarshal))
		return newFriends, nil
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
		logger.Error("Failed to fetch summaries",
			zap.Error(errSummaries), zap.Int64("steam_id", steamID.Int64()))
		return nil, errSummaries
	}
	body, errMarshal := json.Marshal(newSummaries)
	if errMarshal != nil {
		logger.Error("Failed to marshal friends", zap.Error(errMarshal))
		return newSummaries, nil
	}
	if errSet := cacheSet(key, bytes.NewReader(body)); errSet != nil {
		logger.Error("Failed to update cache", zap.Error(errSet), zap.String("site", "ugc"))
	}
	return newSummaries, nil
}

func profileUpdater(ctx context.Context, db *pgStore, inChan <-chan steamid.SID64) {
	var updateQueue steamid.Collection
	updateTicker := time.NewTicker(time.Second * 10)
	for {
		select {
		case <-updateTicker.C:
			if len(updateQueue) == 0 {
				continue
			}
			var expiredIds steamid.Collection
			expiredIds = append(expiredIds, updateQueue...)
			profiles, errProfiles := db.playerGetExpiredProfiles(ctx, 100-len(expiredIds))
			if errProfiles != nil {
				logger.Error("Failed to fetch expired profiles", zap.Error(errProfiles))
			}
			for _, sid64 := range updateQueue {
				var pr playerRecord
				if errQueued := db.playerGetOrCreate(ctx, sid64, &pr); errQueued != nil {
					continue
				}
				profiles = append(profiles, pr)
			}
			if len(profiles) == 0 {
				continue
			}
			updateQueue = nil
			for _, profile := range profiles {
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
			for _, profile := range profiles {
				for _, sum := range summaries {
					if sum.SteamID == profile.SteamID {
						profile.CommunityVisibilityState = sum.CommunityVisibilityState
						profile.Vanity = sum.ProfileURL
						profile.AvatarHash = sum.AvatarHash
						profile.ProfileState = sum.ProfileState
						profile.PersonaName = sum.PersonaName
						profile.TimeCreated = sum.TimeCreated
						profile.LocCityID = sum.LocCityID
						profile.LocCountryCode = sum.LocCountryCode
						profile.LocStateCode = sum.LocStateCode
						break
					}
				}
				for _, ban := range bans {
					if ban.SteamID == profile.SteamID {
						profile.CommunityBanned = ban.CommunityBanned
						profile.VacBanned = ban.VACBanned
						profile.GameBans = ban.NumberOfGameBans
						profile.LastBannedOn = time.Now().Add(-(time.Second * time.Duration(ban.DaysSinceLastBan)))
						if ban.EconomyBan == steamweb.EconBanNone {
							profile.EconomyBanned = 0
						} else if ban.EconomyBan == steamweb.EconBanProbation {
							profile.EconomyBanned = 1
						} else if ban.EconomyBan == steamweb.EconBanBanned {
							profile.EconomyBanned = 2
						}
						profile.UpdatedOn = time.Now()
					}
				}
				if errSave := db.playerRecordSave(ctx, &profile); errSave != nil {

				}
			}
		case updateSid := <-inChan:
			updateQueue = append(updateQueue, updateSid)
		case <-ctx.Done():
			return
		}
	}
}
