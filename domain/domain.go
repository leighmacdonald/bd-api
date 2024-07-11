// Package domain is a separate package so that its struct definitions can be imported by other projects
package domain

import (
	"encoding/json"
	"time"

	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/leighmacdonald/steamweb/v2"
)

type Site string

const (
	Skial            Site = "skial"
	GFL              Site = "gfl"
	Spaceship        Site = "spaceship"
	UGC              Site = "ugc"
	SirPlease        Site = "sirplease"
	Vidyagaems       Site = "vidyagaems"
	OWL              Site = "owl"
	ZMBrasil         Site = "zmbrasil"
	Dixigame         Site = "dixigame"
	ScrapTF          Site = "scraptf"
	Wonderland       Site = "wonderland"
	LazyPurple       Site = "lazypurple"
	FirePowered      Site = "firepowered"
	Harpoon          Site = "harpoongaming"
	Panda            Site = "panda"
	NeonHeights      Site = "neonheights"
	Pancakes         Site = "pancakes"
	Loos             Site = "loos"
	PubsTF           Site = "pubstf"
	ServiLiveCl      Site = "servilivecl"
	CutiePie         Site = "cutiepie"
	SGGaming         Site = "sggaming"
	ApeMode          Site = "apemode"
	MaxDB            Site = "maxdb"
	SvdosBrothers    Site = "svdosbrothers"
	Electric         Site = "electric"
	GlobalParadise   Site = "globalparadise"
	SavageServidores Site = "savageservidores"
	CSIServers       Site = "csiservers"
	LBGaming         Site = "lbgaming"
	FluxTF           Site = "fluxtf"
	DarkPyro         Site = "darkpyro"
	OpstOnline       Site = "opstonline"
	BouncyBall       Site = "bouncyball"
	FurryPound       Site = "furrypound"
	RetroServers     Site = "retroservers"
	SwapShop         Site = "swapshop"
	ECJ              Site = "ecj"
	JumpAcademy      Site = "jumpacademy"
	TF2Ro            Site = "tf2ro"
	SameTeem         Site = "sameteem"
	PowerFPS         Site = "powerfps"
	SevenMau         Site = "7mau"
	GhostCap         Site = "ghostcap"
	Spectre          Site = "spectre"
	DreamFire        Site = "dreamfire"
	Setti            Site = "setti"
	GunServer        Site = "gunserver"
	HellClan         Site = "hellclan"
	Sneaks           Site = "sneaks"
	Nide             Site = "nide"
	AstraMania       Site = "astramania"
	TF2Maps          Site = "tf2maps"
	PetrolTF         Site = "petroltf"
	VaticanCity      Site = "vaticancity"
	LazyNeer         Site = "lazyneer"
	TheVille         Site = "theville"
	Oreon            Site = "oreon"
	TriggerHappy     Site = "triggerhappy"
	Defusero         Site = "defusero"
	Tawerna          Site = "tawerna"
	TitanTF          Site = "titan"
	DiscFF           Site = "discff"
	Otaku            Site = "otaku"
	AMSGaming        Site = "amsgaming"
	BaitedCommunity  Site = "baitedcommunity"
	CedaPug          Site = "cedapug"
	GameSites        Site = "gamesites"
	BachuruServas    Site = "bachuruservas"
	Bierwiese        Site = "bierwiese"
	AceKill          Site = "acekill"
	Magyarhns        Site = "magyarhns"
	GamesTown        Site = "gamestown"
	ProGamesZet      Site = "progameszet"
	G44              Site = "g44"
	CuteProject      Site = "cuteproject"
	PhoenixSource    Site = "phoenixsource"
	SlavonServer     Site = "slavonserver"
	GetSome          Site = "getsome"
	Rushy            Site = "rushy"
	MoeVsMachine     Site = "moevsmachine"
	Prwh             Site = "prwh"
	Vortex           Site = "vortex"
	CasualFun        Site = "casualfun"
	RandomTF2        Site = "randomtf2"
	PlayersRo        Site = "playesro"
	EOTLGaming       Site = "eotlgaming"
	BioCrafting      Site = "biocrafting"
	BigBangGamers    Site = "bigbanggamers"
	EpicZone         Site = "epiczone"
	Zubat            Site = "zubat"
	Lunario          Site = "lunario"
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
	NameID      int64           `json:"name_id"`
	SteamID     steamid.SteamID `json:"steam_id"`
	PersonaName string          `json:"persona_name"`
	CreatedOn   time.Time       `json:"created_on"`
}

type PlayerAvatarRecord struct {
	AvatarID   int64           `json:"avatar_id"`
	SteamID    steamid.SteamID `json:"steam_id"`
	AvatarHash string          `json:"avatar_hash"`
	CreatedOn  time.Time       `json:"created_on"`
}

