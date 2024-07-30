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

type ETF2LScraper struct {
	database   *pgStore
	client     *etf2l.Client
	httpClient *http.Client
}

func NewETF2LScraper(database *pgStore) ETF2LScraper {
	return ETF2LScraper{
		database:   database,
		client:     etf2l.New(),
		httpClient: NewHTTPClient(),
	}
}

func (e *ETF2LScraper) start(ctx context.Context) {
	e.scrape(ctx)

	ticker := time.NewTicker(time.Hour * 6)

	for {
		select {
		case <-ticker.C:
			e.scrape(ctx)
		case <-ctx.Done():
			return
		}
	}
}

func (e *ETF2LScraper) scrape(ctx context.Context) {
	if err := e.updateBans(ctx); err != nil {
		slog.Error("Failed to update ETF2L bans", ErrAttr(err))
	}
}

func (e *ETF2LScraper) updateBans(ctx context.Context) error {
	bans, errBans := e.client.Bans(ctx, e.httpClient, etf2l.BanOpts{
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

	if err := e.database.etf2lBansUpdate(ctx, eBans); err != nil {
		return dbErr(err, "failed to update etf2l bans")
	}

	slog.Info("Got ETF2L bans", slog.Int("count", len(bans)))

	return nil
}
