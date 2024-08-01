package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/leighmacdonald/bd-api/domain"
	"github.com/leighmacdonald/etf2l"
)

var errETF2LFetchBans = errors.New("failed to fetch etf2l bans")

func updateETF2LBans(ctx context.Context, database *pgStore, client *etf2l.Client, httpClient *http.Client) error {
	bans, errBans := client.Bans(ctx, httpClient, etf2l.BanOpts{
		Recursive: etf2l.BaseOpts{Recursive: true},
	})

	if errBans != nil {
		return errors.Join(errBans, errETF2LFetchBans)
	}

	var eBans []domain.ETF2LBan //nolint:prealloc

	for _, ban := range bans {
		if !ban.Steamid64.Valid() {
			continue
		}

		eBans = append(eBans, domain.ETF2LBan{
			SteamID:   ban.Steamid64,
			Alias:     ban.Name,
			ExpiresAt: time.Unix(int64(ban.End), 0).Truncate(time.Second),
			CreatedAt: time.Unix(int64(ban.Start), 0).Truncate(time.Second),
			Reason:    ban.Reason,
		})
	}

	if err := database.etf2lBansUpdate(ctx, eBans); err != nil {
		return dbErr(err, "failed to update etf2l bans")
	}

	slog.Debug("Got ETF2L bans", slog.Int("count", len(bans)))

	return nil
}
