package main

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/leighmacdonald/steamid/v3/steamid"
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

type Players struct {
	Attributes []string `json:"attributes"`
	LastSeen   LastSeen `json:"last_seen,omitempty"`
	Steamid    any      `json:"steamid"`
	Proof      []string `json:"proof,omitempty"`
}

type TF2BDSchema struct {
	Schema   string    `json:"$schema"` //nolint:tagliatelle
	FileInfo FileInfo  `json:"file_info"`
	Players  []Players `json:"players"`
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
	SteamID       steamid.SID64
	Attribute     string
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

	defer resp.Body.Close()

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
// - If a entry contains differing data, it will be updated.
func updateLists(ctx context.Context, lists []BDList, db *pgStore) {
	client := NewHTTPClient()
	wg := sync.WaitGroup{}
	errs := make(chan error, len(lists))
	results := make(chan listMapping, len(lists))

	for _, list := range lists {
		if list.Deleted {
			continue
		}
		wg.Add(1)
		go func(l BDList) {
			defer wg.Done()
			bdList, errFetch := fetchList(ctx, client, l)
			if errFetch != nil {
				errs <- errFetch
				return
			}
			results <- listMapping{list: l, result: bdList}
		}(list)
	}
	go func() {
		wg.Wait()
		close(results)
		close(errs)
	}()

	for err := range errs {
		slog.Error(err.Error())
	}

	for res := range results {
		if errUpdate := updateListEntries(ctx, db, res); errUpdate != nil {
			slog.Error("failed to update entries", ErrAttr(errUpdate))
		}
	}
}

var (
	errUpdateEntryFailed = errors.New("failed to commit updated bd entry")
	errCreateEntryFailed = errors.New("failed to commit created bd entry")
	errDeleteEntryFailed = errors.New("failed to commit deleted bd entry")
)

func updateListEntries(ctx context.Context, db *pgStore, mapping listMapping) error {
	existingList, errExisting := db.bdListEntries(ctx, mapping.list.BDListID)
	if errExisting != nil {
		return errExisting
	}

	newEntries, updatedEntries := findNewAndUpdated(existingList, mapping)
	deletedEntries := findDeleted(existingList, mapping)

	for _, entry := range newEntries {
		if _, err := db.bdListEntryCreate(ctx, entry); err != nil {
			return errors.Join(err, errCreateEntryFailed)
		}
	}
	if len(newEntries) > 0 {
		slog.Info("Added new list entries", slog.Int("count", len(newEntries)),
			slog.String("list", mapping.list.BDListName), slog.Int("bd_list_id", mapping.list.BDListID))
	}

	for _, updated := range updatedEntries {
		if err := db.bdListEntryUpdate(ctx, updated); err != nil {
			return errors.Join(err, errUpdateEntryFailed)
		}
	}
	if len(updatedEntries) > 0 {
		slog.Info("Updated list entries", slog.Int("count", len(updatedEntries)),
			slog.String("list", mapping.list.BDListName), slog.Int("bd_list_id", mapping.list.BDListID))
	}

	for _, entry := range deletedEntries {
		if err := db.bdListEntryDelete(ctx, entry.BDListEntryID); err != nil {
			return errors.Join(err, errDeleteEntryFailed)
		}
	}
	if len(deletedEntries) > 0 {
		slog.Info("Deleted list entries", slog.Int("count", len(deletedEntries)),
			slog.String("list", mapping.list.BDListName), slog.Int("bd_list_id", mapping.list.BDListID))
	}

	return nil
}

// Search results for existing entries with new attrs
func findNewAndUpdated(existingList []BDListEntry, mapping listMapping) ([]BDListEntry, []BDListEntry) {
	var (
		newEntries []BDListEntry
		updated    []BDListEntry
	)

	for _, player := range mapping.result.Players {
		sid := steamid.New(player.Steamid)
		if !sid.Valid() {
			slog.Warn("got invalid steam id", slog.Any("sid", sid))
			continue
		}
		found := false
		for _, existing := range existingList {
			if existing.SteamID == sid {
				found = true
				lastSeen := time.Unix(int64(player.LastSeen.Time), 0)
				attrs := strings.Join(player.Attributes, ",")
				els := existing.LastSeen.Unix()
				ls := lastSeen.Unix()
				if existing.LastName != player.LastSeen.PlayerName || els != ls || existing.Attribute != attrs {
					u := existing
					u.LastSeen = lastSeen
					u.Attribute = attrs
					u.LastName = player.LastSeen.PlayerName
					u.UpdatedOn = time.Now()
					updated = append(updated, u)
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
				Attribute:     strings.Join(player.Attributes, ","),
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

// Search results for deleted entries
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

func doListUpdate(ctx context.Context, db *pgStore) {
	lists, errLists := db.bdLists(ctx)
	if errLists != nil {
		slog.Error("failed to load lists", ErrAttr(errLists))
		return
	}
	updateLists(ctx, lists, db)
}

func listUpdater(ctx context.Context, db *pgStore) {
	ticker := time.NewTicker(time.Hour * 6)

	sync.OnceFunc(func() {
		doListUpdate(ctx, db)
	})

	for {
		select {
		case <-ticker.C:
			doListUpdate(ctx, db)
		case <-ctx.Done():
			return
		}
	}
}
