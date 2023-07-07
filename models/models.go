// Package models provides exported models that can be used by client applications.
package models

import (
	"time"

	"github.com/leighmacdonald/steamid/v3/steamid"
	"github.com/leighmacdonald/steamweb/v2"
)

type EconBanState int

const (
	EconBanNone EconBanState = iota
	EconBanProbation
	EconBanBanned
)

type TimeStamped struct {
	UpdatedOn time.Time `json:"-"`
	CreatedOn time.Time `json:"created_on"`
}

type PlayerNameRecord struct {
	NameID      int64         `json:"name_id"`
	SteamID     steamid.SID64 `json:"steam_id"`
	PersonaName string        `json:"persona_name"`
	CreatedOn   time.Time     `json:"created_on"`
}

type PlayerAvatarRecord struct {
	AvatarID   int64         `json:"avatar_id"`
	SteamID    steamid.SID64 `json:"steam_id"`
	AvatarHash string        `json:"avatar_hash"`
	CreatedOn  time.Time     `json:"created_on"`
}

type PlayerVanityRecord struct {
	VanityID  int64         `json:"vanity_id"`
	SteamID   steamid.SID64 `json:"steam_id"`
	Vanity    string        `json:"vanity"`
	CreatedOn time.Time     `json:"created_on"`
}

type Player struct {
	SteamID                  steamid.SID64            `json:"steam_id"`
	CommunityVisibilityState steamweb.VisibilityState `json:"community_visibility_state"`
	ProfileState             steamweb.ProfileState    `json:"profile_state"`
	PersonaName              string                   `json:"persona_name"`
	Vanity                   string                   `json:"vanity"`
	AvatarHash               string                   `json:"avatar_hash"`
	PersonaState             steamweb.PersonaState    `json:"persona_state"`
	RealName                 string                   `json:"real_name"`
	TimeCreated              time.Time                `json:"time_created"`
	LocCountryCode           string                   `json:"loc_country_code"`
	LocStateCode             string                   `json:"loc_state_code"`
	LocCityID                int                      `json:"loc_city_id"`
	CommunityBanned          bool                     `json:"community_banned"`
	VacBanned                bool                     `json:"vac_banned"`
	LastBannedOn             time.Time                `json:"last_banned_on"`
	GameBans                 int                      `json:"game_bans"`
	EconomyBanned            EconBanState             `json:"economy_banned"`
	LogsTFCount              int                      `json:"logs_tf_count"`
	UGCUpdatedOn             time.Time                `json:"ugc_updated_on"`
	RGLUpdatedOn             time.Time                `json:"rgl_updated_on"`
	ETF2LUpdatedOn           time.Time                `json:"etf2_l_updated_on"`
	LogsTFUpdatedOn          time.Time                `json:"logs_tf_updated_on"`
	TimeStamped
}

type SbBanRecord struct {
	BanID       int           `json:"ban_id"`
	SiteName    string        `json:"site_name"`
	SiteID      int           `json:"site_id"`
	PersonaName string        `json:"persona_name"`
	SteamID     steamid.SID64 `json:"steam_id"`
	Reason      string        `json:"reason"`
	Duration    time.Duration `json:"duration"`
	Permanent   bool          `json:"permanent"`
	TimeStamped
}

type SbSite struct {
	SiteID int    `json:"site_id"`
	Name   string `json:"name"`
	TimeStamped
}