type PlayerVanityRecord struct {
	VanityID  int64           `json:"vanity_id"`
	SteamID   steamid.SteamID `json:"steam_id"`
	Vanity    string          `json:"vanity"`
	CreatedOn time.Time       `json:"created_on"`
}

type Player struct {
	SteamID                  steamid.SteamID          `json:"steam_id"`
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
	BanID       int             `json:"ban_id"`
	SiteName    Site            `json:"site_name"`
	SiteID      int             `json:"site_id"`
	PersonaName string          `json:"persona_name"`
	SteamID     steamid.SteamID `json:"steam_id"`
	Reason      string          `json:"reason"`
	Duration    time.Duration   `json:"duration"`
	Permanent   bool            `json:"permanent"`
	TimeStamped
}

type SbSite struct {
	SiteID int  `json:"site_id"`
	Name   Site `json:"name"`
	TimeStamped
}

// Profile is a high level meta profile of several services.
type Profile struct {
	Summary    steamweb.PlayerSummary  `json:"summary"`
	BanState   steamweb.PlayerBanState `json:"ban_state"`
	Seasons    []Season                `json:"seasons"`
	Friends    []steamweb.Friend       `json:"friends"`
	SourceBans []SbBanRecord           `json:"source_bans"`
	ServeMe    *ServeMeRecord          `json:"serve_me"`
	LogsCount  int                     `json:"logs_count"`
}

// Division tries to define a generalized ranked division order.
type Division int

// *Rough* mapping of skill for each division for sorting, 0 being invite.
const (
	RGLRankInvite       Division = 0
	ETF2LPremiership    Division = 0
	UGCRankPlatinum     Division = 1
	ETF2LDiv1           Division = 1
	RGLRankDiv1         Division = 1
	RGLRankDiv2         Division = 1
	ETF2LDiv2           Division = 2
	RGLRankMain         Division = 2
	RGLRankAdvanced     Division = 2
	ETF2LMid            Division = 3
	UGCRankGold         Division = 3
	ETF2LLow            Division = 4
	RGLRankIntermediate Division = 4
	ETF2LOpen           Division = 5
	RGLRankOpen         Division = 5
	UGCRankSilver       Division = 6
	RGLRankAmateur      Division = 6
	UGCRankSteel        Division = 7
	UGCRankIron         Division = 8
	RGLRankFreshMeat    Division = 9
	RGLRankNone         Division = 10
	UGCRankNone         Division = 10
	UnknownDivision     Division = 20
)

// League represents supported leagues.
type League string

//
// const (
//	leagueUGC   League = "ugc"
//	leagueESEA  League = "esea"
//	leagueETF2L League = "etf2l"
//	leagueRGL   League = "rgl"
//)

// Season stores generalized league season data.
type Season struct {
	League      League   `json:"league"`
	Division    string   `json:"division"`
	DivisionInt Division `json:"division_int"`
	Format      string   `json:"format"`
	TeamName    string   `json:"team_name"`
}

type CompMap map[steamid.SteamID][]Season

// JSONDuration handles encoding time.Duration values into seconds.
type JSONDuration struct {
	time.Duration
}

func (d JSONDuration) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.Seconds()) //nolint:wrapcheck
}

type LogsTFMatchInfo struct {
	LogID        int          `json:"log_id"`
	Title        string       `json:"title"`
	Map          string       `json:"map"`
	Format       string       `json:"format"`
	Views        int          `json:"-"`
	Duration     JSONDuration `json:"duration"`
	ScoreRED     int          `json:"score_red"`
	ScoreBLU     int          `json:"score_blu"`
	CreatedOn    time.Time    `json:"created_on"`
	LogFormatOld bool         `json:"-"`
}

type LogsTFMatch struct {
	LogsTFMatchInfo

	Rounds  []LogsTFRound  `json:"rounds"`
	Players []LogsTFPlayer `json:"players"`
	Medics  []LogsTFMedic  `json:"medics"`
}

type PlayerClass int

const (
	Spectator PlayerClass = iota
	Scout
	Soldier
	Pyro
	Demo
	Heavy
	Engineer
	Medic
	Sniper
	Spy
)

type Team int

const (
	RED Team = 3
	BLU Team = 4
)

type LogsTFPlayer struct {
	LogID        int                 `json:"-"`
	SteamID      steamid.SteamID     `json:"steam_id"`
	Team         Team                `json:"team"`
	Name         string              `json:"name"`
	Classes      []LogsTFPlayerClass `json:"classes"`
	Kills        int                 `json:"kills"`
	Assists      int                 `json:"assists"`
	Deaths       int                 `json:"deaths"`
	Damage       int64               `json:"damage"`
	DPM          int                 `json:"dpm"`
	KAD          float32             `json:"kad"`
	KD           float32             `json:"kd"`
	DamageTaken  int                 `json:"damage_taken"`
	DTM          int                 `json:"dtm"`
	HealthPacks  int                 `json:"health_packs"`
	Backstabs    int                 `json:"backstabs"`
	Headshots    int                 `json:"headshots"`
	Airshots     int                 `json:"airshots"`
	Caps         int                 `json:"caps"`
	HealingTaken int                 `json:"healing_taken"`
}

