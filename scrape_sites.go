package main

import (
	"context"
	"fmt"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/leighmacdonald/bd-api/models"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

func newSkialScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, models.Skial, "https://www.skial.com/sourcebans/", "",
		parseDefault, nextURLFirst, parseSkialTime,
	)
}

func newGFLScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, models.GFL, "https://sourcebans.gflclan.com/", "",
		parseDefault, nextURLLast, parseDefaultTime)
}

func newSpaceShipScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, models.Spaceship, "https://sappho.io/bans/", "",
		parseDefault, nextURLLast, parseDefaultTime)
}

func newUGCScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, models.UGC, "https://sb.ugc-gaming.net/", "",
		parseFluent, nextURLFluent, parseDefaultTime)
}

func newSirPleaseScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, models.SirPlease, "https://sirplease.gg/", "",
		parseFluent, nextURLFluent, parseDefaultTime)
}

func newVidyaGaemsScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, models.Vidyagaems, "https://www.vidyagaems.net/sourcebans/", "",
		parseFluent, nextURLFluent, parseTrailYear)
}

func newOwlTFScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, models.OWL, "https://kingpandagamer.xyz/sb/", "",
		parseDefault, nextURLLast, parseDefaultTime)
}

func newZMBrasilScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, models.ZMBrasil, "http://bans.zmbrasil.com.br/", "",
		parseDefault, nextURLLast, parseSkialTime)
}

func newDixiGameScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, models.Dixigame, "https://dixigame.com/bans/", "",
		parseDefault, nextURLLast, parseDefaultTime)
}

func newScrapTFScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, models.ScrapTF, "https://bans.scrap.tf/", "",
		parseDefault, nextURLLast, parseDefaultTime)
}

func newWonderlandTFScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	const siteSleepTime = time.Second * 10

	transport := newCFTransport()

	if errOpen := transport.Open(context.Background()); errOpen != nil {
		return nil, errors.Wrap(errOpen, "Failed to setup browser")
	}

	scraper, errScraper := newScraperWithTransport(logger, cacheDir,
		models.Wonderland,
		"https://bans.wonderland.tf/",
		"",
		parseDefault,
		nextURLLast,
		parseWonderlandTime,
		transport)
	if errScraper != nil {
		return nil, errScraper
	}

	scraper.sleepTime = siteSleepTime

	// Cached versions do not have a proper next link, so we have to generate one.
	scraper.nextURL = func(scraper *sbScraper, doc *goquery.Selection) string {
		scraper.curPage++

		return scraper.url(fmt.Sprintf("index.php?p=banlist&page=%d", scraper.curPage))
	}

	return scraper, nil
}

func newLazyPurpleScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, models.LazyPurple, "https://www.lazypurple.com/sourcebans/", "",
		parseDefault, nextURLLast, parseDefaultTime)
}

func newFirePoweredScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, models.FirePowered, "https://firepoweredgaming.com/sourcebanspp/", "",
		parseDefault, nextURLLast, parseSkialTime)
}

func newHarpoonScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, models.Harpoon, "https://bans.harpoongaming.com/", "",
		parseDefault, nextURLLast, parseDefaultTime)
}

func newPandaScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, models.Panda, "https://bans.panda-community.com/", "",
		parseFluent, nextURLFluent, parseDefaultTime)
}

func newNeonHeightsScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, models.NeonHeights, "https://neonheights.xyz/bans/", "",
		parseDefault, nextURLLast, parseSkialTime)
}

func newPancakesScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, models.Pancakes, "https://pancakes.tf/", "",
		parseDefault, nextURLLast, parsePancakesTime)
}

func newLOOSScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, models.Loos, "https://looscommunity.com/bans/", "",
		parseDefault, nextURLLast, parseDefaultTime)
}

func newPubsTFScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, models.PubsTF, "https://bans.pubs.tf/", "",
		parseDefault, nextURLLast, parseSkialTime)
}

func newServiliveClScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, models.ServiLiveCl, "https://sourcebans.servilive.cl/", "",
		parseFluent, nextURLFluent, parseDefaultTimeMonthFirst)
}

func newCutiePieScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, models.CutiePie, "https://bans.cutiepie.tf/", "",
		parseDefault, nextURLLast, parseDefaultTime)
}

func newSGGamingScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, models.SGGaming, "https://sg-gaming.net/bans/", "",
		parseDefault, nextURLLast, parseSGGamingTime)
}

func newApeModeScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, models.ApeMode, "https://sourcebans.apemode.tf/", "",
		parseDefault, nextURLLast, parseSkialTime)
}

func newMaxDBScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, models.MaxDB, "https://bans.maxdb.net/", "",
		parseDefault, nextURLLast, parseDefaultTime)
}

func newSvdosBrothersScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, models.SvdosBrothers, "https://bans.svdosbrothers.com/", "",
		parseFluent, nextURLFluent, parseSVDos)
}

func newElectricScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, models.Electric, "http://168.181.184.179/", "",
		parseFluent, nextURLFluent, parseDefaultTime)
}

func newGlobalParadiseScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, models.GlobalParadise, "https://bans.theglobalparadise.org/", "",
		parseDefault, nextURLLast, parseDefaultTime)
}

func newSavageServidoresScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, models.SavageServidores, "https://bans.savageservidores.com/", "",
		parseFluent, nextURLFluent, parseDefaultTime)
}

func newCSIServersScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, models.CSIServers, "https://bans.csiservers.com/", "",
		parseDefault, nextURLLast, parseDefaultTime)
}

func newLBGamingScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, models.LBGaming, "https://bans.lbgaming.co/", "",
		parseDefault, nextURLLast, parseSkialTime)
}

func newFluxTFScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, models.FluxTF, "https://bans.flux.tf/", "",
		parseDefault, nextURLLast, parseFluxTime)
}

func newDarkPyroScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, models.DarkPyro, "https://bans.darkpyrogaming.com/", "",
		parseDefault, nextURLLast, parseDarkPyroTime)
}

func newOpstOnlineScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, models.OpstOnline, "https://www.opstonline.com/bans/", "",
		parseDefault, nextURLLast, parseSkialTime)
}

func newBouncyBallScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, models.BouncyBall, "https://www.bouncyball.eu/bans2/", "",
		parseDefault, nextURLLast, parseSkialTime)
}

func newFurryPoundScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, models.FurryPound, "http://sourcebans.thefurrypound.org/", "",
		parseDefault, nextURLLast, parseFurryPoundTime)
}

func newRetroServersScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, models.RetroServers, "https://bans.retroservers.net/", "",
		parseDefault, nextURLLast, parseDefaultTime)
}

func newSwapShopScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, models.SwapShop, "http://tf2swapshop.com/sourcebans/", "",
		parseDefault, nextURLLast, parseSkialTime)
}

func newECJScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, models.ECJ, "https://ecj.tf/sourcebans/", "",
		parseDefault, nextURLLast, parseSkialTime)
}

func newJumpAcademyScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, models.JumpAcademy, "https://bans.jumpacademy.tf/", "",
		parseDefault, nextURLLast, parseDefaultTime)
}

func newTF2ROScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	// Not enough values to page yet...
	return newScraper(logger, cacheDir, models.TF2Ro, "https://bans.tf2ro.com/", "",
		parseDefault, nextURLLast, parseDefaultTime)
}

func newSameTeemScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, models.SameTeem, "https://sameteem.com/sourcebans/", "",
		parseDefault, nextURLLast, parseDefaultTime)
}

func newPowerFPSScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, models.PowerFPS, "https://bans.powerfps.com/", "",
		parseDefault, nextURLLast, parseSkialTime)
}

func new7MauScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, models.SevenMau, "https://7-mau.com/server/", "",
		parseFluent, nextURLFluent, parseDefaultTime)
}

func newGhostCapScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, models.GhostCap, "https://sourcebans.ghostcap.com/", "",
		parseDefault, nextURLLast, parseDefaultTime)
}

func newSpectreScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, models.Spectre, "https://spectre.gg/bans/", "",
		parseDefault, nextURLLast, parseDefaultTime)
}

func newDreamFireScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, models.DreamFire, "https://sourcebans.dreamfire.fr/", "",
		parseDefault, nextURLLast, parseDefaultTime)
}

func newSettiScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, models.Setti, "https://pong.setti.info/sourcebans/", "",
		parseDefault, nextURLLast, parseDefaultTime)
}

func newGunServerScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, models.GunServer, "https://gunserver.ru/sourcebans/", "",
		parseDefault, nextURLFirst, parseGunServer)
}

func newHellClanScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, models.HellClan, "https://hellclan.co.uk/sourcebans/", "",
		parseDefault, nextURLLast, parseHellClanTime)
}

func newSneaksScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, models.Sneaks, "https://bans.snksrv.com/", "",
		parseDefault, nextURLLast, parseSneakTime)
}

func newNideScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, models.Nide, "https://bans.nide.gg/", "",
		parseFluent, nextURLFluent, parseDefaultTime)
}

func newAstraManiaScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, models.AstraMania, "https://astramania.ro/sban2/", "",
		parseDefault, nextURLLast, parseTrailYear)
}

func newTF2MapsScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, models.TF2Maps, "https://bans.tf2maps.net/", "",
		parseDefault, nextURLLast, parseDefaultTime)
}

func newPetrolTFScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, models.PetrolTF, "https://petrol.tf/sb/", "",
		parseDefault, nextURLLast, parseDefaultTime)
}

func newVaticanCityScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, models.VaticanCity, "https://www.the-vaticancity.com/sourcebans/", "",
		parseDefault, nextURLLast, parseSkialTime)
}

func newLazyNeerScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, models.LazyNeer, "https://www.lazyneer.com/SourceBans/", "",
		parseDefault, nextURLLast, parseSkialAltTime)
}

func newTheVilleScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, models.TheVille, "https://www.theville.org/sourcebans/", "",
		parseDefault, nextURLLast, parseSkialTime)
}

func newOreonScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, models.Oreon, "https://www.tf2-oreon.fr/sourceban/", "",
		parseDefault, nextURLLast, parseSkialTime)
}

func newTriggerHappyScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, models.TriggerHappy, "https://triggerhappygamers.com/sourcebans/", "",
		parseDefault, nextURLLast, parseTriggerHappyTime)
}

func newDefuseRoScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, models.Defusero, "https://bans.defusero.org/", "",
		parseFluent, nextURLFluent, parseDefaultTime)
}

// Has cloudflare.
// func newTawernaScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
//	return newScraper(logger, cacheDir, Tawerna, "https://sb.tawerna.tf/", "",
//		parseDefault, nextURLLast, parseSkialTime)
// }

func newTitanScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, models.TitanTF, "https://bans.titan.tf/", "",
		parseDefault, nextURLLast, parseTitanTime)
}

func newDiscFFScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, models.DiscFF, "http://disc-ff.site.nfoservers.com/sourcebanstf2/", "",
		parseDefault, nextURLLast, parseDefaultTime)
}

// New theme
// func NewOtakuScraper( ) (*sbScraper, error){
//	return newScraper(logger, cacheDir, Otaku, "https://bans.otaku.tf/bans", "",
//		parseDefault, nextURLLast, parseOtakuTime)
//}

func newAMSGamingScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, models.AMSGaming, "https://bans.amsgaming.in/", "",
		parseStar, nextURLLast, parseAMSGamingTime)
}

func newBaitedCommunityScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, models.BaitedCommunity, "https://bans.baitedcommunity.com/", "",
		parseStar, nextURLLast, parseBaitedTime)
}

func newCedaPugScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, models.CedaPug, "https://cedapug.com/sourcebans/", "",
		parseStar, nextURLLast, parseSkialTime)
}

func newGameSitesScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, models.GameSites, "https://banlist.gamesites.cz/tf2/", "",
		parseStar, nextURLLast, parseSkialTime)
}

func newBachuruServasScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, models.BachuruServas, "https://bachuruservas.lt/sb/", "",
		parseStar, nextURLLast, parseBachuruServasTime)
}

func newBierwieseScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, models.Bierwiese, "http://94.249.194.218/sb/", "",
		parseStar, nextURLLast, parseSkialTime)
}

func newAceKillScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, models.AceKill, "https://sourcebans.acekill.pl/", "",
		parseStar, nextURLLast, parseSkialTime)
}

func newMagyarhnsScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, models.Magyarhns, "https://magyarhns.hu/sourcebans/", "",
		parseStar, nextURLLast, parseSkialTime)
}

func newGamesTownScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, models.GamesTown, "https://banlist.games-town.eu/", "",
		parseStar, nextURLLast, parseTrailYear)
}

func newProGamesZetScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, models.ProGamesZet, "https://bans.progameszet.ru/", "",
		parseMaterial, nextURLLast, parseProGamesZetTime)
}

func newG44Scraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, models.G44, "http://bans.allmaps.g44.rocks/", "",
		parseMaterial, nextURLLast, parseProGamesZetTime)
}

func newCuteProjectScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	const siteSleepTime = time.Second * 4

	scraper, errScraper := newScraper(logger, cacheDir, models.CuteProject, "https://bans.cute-project.net/", "",
		parseMaterial, nextURLLast, parseProGamesZetTime)
	if errScraper != nil {
		return nil, errScraper
	}

	scraper.sleepTime = siteSleepTime

	return scraper, nil
}

func newPhoenixSourceScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, models.PhoenixSource, "https://phoenix-source.ru/sb/", "",
		parseMaterial, nextURLLast, parseProGamesZetTime)
}

func newSlavonServerScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, models.SlavonServer, "http://slavonserver.ru/ma/", "",
		parseMaterial, nextURLLast, parseProGamesZetTime)
}

func newGetSomeScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, models.GetSome, "https://bans.getsome.co.nz/", "",
		parseDefault, nextURLLast, parseSkialTime)
}

func newRushyScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, models.Rushy, "https://sourcebans.rushyservers.com/", "",
		parseDefault, nextURLLast, parseRushyTime)
}

func newMoevsMachineScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, models.MoeVsMachine, "https://moevsmachine.tf/bans/", "",
		parseDefault, nextURLLast, parseDefaultTime)
}

func newPRWHScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, models.Prwh, "https://sourcebans.prwh.de/", "",
		parseDefault, nextURLLast, parsePRWHTime)
}

func newVortexScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, models.Vortex, "http://vortex.oyunboss.net/sourcebans/", "",
		parseStar, nextURLLast, parseSkialTime)
}

func newCasualFunScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, models.Casualness, "https://tf2-casual-fun.de/sourcebans/", "",
		parseDefault, nextURLLast, parsePRWHTime)
}

func newRandomTF2Scraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, models.RandomTF2, "https://bans.randomtf2.com/", "",
		parseDefault, nextURLLast, parseSkialTime)
}

func newPlayesROScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, models.PlayersRo, "https://www.playes.ro/csgobans/", "",
		parseStar, nextURLLast, parseSkialTime)
}

func newEOTLGamingScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, models.EOTLGaming, "https://tf2.endofthelinegaming.com/sourcebans/", "",
		parseDefault, nextURLLast, parseDefaultTime)
}

func newBioCraftingScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, models.BioCrafting, "https://sourcebans.biocrafting.net/", "",
		parseDefault, nextURLLast, parseSkialTime)
}

func newBigBangGamersScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, models.BigBangGamers, "http://208.71.172.9/", "",
		parseDefault, nextURLLast, parseSkialTime)
}

func newEpicZoneScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, models.EpicZone, "https://sourcebans.epiczone.sk/", "",
		parseStar, nextURLLast, parseGunServer)
}

func newZubatScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, models.Zubat, "https://sb.zubat.ru/", "",
		parseStar, nextURLLast, parseSkialTime)
}

func newLunarioScraper(logger *zap.Logger, cacheDir string) (*sbScraper, error) {
	return newScraper(logger, cacheDir, models.Lunario, "https://sb.lunario.ro/", "",
		parseStar, nextURLLast, parseSkialTime)
}
