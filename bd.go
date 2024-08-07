package main

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/leighmacdonald/bd-api/domain"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

// fetchList downloads and parses the list defined by BDList and returns the parsed schema object.
func fetchList(ctx context.Context, client *http.Client, list domain.BDList) (*domain.TF2BDSchema, error) {
	lCtx, cancel := context.WithTimeout(ctx, time.Second*10)
	defer cancel()

	req, errReq := http.NewRequestWithContext(lCtx, http.MethodGet, list.URL, nil)
	if errReq != nil {
		return nil, errors.Join(errReq, errRequestCreate)
	}
	resp, errResp := client.Do(req)
	if errResp != nil {
		return nil, errors.Join(errResp, errRequestPerform)
	}

	defer func() {
		if errClose := resp.Body.Close(); errClose != nil {
			slog.Error("failed to close response body", ErrAttr(errClose))
		}
	}()

	var schema domain.TF2BDSchema
	if errDecode := json.NewDecoder(resp.Body).Decode(&schema); errDecode != nil {
		return nil, errResponseDecode
	}

	return &schema, nil
}

type listMapping struct {
	list   domain.BDList
	result *domain.TF2BDSchema
}

// updateLists will fetch all provided lists and update the local database.
//
// - If the entry is not known locally, it is created
// - If a known entry is no longer in the downloaded list, it is marked as deleted.
// - If an entry contains differing data, it will be updated.
func updateLists(ctx context.Context, lists []domain.BDList, database *pgStore) error {
	client := NewHTTPClient()
	waitGroup := sync.WaitGroup{}
	errs := make(chan error, len(lists))
	results := make(chan listMapping, len(lists))

	for _, list := range lists {
		if list.Deleted {
			continue
		}
		waitGroup.Add(1)
		go func(lCtx context.Context, localList domain.BDList) {
			defer waitGroup.Done()
			bdList, errFetch := fetchList(lCtx, client, localList)
			if errFetch != nil {
				errs <- errFetch

				return
			}
			results <- listMapping{list: localList, result: bdList}
		}(ctx, list)
	}
	go func() {
		waitGroup.Wait()
		close(results)
		close(errs)
	}()

	for err := range errs {
		slog.Error(err.Error())
	}

	for res := range results {
		if errUpdate := updateListEntries(ctx, database, res); errUpdate != nil {
			return errUpdate
		}
	}

	return nil
}

var (
	errUpdateEntryFailed = errors.New("failed to commit updated bd entry")
	errDeleteEntryFailed = errors.New("failed to commit deleted bd entry")
	errPlayerGetOrCreate = errors.New("failed to get/create player record")
)

func updateListEntries(ctx context.Context, database *pgStore, mapping listMapping) error {
	existingList, errExisting := database.botDetectorListEntries(ctx, mapping.list.BDListID)
	if errExisting != nil {
		return errExisting
	}

	newEntries, updatedEntries := findNewAndUpdated(existingList, mapping)
	deletedEntries := findDeleted(existingList, mapping)

	for _, entry := range newEntries {
		pr := newPlayerRecord(entry.SteamID)
		if err := database.playerGetOrCreate(ctx, entry.SteamID, &pr); err != nil {
			return errors.Join(err, errPlayerGetOrCreate)
		}
		if _, err := database.botDetectorListEntryCreate(ctx, entry); err != nil {
			if errors.Is(err, errDatabaseUnique) {
				continue
			}
			slog.Error("Failed to create list entry", ErrAttr(err))

			continue
		}
	}
	if len(newEntries) > 0 {
		slog.Info("Added new list entries", slog.Int("count", len(newEntries)),
			slog.String("list", mapping.list.BDListName), slog.Int("bd_list_id", mapping.list.BDListID))
	}

	for _, updated := range updatedEntries {
		if err := database.botDetectorListEntryUpdate(ctx, updated); err != nil {
			return errors.Join(err, errUpdateEntryFailed)
		}
	}
	if len(updatedEntries) > 0 {
		slog.Info("Updated list entries", slog.Int("count", len(updatedEntries)),
			slog.String("list", mapping.list.BDListName), slog.Int("bd_list_id", mapping.list.BDListID))
	}

	for _, entry := range deletedEntries {
		if err := database.bdListEntryDelete(ctx, entry.BDListEntryID); err != nil {
			return errors.Join(err, errDeleteEntryFailed)
		}
	}
	if len(deletedEntries) > 0 {
		slog.Info("Deleted list entries", slog.Int("count", len(deletedEntries)),
			slog.String("list", mapping.list.BDListName), slog.Int("bd_list_id", mapping.list.BDListID))
	}

	return nil
}