type LogsTFRound struct {
	LogID     int          `json:"-"`
	Round     int          `json:"round"`
	Length    JSONDuration `json:"length"`
	ScoreBLU  int          `json:"score_blu"`
	ScoreRED  int          `json:"score_red"`
	KillsBLU  int          `json:"kills_blu"`
	KillsRED  int          `json:"kills_red"`
	UbersBLU  int          `json:"ubers_blu"`
	UbersRED  int          `json:"ubers_red"`
	DamageBLU int          `json:"damage_blu"`
	DamageRED int          `json:"damage_red"`
	MidFight  Team         `json:"mid_fight,omitempty"`
}

type LogsTFPlayerClass struct {
	LogID   int                       `json:"-"`
	SteamID steamid.SteamID           `json:"steam_id"`
	Class   PlayerClass               `json:"class"`
	Played  JSONDuration              `json:"played"`
	Kills   int                       `json:"kills"`
	Assists int                       `json:"assists"`
	Deaths  int                       `json:"deaths"`
	Damage  int                       `json:"damage"`
	Weapons []LogsTFPlayerClassWeapon `json:"weapons"`
}

type LogsTFPlayerClassWeapon struct {
	LogID    int             `json:"-"`
	SteamID  steamid.SteamID `json:"steam_id"`
	Weapon   string          `json:"weapon,omitempty"`
	Kills    int             `json:"kills,omitempty"`
	Damage   int             `json:"damage,omitempty"`
	Accuracy int             `json:"accuracy,omitempty"`
}

type LogsTFMedic struct {
	LogID            int             `json:"-"`
	SteamID          steamid.SteamID `json:"steam_id"`
	Healing          int64           `json:"healing"`
	HealingPerMin    int             `json:"healing_per_min"`
	ChargesKritz     int             `json:"charges_kritz"`
	ChargesQuickfix  int             `json:"charges_quickfix"`
	ChargesMedigun   int             `json:"charges_medigun"`
	ChargesVacc      int             `json:"charges_vacc"`
	Drops            int             `json:"drops"`
	AvgTimeBuild     JSONDuration    `json:"avg_time_build"`
	AvgTimeUse       JSONDuration    `json:"avg_time_use"`
	NearFullDeath    int             `json:"near_full_death"`
	AvgUberLen       JSONDuration    `json:"avg_uber_len"`
	DeathAfterCharge int             `json:"death_after_charge"`
	MajorAdvLost     int             `json:"major_adv_lost"`
	BiggestAdvLost   JSONDuration    `json:"biggest_adv_lost"`
}

type LogsTFPlayerSums struct {
	KillsSum        int   `json:"kills_sum"`
	AssistsSum      int   `json:"assists_sum"`
	DeathsSum       int   `json:"deaths_sum"`
	DamageSum       int64 `json:"damage_sum"`
	DamageTakenSum  int   `json:"damage_taken_sum"`
	HealthPacksSum  int   `json:"health_packs_sum"`
	BackstabsSum    int   `json:"backstabs_sum"`
	HeadshotsSum    int   `json:"headshots_sum"`
	AirshotsSum     int   `json:"airshots_sum"`
	CapsSum         int   `json:"caps_sum"`
	HealingTakenSum int   `json:"healing_taken_sum"`
}

type LogsTFPlayerAverages struct {
	KillsAvg        float32 `json:"kills_avg"`
	AssistsAvg      float32 `json:"assists_avg"`
	DeathsAvg       float32 `json:"deaths_avg"`
	DamageAvg       float32 `json:"damage_avg"`
	DPMAvg          float32 `json:"dpm_avg"`
	KADAvg          float32 `json:"kad_avg"`
	KDAvg           float32 `json:"kd_avg"`
	DamageTakenAvg  float32 `json:"damage_taken_avg"`
	DTMAvg          float32 `json:"dtm_avg"`
	HealthPacksAvg  float32 `json:"health_packs_avg"`
	BackstabsAvg    float32 `json:"backstabs_avg"`
	HeadshotsAvg    float32 `json:"headshots_avg"`
	AirshotsAvg     float32 `json:"airshots_avg"`
	CapsAvg         float32 `json:"caps_avg"`
	HealingTakenAvg float32 `json:"healing_taken_avg"`
}
type LogsTFPlayerSummary struct {
	Logs int `json:"logs"`
	LogsTFPlayerAverages
	LogsTFPlayerSums
}

type ServeMeRecord struct {
	SteamID steamid.SteamID `json:"steam_id"`
	Name    string          `json:"name"`
	Reason  string          `json:"reason"`
	Deleted bool            `json:"deleted"`
	TimeStamped
}
