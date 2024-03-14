package main

import (
	"context"
	"fmt"
	"github.com/spf13/cobra"
	"log/slog"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var (
	BuildVersion = "dev" //nolint:gochecknoglobals
	BuildCommit  = "dev" //nolint:gochecknoglobals
	BuildDate    = ""    //nolint:gochecknoglobals
)

func bdListCmd() *cobra.Command {
	var (
		name    string
		listUrl string
		game    string
		weight  = 5
	)
	bdCmd := &cobra.Command{ //nolint:exhaustruct
		Use:   "bd",
		Short: "Bot detector list commands",
	}

	bdCmd.PersistentFlags().StringVar(&name, "name", "", "Unique name of the list")
	bdCmd.PersistentFlags().StringVar(&listUrl, "url", "", "Unique url for the list to download from")
	bdCmd.PersistentFlags().StringVar(&game, "game", "tf2", "Shorthand for the game that the list is from")
	bdCmd.PersistentFlags().IntVar(&weight, "weight", 5, "Weight/confidence of the lists. 0-10")

	bdCmd.AddCommand(&cobra.Command{ //nolint:exhaustruct
		Use:     "list",
		Aliases: []string{"l"},
		Run: func(cmd *cobra.Command, _ []string) {
			_, _, database, errSetup := createAppDeps(cmd.Context())
			if errSetup != nil {
				slog.Error("failed to setup app dependencies", ErrAttr(errSetup))

				return
			}
			lists, errLists := database.bdLists(cmd.Context())
			if errLists != nil {
				slog.Error("Failed to load lists", ErrAttr(errLists))

				return
			}

			for _, list := range lists {
				_, err := fmt.Fprintf(os.Stdout, "id: %d name: %s url: %s weight: %d\n",
					list.BDListID, list.BDListName, list.URL, list.TrustWeight)
				if err != nil {
					slog.Error("Failed to write output", ErrAttr(err))
				}
			}
		},
	})

	bdCmd.AddCommand(&cobra.Command{ //nolint:exhaustruct
		Use:     "add",
		Aliases: []string{"a"},
		Run: func(cmd *cobra.Command, _ []string) {
			if name == "" {
				slog.Error("Name cannot be empty")

				return
			}

			if listUrl == "" {
				slog.Error("URL cannot be empty")

				return
			}

			_, err := url.Parse(listUrl)
			if err != nil {
				slog.Error("Invalid URL, cannot parse", ErrAttr(err))

				return
			}

			_, _, database, errSetup := createAppDeps(cmd.Context())
			if errSetup != nil {
				slog.Error("failed to setup app dependencies", ErrAttr(errSetup))

				return
			}

			now := time.Now()

			list, errCreate := database.bdListCreate(cmd.Context(), BDList{ //nolint:exhaustruct
				BDListName:  name,
				URL:         listUrl,
				Game:        game,
				TrustWeight: weight,
				Deleted:     false,
				CreatedOn:   now,
				UpdatedOn:   now,
			})
			if errCreate != nil {
				slog.Error("Failed to create new bd list", ErrAttr(errCreate))

				return
			}

			slog.Info("Create list successfully", slog.String("name", name), slog.Int("bd_list_id", list.BDListID))
		},
	})

	bdCmd.AddCommand(&cobra.Command{ //nolint:exhaustruct
		Use:     "del",
		Aliases: []string{"d"},
		Run: func(cmd *cobra.Command, _ []string) {
			_, _, database, errSetup := createAppDeps(cmd.Context())
			if errSetup != nil {
				slog.Error("failed to setup app dependencies", ErrAttr(errSetup))

				return
			}

			list, errGet := database.bdListByName(cmd.Context(), name)
			if errGet != nil {
				slog.Error("Failed to find list by name", slog.String("name", name))

				return
			}

			if errDelete := database.bdListDelete(cmd.Context(), list.BDListID); errDelete != nil {
				slog.Error("failed to delete list", ErrAttr(errDelete))

				return
			}

			slog.Info("Deleted list successfully", slog.Int("id", list.BDListID))
		},
	})

	return bdCmd
}

func runCmd() *cobra.Command {
	return &cobra.Command{ //nolint:exhaustruct
		Use: "run",
		Run: func(cmd *cobra.Command, _ []string) {
			os.Exit(run(cmd.Context()))
		},
	}
}

func execute() {
	root := &cobra.Command{ //nolint:exhaustruct
		Use:     "bdapi",
		Version: fmt.Sprintf("%s %s %s", BuildVersion, BuildCommit, BuildDate),
	}

	root.AddCommand(runCmd())
	root.AddCommand(bdListCmd())

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	if err := root.ExecuteContext(ctx); err != nil {
		stop()
		os.Exit(1)
	}

	stop()
	os.Exit(0)
}
