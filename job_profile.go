package main

import (
	"context"
	"log/slog"

	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/riverqueue/river"
)

type JobPriority int

const (
	RealTime JobPriority = 1
	Normal   JobPriority = 2
	Slow     JobPriority = 3
)

type JobKinds string

const (
	Profile JobKinds = "profile"
)

type ProfileArgs struct {
	SteamID     steamid.SteamID `json:"steam_id"`
	JobPriority JobPriority     `json:"job_priority"`
}

func (ProfileArgs) Kind() string {
	return string(Profile)
}

type ProfileWorker struct {
	river.WorkerDefaults[ProfileArgs]
}

func (w *ProfileWorker) Work(ctx context.Context, job *river.Job[ProfileArgs]) error {
	slog.Info("Updating profile", slog.String("steam_id", job.Args.SteamID.String()))

	return nil
}
