package main

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/leighmacdonald/bd-api/domain"
	"github.com/leighmacdonald/etf2l"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/riverqueue/river/rivermigrate"
	"github.com/riverqueue/river/rivershared/util/slogutil"
	"github.com/riverqueue/river/rivertype"
)

var (
	errQueueSetup  = errors.New("could not setup job queue")
	errQueueInsert = errors.New("failed to insert new jobs")
)

type JobPriority int

const (
	RealTime JobPriority = 1
	High     JobPriority = 2
	Normal   JobPriority = 3
	Slow     JobPriority = 4
)

type JobsKind string

const (
	KindRGLSeason    JobsKind = "rgl_season"
	KindRGLTeam      JobsKind = "rgl_team"
	KindRGLMatch     JobsKind = "rgl_match"
	KindRGLBan       JobsKind = "rgl_ban"
	KindETF2LBan     JobsKind = "etf2l_ban"
	KindSteamSummary JobsKind = "steam_summary"
	KindSteamBan     JobsKind = "steam_ban"
	KindSteamGames   JobsKind = "steam_games"
	KindSteamServers JobsKind = "steam_servers"
	KindServemeBan   JobsKind = "serveme_ban"
	KindSourcebans   JobsKind = "sourcebans"
	KindLogsTF       JobsKind = "logstf"
	KindBDLists      JobsKind = "bd_lists"
)

type JobQueue string

const (
	QueueDefault    JobQueue = "default"
	QueuePriority   JobQueue = "queue_priority"
	QueueRGL        JobQueue = "queue_rgl"
	QueueETF2L      JobQueue = "queue_etf2l"
	QueueSteam      JobQueue = "queue_steam"
	QueueLogsTF     JobQueue = "queue_logstf"
	QueueSourcebans JobQueue = "queue_sourcebans"
)

func rglInsertOpts() river.InsertOpts {
	return river.InsertOpts{
		Queue:    string(QueueRGL),
		Priority: int(Slow),
		UniqueOpts: river.UniqueOpts{
			ByArgs:   true,
			ByPeriod: 24 * time.Hour,
		},
	}
}

func steamInsertOpts() river.InsertOpts {
	return river.InsertOpts{
		Queue:    string(QueueSteam),
		Priority: int(Slow),
		// UniqueOpts: river.UniqueOpts{
		//	ByArgs:   true,
		//	ByPeriod: 24 * time.Hour,
		// },
	}
}

func etf2lInsertOpts() river.InsertOpts {
	return river.InsertOpts{
		Queue:    string(QueueETF2L),
		Priority: int(Slow),
	}
}

func setupQueue(ctx context.Context, dbPool *pgxpool.Pool) error {
	migrator := rivermigrate.New[pgx.Tx](riverpgxv5.New(dbPool), nil)

	res, err := migrator.Migrate(ctx, rivermigrate.DirectionUp, nil)
	if err != nil {
		return errors.Join(err, errDatabaseMigrate)
	}

	for _, ver := range res.Versions {
		slog.Info("Migrated river version", slog.Int("version", ver.Version))
	}

	return nil
}

func createJobWorkers(database *pgStore, config appConfig) *river.Workers {
	workers := river.NewWorkers()
	steamLimiter := NewSteamLimiter()

	// Steam
	river.AddWorker[SteamSummaryArgs](workers, &SteamSummaryWorker{
		database: database,
		limiter:  steamLimiter,
	})
	river.AddWorker[SteamBanArgs](workers, &SteamBanWorker{
		database: database,
		limiter:  steamLimiter,
	})
	river.AddWorker[SteamGamesArgs](workers, &SteamGamesWorker{
		database: database,
		limiter:  steamLimiter,
	})
	river.AddWorker[SteamServersArgs](workers, &SteamServersWorker{
		database:    database,
		limiter:     steamLimiter,
		serverCache: map[steamid.SteamID]domain.SteamServer{},
		mapCache:    map[string]domain.Map{},
	})

	// Serveme.tf
	river.AddWorker[ServemeArgs](workers, &ServemeWorker{
		database: database,
	})

	// Bot detector lists
	river.AddWorker[BDListArgs](workers, &BDListWorker{
		database: database,
	})

	// RGL
	if config.RGLScraperEnabled {
		rglLimiter := NewRGLLimiter()
		rglClient := NewHTTPClient()

		river.AddWorker[RGLBanArgs](workers, &RGLBansWorker{
			database:   database,
			limiter:    rglLimiter,
			httpClient: rglClient,
		})
		river.AddWorker[RGLSeasonArgs](workers, &RGLSeasonWorker{
			database:   database,
			limiter:    rglLimiter,
			httpClient: rglClient,
		})
		river.AddWorker[RGLTeamArgs](workers, &RGLTeamWorker{
			database:   database,
			limiter:    rglLimiter,
			httpClient: rglClient,
		})
		river.AddWorker[RGLMatchArgs](workers, &RGLMatchWorker{
			database:   database,
			limiter:    rglLimiter,
			httpClient: rglClient,
		})
	}

	// ETF2L
	if config.ETF2LScraperEnabled {
		river.AddWorker[ETF2LBanArgs](workers, &ETF2LBanWorker{
			database:   database,
			client:     etf2l.New(),
			httpClient: NewHTTPClient(),
		})
	}

	// Sourcebans
	if config.SourcebansScraperEnabled {
		river.AddWorker[SourcebansArgs](workers, &SourcebansWorker{
			database: database,
			config:   config,
		})
	}

	// Logs.tf
	if config.LogstfScraperEnabled {
		river.AddWorker[LogsTFArgs](workers, &LogsTFWorker{
			database: database,
			config:   config,
		})
	}

	return workers
}

