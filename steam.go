package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/leighmacdonald/bd-api/domain"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/leighmacdonald/steamweb/v2"
)

var (
	errSteamBanFetch      = errors.New("failed to fetch steam ban state")
	errSteamBanDecode     = errors.New("failed to decode steam ban state")
	errSteamSummaryFetch  = errors.New("failed to fetch steam summary")
	errSteamSummaryDecode = errors.New("failed to decode steam summary")
	errAddr               = errors.New("failed to parse server addr")
	errPort               = errors.New("failed to parse server port")
	errFetchServers       = errors.New("could not fetch servers")
)

func parseHostPort(hostPortStr string) (net.IP, int, error) {
	hostParts := strings.Split(hostPortStr, ":")
	if len(hostParts) != 2 {
		return nil, 0, errAddr
	}

	addr := net.ParseIP(hostParts[0])
	if addr == nil {
		return nil, 0, errAddr
	}

	port, err := strconv.Atoi(hostParts[1])
	if err != nil {
		return nil, 0, errors.Join(err, errPort)
	}

	return addr, port, nil
}

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

func updateOwnedGames(ctx context.Context, database *pgStore, steamID steamid.SteamID) ([]domain.PlayerSteamGameOwned, error) {
	games, errGames := steamweb.GetOwnedGames(ctx, steamID)
	if errGames != nil {
		// TODO check for just private info.
		return nil, errors.Join(errGames, errSteamAPIResult)
	}

	var owned []domain.PlayerSteamGameOwned //nolint:prealloc

	for _, game := range games {
		steamGame := domain.SteamGame{
			AppID:      game.AppID,
			Name:       game.Name,
			ImgIconURL: game.ImgIconURL,
			ImgLogoURL: game.ImgLogoURL,
			TimeStamped: domain.TimeStamped{
				CreatedOn: time.Now(),
				UpdatedOn: time.Now(),
			},
		}

		if err := database.ensureSteamGame(ctx, steamGame); err != nil {
			return nil, err
		}

		ownedGame := domain.SteamGameOwned{
			SteamID:                  steamID,
			AppID:                    game.AppID,
			PlaytimeForeverMinutes:   game.PlaytimeForever,
			PlaytimeTwoWeeks:         game.Playtime2Weeks,
			HasCommunityVisibleStats: game.HasCommunityVisibleStats,
			TimeStamped: domain.TimeStamped{
				CreatedOn: time.Now(),
				UpdatedOn: time.Now(),
			},
		}

		if err := database.updateOwnedGame(ctx, ownedGame); err != nil {
			return nil, err
		}

		owned = append(owned, domain.PlayerSteamGameOwned{
			SteamID:                  steamID,
			AppID:                    game.AppID,
			Name:                     game.Name,
			ImgIconURL:               game.ImgIconURL,
			ImgLogoURL:               game.ImgLogoURL,
			PlaytimeForeverMinutes:   ownedGame.PlaytimeForeverMinutes,
			PlaytimeTwoWeeks:         ownedGame.PlaytimeTwoWeeks,
			HasCommunityVisibleStats: ownedGame.HasCommunityVisibleStats,
			TimeStamped:              ownedGame.TimeStamped,
		})
	}

	return owned, nil
}

func getOwnedGames(ctx context.Context, database *pgStore, steamIDs steamid.Collection, forceUpdate bool) (OwnedGameMap, error) {
	var (
		missed steamid.Collection
		owned  = OwnedGameMap{}
	)

	if !forceUpdate {
		existing, errOwned := database.getOwnedGames(ctx, steamIDs)
		if errOwned != nil && !errors.Is(errOwned, errDatabaseNoResults) {
			return nil, errOwned
		}

		for _, sid := range steamIDs {
			games, ok := existing[sid]
			if !ok || len(games) == 0 {
				missed = append(missed, sid)
			}
		}

		owned = existing
	} else {
		missed = steamIDs
	}

	for _, missedSid := range missed {
		time.Sleep(time.Millisecond * 100)
		updated, err := updateOwnedGames(ctx, database, missedSid)
		if err != nil {
			return nil, err
		}

		if updated == nil {
			updated = []domain.PlayerSteamGameOwned{}
		}

		owned[missedSid] = updated
	}

	return owned, nil
}
