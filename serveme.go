package main

import (
	"context"
	"encoding/csv"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/leighmacdonald/bd-api/domain"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

var errReadCSVRows = errors.New("failed to read csv rows")

func updateServeMe(ctx context.Context, database *pgStore) error {
	entries, errDownload := downloadServeMeList(ctx)
	if errDownload != nil {
		return errDownload
	}

	if len(entries) == 0 {
		return nil
	}

	// Ensure FK's are satisfied
	for _, entry := range entries {
		if _, err := database.playerGetOrCreate(ctx, entry.SteamID); err != nil {
			return err
		}
	}

	if err := database.servemeUpdate(ctx, entries); err != nil {
		return err
	}

	slog.Info("Inserted serveme records", slog.Int("count", len(entries)))

	return nil
}

func downloadServeMeList(ctx context.Context) ([]domain.ServeMeRecord, error) {
	client := NewHTTPClient()
	req, errReq := http.NewRequestWithContext(ctx, http.MethodGet, "https://raw.githubusercontent.com/Arie/serveme/master/doc/banned_steam_ids.csv", nil)
	if errReq != nil {
		return nil, errors.Join(errReq, errRequestCreate)
	}

	resp, errResp := client.Do(req)
	if errResp != nil {
		return nil, errors.Join(errResp, errRequestPerform)
	}

	defer func() {
		if err := resp.Body.Close(); err != nil {
			slog.Error("failed to close serveme response body", ErrAttr(err))
		}
	}()

	reader := csv.NewReader(resp.Body)
	reader.FieldsPerRecord = 3
	reader.TrimLeadingSpace = true

	now := time.Now()

	var records []domain.ServeMeRecord
	for {
		row, errRows := reader.Read()
		if errRows == io.EOF {
			break
		}

		if errRows != nil {
			return nil, errors.Join(errRows, errReadCSVRows)
		}

		if len(row) != 3 {
			continue
		}

		sid := steamid.New(row[0])
		if !sid.Valid() {
			slog.Warn("Got invalid serveme steamid", slog.String("steam_id", row[0]))

			continue
		}

		records = append(records, domain.ServeMeRecord{
			SteamID: sid,
			Name:    row[1],
			Reason:  row[2],
			Deleted: false,
			TimeStamped: domain.TimeStamped{
				UpdatedOn: now,
				CreatedOn: now,
			},
		})
	}

	return records, nil
}
