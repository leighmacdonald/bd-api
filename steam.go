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
	errAddr         = errors.New("failed to parse server addr")
	errPort         = errors.New("failed to parse server port")
	errFetchServers = errors.New("could not fetch servers")
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

func playerBanStateFromSteam(s steamweb.PlayerBanState) domain.PlayerBanState {
	return domain.PlayerBanState{
		SteamID:          s.SteamID,
		CommunityBanned:  s.CommunityBanned,
		VACBanned:        s.VACBanned,
		NumberOfVACBans:  s.NumberOfVACBans,
		DaysSinceLastBan: s.DaysSinceLastBan,
		NumberOfGameBans: s.NumberOfGameBans,
		EconomyBan:       s.EconomyBan,
		TimeStamped:      newTimeStamped(),
	}
}

func getSteamBans(ctx context.Context, database *pgStore, steamIDs steamid.Collection) ([]domain.PlayerBanState, error) {
	var (
		invalid   steamid.Collection
		validBans []domain.PlayerBanState
	)

	bans, errBans := database.getPlayerBanStates(ctx, steamIDs)
	if errBans != nil {
		return nil, errBans
	}

	for _, sid := range steamIDs {
		foundAndValid := false

		for _, ban := range bans {
			if ban.SteamID == sid {
				// Mark old entries to be refreshed from API
				if time.Since(ban.UpdatedOn) < time.Hour*12 {
					foundAndValid = true

					validBans = append(validBans, ban)
				}

				break
			}
		}

		if !foundAndValid {
			invalid = append(invalid, sid)
		}
	}

	if len(invalid) > 0 {
		updates, err := steamweb.GetPlayerBans(ctx, invalid)
		if err != nil {
			return nil, err
		}

		for _, banUpdate := range updates {
			banState := playerBanStateFromSteam(banUpdate)
			if errSave := database.playerBanStateSave(ctx, banState); errSave != nil {
				return nil, errSave
			}

			validBans = append(validBans, banState)
		}

	}

	return validBans, nil
}

func newTimeStamped() domain.TimeStamped {
	now := time.Now()

	return domain.TimeStamped{CreatedOn: now, UpdatedOn: now}
}

func playerFromSteamSummary(summary steamweb.PlayerSummary) domain.Player {
	return domain.Player{
		SteamID:                  summary.SteamID,
		CommunityVisibilityState: summary.CommunityVisibilityState,
		ProfileState:             summary.ProfileState,
		PersonaName:              summary.PersonaName,
		Vanity:                   "",
		AvatarHash:               summary.AvatarHash,
		PersonaState:             summary.PersonaState,
		RealName:                 summary.RealName,
		TimeCreated:              time.Unix(int64(summary.TimeCreated), 0),
		LocCountryCode:           summary.LocCountryCode,
		LocStateCode:             summary.LocStateCode,
		LocCityID:                summary.LocCityID,
		LogsTFCount:              0,
		TimeStamped:              newTimeStamped(),
	}
}

// getSteamSummaries handles querying player profiles from the database. If the player
// does not already exist, or if the player otherwise is out of date, the steam api
// will be queried to pull in the latest data.
func getSteamSummaries(ctx context.Context, database *pgStore, steamIDs steamid.Collection) ([]domain.Player, error) {
	var (
		invalid        steamid.Collection
		validSummaries []domain.Player
	)

	summaries, errSummaries := database.getPlayerSummaries(ctx, steamIDs)
	if errSummaries != nil && !errors.Is(errSummaries, errDatabaseNoResults) {
		return nil, errSummaries
	}

	for _, sid := range steamIDs {
		foundAndValid := false

		for _, sum := range summaries {
			if sum.SteamID == sid {
				// Mark old entries to be refreshed from API
				if time.Since(sum.UpdatedOn) < time.Hour*12 {
					foundAndValid = true

					validSummaries = append(validSummaries, sum)
				}

				break
			}
		}

		if !foundAndValid {
			invalid = append(invalid, sid)

			continue
		}
	}

	if len(invalid) > 0 {
		updates, err := steamweb.PlayerSummaries(ctx, invalid)
		if err != nil {
			return nil, err
		}

		for _, summaryUpdate := range updates {
			player := playerFromSteamSummary(summaryUpdate)
			if errSave := database.playerSave(ctx, player); errSave != nil {
				return nil, errSave
			}

			validSummaries = append(validSummaries, player)
		}

	}

	return validSummaries, nil
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
