package main

// Division tries to define a generalized ranked division order
type Division int

// *Rough* mapping of skill for each division for sorting, 0 being invite
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

// League represents supported leagues
type League string

//
//const (
//	leagueUGC   League = "ugc"
//	leagueESEA  League = "esea"
//	leagueETF2L League = "etf2l"
//	leagueRGL   League = "rgl"
//)

// Season stores generalized league season data
type Season struct {
	League      League   `json:"league"`
	Division    string   `json:"division"`
	DivisionInt Division `json:"division_int"`
	Format      string   `json:"format"`
	TeamName    string   `json:"team_name"`
}
