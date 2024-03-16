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

	"github.com/leighmacdonald/steamid/v4/steamid"
)

type FileInfo struct {
	Authors     []string `json:"authors"`
	Description string   `json:"description"`
	Title       string   `json:"title"`
	UpdateURL   string   `json:"update_url"`
}

type LastSeen struct {
	PlayerName string `json:"player_name,omitempty"`
	Time       int    `json:"time,omitempty"`
}

type TF2BDPlayer struct {
	Attributes []string `json:"attributes"`
	LastSeen   LastSeen `json:"last_seen,omitempty"`
	Steamid    any      `json:"steamid"`
	Proof      []string `json:"proof"`
}

type TF2BDSchema struct {
	Schema   string        `json:"$schema"` //nolint:tagliatelle
	FileInfo FileInfo      `json:"file_info"`
	Players  []TF2BDPlayer `json:"players"`
}

type BDList struct {
	BDListID    int
	BDListName  string
	URL         string
	Game        string
	TrustWeight int
	Deleted     bool
	CreatedOn   time.Time
	UpdatedOn   time.Time
}

type BDListEntry struct {
	BDListEntryID int64
	BDListID      int
	SteamID       steamid.SteamID
	Attributes    []string
	Proof         []string
	LastSeen      time.Time
	LastName      string
	Deleted       bool
	CreatedOn     time.Time
	UpdatedOn     time.Time
}

var (
	errRequestCreate  = errors.New("failed to create request")
	errRequestPerform = errors.New("failed to perform request")
	errRequestDecode  = errors.New("failed to decode request")
)

// fetchList downloads and parses the list defined by BDList and returns the parsed schema object.
func fetchList(ctx context.Context, client *http.Client, list BDList) (*TF2BDSchema, error) {
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

	var schema TF2BDSchema
	if errDecode := json.NewDecoder(resp.Body).Decode(&schema); errDecode != nil {
		return nil, errRequestDecode
	}

	return &schema, nil
}

type listMapping struct {
	list   BDList
	result *TF2BDSchema
}

// updateLists will fetch all provided lists and update the local database.
//
// - If the entry is not known locally, it is created
// - If a known entry is no longer in the downloaded list, it is marked as deleted.
// - If an entry contains differing data, it will be updated.
func updateLists(ctx context.Context, lists []BDList, database *pgStore) {
	client := NewHTTPClient()
	waitGroup := sync.WaitGroup{}
	errs := make(chan error, len(lists))
	results := make(chan listMapping, len(lists))

	for _, list := range lists {
		if list.Deleted {
			continue
		}
		waitGroup.Add(1)
		go func(lCtx context.Context, localList BDList) {
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
			slog.Error("failed to update entries", ErrAttr(errUpdate))
		}
	}
}

var (
	errUpdateEntryFailed = errors.New("failed to commit updated bd entry")
	errDeleteEntryFailed = errors.New("failed to commit deleted bd entry")
	errPlayerGetOrCreate = errors.New("failed to get/create player record")
)

func updateListEntries(ctx context.Context, database *pgStore, mapping listMapping) error {
	existingList, errExisting := database.bdListEntries(ctx, mapping.list.BDListID)
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
		if _, err := database.bdListEntryCreate(ctx, entry); err != nil {
			if errors.Is(err, errDuplicate) {
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
		if err := database.bdListEntryUpdate(ctx, updated); err != nil {
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
func findNewAndUpdated(existingList []BDListEntry, mapping listMapping) ([]BDListEntry, []BDListEntry) {
	var (
		newEntries []BDListEntry
		updated    []BDListEntry
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
				els := existing.LastSeen.Unix()
				ls := lastSeen.Unix()
				if existing.LastName != player.LastSeen.PlayerName || els != ls || !slices.Equal(existing.Attributes, attrs) || !slices.Equal(existing.Proof, player.Proof) {
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
			newEntry := BDListEntry{
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
func findDeleted(existingList []BDListEntry, mapping listMapping) []BDListEntry {
	var deleted []BDListEntry
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

func doListUpdate(ctx context.Context, database *pgStore) {
	lists, errLists := database.bdLists(ctx)
	if errLists != nil {
		slog.Error("failed to load lists", ErrAttr(errLists))

		return
	}
	updateLists(ctx, lists, database)
}

func listUpdater(ctx context.Context, database *pgStore) {
	ticker := time.NewTicker(time.Hour * 6)

	doListUpdate(ctx, database)

	for {
		select {
		case <-ticker.C:
			doListUpdate(ctx, database)
		case <-ctx.Done():
			return
		}
	}
}