// normalizeAttrs will filter all attributes, ensuring they are lowercase, unique, not empty, have any surrounding
// whitespace removed and sorted.
func normalizeAttrs(inputAttrs []string) []string {
	var attrs []string
	for idx := range inputAttrs {
		value := strings.TrimSpace(strings.ToLower(inputAttrs[idx]))
		if value == "" {
			continue
		}

		if !slices.Contains(attrs, value) {
			attrs = append(attrs, value)
		}
	}

	slices.Sort(attrs)

	return attrs
}

// Search results for existing entries with new attrs.
func findNewAndUpdated(existingList []domain.BDListEntry, mapping listMapping) ([]domain.BDListEntry, []domain.BDListEntry) {
	var (
		newEntries []domain.BDListEntry
		updated    []domain.BDListEntry
	)

	for _, player := range mapping.result.Players {
		sid := steamid.New(player.Steamid)
		if !sid.Valid() {
			slog.Warn("got invalid steam id", slog.Any("sid", sid), slog.String("list", mapping.list.BDListName))

			continue
		}
		found := false
		for _, existing := range existingList {
			if existing.SteamID == sid {
				found = true
				lastSeen := time.Unix(int64(player.LastSeen.Time), 0)
				attrs := normalizeAttrs(player.Attributes)
				if player.Proof == nil {
					player.Proof = []string{}
				}
				if existing.LastName != player.LastSeen.PlayerName || !slices.Equal(existing.Attributes, attrs) || !slices.Equal(existing.Proof, player.Proof) {
					updatedEntry := existing
					updatedEntry.LastSeen = lastSeen
					updatedEntry.Attributes = attrs
					updatedEntry.Proof = player.Proof
					updatedEntry.LastName = player.LastSeen.PlayerName
					updatedEntry.UpdatedOn = time.Now()
					updated = append(updated, updatedEntry)
				}

				break
			}
		}
		if !found {
			now := time.Now()
			newEntry := domain.BDListEntry{
				BDListEntryID: 0,
				BDListID:      mapping.list.BDListID,
				SteamID:       sid,
				Attributes:    normalizeAttrs(player.Attributes),
				Proof:         player.Proof,
				LastSeen:      time.Unix(int64(player.LastSeen.Time), 0),
				LastName:      player.LastSeen.PlayerName,
				Deleted:       false,
				CreatedOn:     now,
				UpdatedOn:     now,
			}
			newEntries = append(newEntries, newEntry)
		}
	}

	return newEntries, updated
}

// Search results for deleted entries.
func findDeleted(existingList []domain.BDListEntry, mapping listMapping) []domain.BDListEntry {
	var deleted []domain.BDListEntry
	for _, player := range existingList {
		found := false
		for _, entry := range mapping.result.Players {
			sid := steamid.New(entry.Steamid)

			if sid == player.SteamID {
				found = true

				break
			}
		}
		if !found {
			deleted = append(deleted, player)
		}
	}

	return deleted
}

func doListUpdate(ctx context.Context, database *pgStore) error {
	lists, errLists := database.botDetectorLists(ctx)
	if errLists != nil {
		return errLists
	}

	return updateLists(ctx, lists, database)
}