func createPeriodicJobs(config appConfig) []*river.PeriodicJob {
	jobs := []*river.PeriodicJob{
		river.NewPeriodicJob(
			river.PeriodicInterval(12*time.Hour),
			func() (river.JobArgs, *river.InsertOpts) {
				return ServemeArgs{}, nil
			},
			&river.PeriodicJobOpts{RunOnStart: true}),
		river.NewPeriodicJob(
			river.PeriodicInterval(2*time.Minute),
			func() (river.JobArgs, *river.InsertOpts) {
				return SteamSummaryArgs{}, nil
			},
			&river.PeriodicJobOpts{RunOnStart: true}),
		river.NewPeriodicJob(
			river.PeriodicInterval(5*time.Minute),
			func() (river.JobArgs, *river.InsertOpts) {
				return SteamServersArgs{}, nil
			},
			&river.PeriodicJobOpts{RunOnStart: true}),
		river.NewPeriodicJob(
			river.PeriodicInterval(1*time.Minute),
			func() (river.JobArgs, *river.InsertOpts) {
				return BDListArgs{}, nil
			},
			&river.PeriodicJobOpts{RunOnStart: true}),
	}

	if config.RGLScraperEnabled {
		jobs = append(jobs,
			river.NewPeriodicJob(
				river.PeriodicInterval(12*time.Hour),
				func() (river.JobArgs, *river.InsertOpts) {
					return RGLBanArgs{}, nil
				},
				&river.PeriodicJobOpts{RunOnStart: true}),
			river.NewPeriodicJob(
				river.PeriodicInterval(24*time.Hour),
				func() (river.JobArgs, *river.InsertOpts) {
					return RGLSeasonArgs{StartSeasonID: 1}, nil
				},
				&river.PeriodicJobOpts{RunOnStart: true},
			),
		)
	}

	if config.ETF2LScraperEnabled {
		jobs = append(jobs,
			river.NewPeriodicJob(
				river.PeriodicInterval(12*time.Hour),
				func() (river.JobArgs, *river.InsertOpts) {
					return ETF2LBanArgs{}, nil
				},
				&river.PeriodicJobOpts{RunOnStart: true}))
	}

	if config.SourcebansScraperEnabled {
		jobs = append(jobs,
			river.NewPeriodicJob(
				river.PeriodicInterval(24*time.Hour),
				func() (river.JobArgs, *river.InsertOpts) {
					return SourcebansArgs{}, nil
				},
				&river.PeriodicJobOpts{RunOnStart: false}))
	}

	if config.LogstfScraperEnabled {
		jobs = append(jobs,
			river.NewPeriodicJob(
				river.PeriodicInterval(time.Hour),
				func() (river.JobArgs, *river.InsertOpts) {
					return LogsTFArgs{}, nil
				},
				&river.PeriodicJobOpts{RunOnStart: true}))
	}

	return jobs
}

func createJobClient(dbPool *pgxpool.Pool, workers *river.Workers, periodic []*river.PeriodicJob) (*river.Client[pgx.Tx], error) {
	newRiverClient, err := river.NewClient[pgx.Tx](riverpgxv5.New(dbPool), &river.Config{
		Logger:     slog.New(&slogutil.SlogMessageOnlyHandler{Level: slog.LevelWarn}),
		JobTimeout: time.Minute * 5,
		Queues: map[string]river.QueueConfig{
			string(QueueDefault):    {MaxWorkers: 2},
			string(QueuePriority):   {MaxWorkers: 1},
			string(QueueRGL):        {MaxWorkers: 1},
			string(QueueSteam):      {MaxWorkers: 1},
			string(QueueETF2L):      {MaxWorkers: 1},
			string(QueueLogsTF):     {MaxWorkers: 1},
			string(QueueSourcebans): {MaxWorkers: 1},
		},
		Workers:      workers,
		PeriodicJobs: periodic,
		ErrorHandler: &JobErrorHandler{},
		MaxAttempts:  3,
	})
	if err != nil {
		return nil, errors.Join(err, errQueueSetup)
	}

	return newRiverClient, nil
}

func initJobClient(ctx context.Context, database *pgStore, config appConfig) (*river.Client[pgx.Tx], error) {
	workers := createJobWorkers(database, config)
	periodic := createPeriodicJobs(config)

	newRiverClient, err := createJobClient(database.pool, workers, periodic)
	if err != nil {
		return nil, err
	}

	if errClientStart := newRiverClient.Start(ctx); errClientStart != nil {
		return nil, errors.Join(errClientStart, errQueueSetup)
	}

	return newRiverClient, nil
}

type JobErrorHandler struct{}

func (*JobErrorHandler) HandleError(_ context.Context, job *rivertype.JobRow, err error) *river.ErrorHandlerResult {
	slog.Error("Job returned error", ErrAttr(err),
		slog.String("queue", job.Queue), slog.String("kind", job.Kind),
		slog.String("args", string(job.EncodedArgs)))

	return nil
}

func (*JobErrorHandler) HandlePanic(_ context.Context, job *rivertype.JobRow, panicVal any, trace string) *river.ErrorHandlerResult {
	slog.Error("Job panic",
		slog.String("trace", trace), slog.Any("value", panicVal),
		slog.String("queue", job.Queue), slog.String("kind", job.Kind),
		slog.String("args", string(job.EncodedArgs)))

	return nil
}
