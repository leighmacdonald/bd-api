// Package models provides exported models that can be used by client applications.
package models

import (
	"time"

	"github.com/leighmacdonald/steamid/v3/steamid"
	"github.com/leighmacdonald/steamweb/v2"
)

type SiteName string

const (
	Skial            SiteName = "skial"
	GFL              SiteName = "gfl"
	Spaceship        SiteName = "spaceship"
	UGC              SiteName = "ugc"
	SirPlease        SiteName = "sirplease"
	Vidyagaems       SiteName = "vidyagaems"
	OWL              SiteName = "owl"
	ZMBrasil         SiteName = "zmbrasil"
	Dixigame         SiteName = "dixigame"
	ScrapTF          SiteName = "scraptf"
	Wonderland       SiteName = "wonderland"
	LazyPurple       SiteName = "lazypurple"
	FirePowered      SiteName = "firepowered"
	Harpoon          SiteName = "harpoongaming"
	Panda            SiteName = "panda"
	NeonHeights      SiteName = "neonheights"
	Pancakes         SiteName = "pancakes"
	Loos             SiteName = "loos"
	PubsTF           SiteName = "pubstf"
	ServiLiveCl      SiteName = "servilivecl"
	CutiePie         SiteName = "cutiepie"
	SGGaming         SiteName = "sggaming"
	ApeMode          SiteName = "apemode"
	MaxDB            SiteName = "maxdb"
	SvdosBrothers    SiteName = "svdosbrothers"
	Electric         SiteName = "electric"
	GlobalParadise   SiteName = "globalparadise"
	SavageServidores SiteName = "savageservidores"
	CSIServers       SiteName = "csiservers"
	LBGaming         SiteName = "lbgaming"
	FluxTF           SiteName = "fluxtf"
	DarkPyro         SiteName = "darkpyro"
	OpstOnline       SiteName = "opstonline"
	BouncyBall       SiteName = "bouncyball"
	FurryPound       SiteName = "furrypound"
	RetroServers     SiteName = "retroservers"
	SwapShop         SiteName = "swapshop"
	ECJ              SiteName = "ecj"
	JumpAcademy      SiteName = "jumpacademy"
	TF2Ro            SiteName = "tf2ro"
	SameTeem         SiteName = "sameteem"
	PowerFPS         SiteName = "powerfps"
	SevenMau         SiteName = "7mau"
	GhostCap         SiteName = "ghostcap"
	Spectre          SiteName = "spectre"
	DreamFire        SiteName = "dreamfire"
	Setti            SiteName = "setti"
	GunServer        SiteName = "gunserver"
	HellClan         SiteName = "hellclan"
	Sneaks           SiteName = "sneaks"
	Nide             SiteName = "nide"
	AstraMania       SiteName = "astramania"
	TF2Maps          SiteName = "tf2maps"
	PetrolTF         SiteName = "petroltf"
	VaticanCity      SiteName = "vaticancity"
	LazyNeer         SiteName = "lazyneer"
	TheVille         SiteName = "theville"
	Oreon            SiteName = "oreon"
	TriggerHappy     SiteName = "triggerhappy"
	Defusero         SiteName = "defusero"
	Tawerna          SiteName = "tawerna"
	TitanTF          SiteName = "titan"
	DiscFF           SiteName = "discff"
	Otaku            SiteName = "otaku"
	AMSGaming        SiteName = "amsgaming"
	BaitedCommunity  SiteName = "baitedcommunity"
	CedaPug          SiteName = "cedapug"
	GameSites        SiteName = "gamesites"
	BachuruServas    SiteName = "bachuruservas"
	Bierwiese        SiteName = "bierwiese"
	AceKill          SiteName = "acekill"
	Magyarhns        SiteName = "magyarhns"
	GamesTown        SiteName = "gamestown"
	ProGamesZet      SiteName = "progameszet"
	G44              SiteName = "g44"
	CuteProject      SiteName = "cuteproject"
	PhoenixSource    SiteName = "phoenixsource"
	SlavonServer     SiteName = "slavonserver"
	GetSome          SiteName = "getsome"
	Rushy            SiteName = "rushy"
	MoeVsMachine     SiteName = "moevsmachine"
	Prwh             SiteName = "prwh"
	Vortex           SiteName = "vortex"
	Casualness       SiteName = "casualness"
	RandomTF2        SiteName = "randomtf2"
	PlayersRo        SiteName = "playesro"
	EOTLGaming       SiteName = "eotlgaming"
	BioCrafting      SiteName = "biocrafting"
	BigBangGamers    SiteName = "bigbanggamers"
	EpicZone         SiteName = "epiczone"
	Zubat            SiteName = "zubat"
	Lunario          SiteName = "lunario"
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
	SiteName    SiteName      `json:"site_name"`
	SiteID      int           `json:"site_id"`
	PersonaName string        `json:"persona_name"`
	SteamID     steamid.SID64 `json:"steam_id"`
	Reason      string        `json:"reason"`
	Duration    time.Duration `json:"duration"`
	Permanent   bool          `json:"permanent"`
	TimeStamped
}

type SbSite struct {
	SiteID int      `json:"site_id"`
	Name   SiteName `json:"name"`
	TimeStamped
}
