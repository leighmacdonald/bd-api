package main

import (
	"context"
	"log/slog"

	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/riverqueue/river"
)

type SteamGamesArgs struct {
	SteamID steamid.SteamID `json:"steam_id"`
}

func (SteamGamesArgs) Kind() string {
	return string(KindSteamGames)
}

func (SteamGamesArgs) InsertOpts() river.InsertOpts {
	return steamInsertOpts()
}

type SteamGamesWorker struct {
	river.WorkerDefaults[SteamGamesArgs]
	// database   *pgStore
	// httpClient *http.Client
}

func (w *SteamGamesWorker) Work(_ context.Context, job *river.Job[SteamGamesArgs]) error { //nolint:unparam
	// https://wiki.teamfortress.com/wiki/WebAPI/GetOwnedGames

	slog.Debug("Updating games", slog.String("steam_id", job.Args.SteamID.String()))

	return nil
}
