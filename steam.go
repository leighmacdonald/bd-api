package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/leighmacdonald/steamweb"
	"go.uber.org/zap"
)

func getSteamFriends(steamID steamid.SID64) ([]steamweb.Friend, error) {
	var friends []steamweb.Friend
	key := fmt.Sprintf("steam-friends-%d", steamID.Int64())
	friendsBody, errCache := cacheGet(key)
	if errCache == nil {
		if err := json.Unmarshal(friendsBody, &friends); err != nil {
			logger.Error("Failed to unmarshal cached result", zap.Error(err))
		}
		return friends, nil
	}
	newFriends, errFriends := steamweb.GetFriendList(steamID)
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

func getSteamSummary(steamID steamid.SID64) ([]steamweb.PlayerSummary, error) {
	var summaries []steamweb.PlayerSummary
	key := fmt.Sprintf("steam-summary-%d", steamID.Int64())
	summaryBody, errCache := cacheGet(key)
	if errCache == nil {
		if err := json.Unmarshal(summaryBody, &summaries); err != nil {
			logger.Error("Failed to unmarshal cached result", zap.Error(err))
		}
		return summaries, nil
	}
	newSummaries, errSummaries := steamweb.PlayerSummaries(steamid.Collection{steamID})
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
