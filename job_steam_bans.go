package main

import (
	"context"
	"errors"
	"net/http"

	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/leighmacdonald/steamweb/v2"
	"github.com/riverqueue/river"
)

type SteamBanArgs struct {
	SteamIDs steamid.Collection
}

func (SteamBanArgs) Kind() string {
	return string(KindSteamBan)
}

func (SteamBanArgs) InsertOpts() river.InsertOpts {
	return steamInsertOpts()
}

type SteamBanWorker struct {
	river.WorkerDefaults[SteamBanArgs]
	database   *pgStore
	httpClient *http.Client
}

func (w *SteamBanWorker) Work(ctx context.Context, job *river.Job[SteamBanArgs]) error {
	bans, errBans := steamweb.GetPlayerBans(ctx, job.Args.SteamIDs)
	if errBans != nil {
		return errors.Join(errBans, errSteamAPIResult)
	}

	for _, ban := range bans {
		var record PlayerRecord
		if errQueued := w.database.playerGetOrCreate(ctx, ban.SteamID, &record); errQueued != nil {
			continue
		}

		record.applyBans(ban)

		if errSave := w.database.playerRecordSave(ctx, &record); errSave != nil {
			return errSave
		}
	}

	return nil
}
