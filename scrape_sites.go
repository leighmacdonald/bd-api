package main

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/leighmacdonald/bd-api/domain"
)

func newSkialScraper(cacheDir string) (*sbScraper, error) {
	return newScraper(cacheDir, domain.Skial, "https://www.skial.com/sourcebans/", "",
		parseDefault, nextURLFirst, parseSkialTime,
	)
}

func newGFLScraper(cacheDir string) (*sbScraper, error) {
	return newScraper(cacheDir, domain.GFL, "https://sourcebans.gflclan.com/", "",
		parseDefault, nextURLLast, parseDefaultTime)
}

func newSpaceShipScraper(cacheDir string) (*sbScraper, error) {
	return newScraper(cacheDir, domain.Spaceship, "https://sappho.io/bans/", "",
		parseDefault, nextURLLast, parseDefaultTime)
}

func newUGCScraper(cacheDir string) (*sbScraper, error) {
	return newScraper(cacheDir, domain.UGC, "https://sb.ugc-gaming.net/", "",
		parseFluent, nextURLFluent, parseDefaultTime)
}

func newSirPleaseScraper(cacheDir string) (*sbScraper, error) {
	return newScraper(cacheDir, domain.SirPlease, "https://sirplease.gg/", "",
		parseFluent, nextURLFluent, parseDefaultTime)
}

func newVidyaGaemsScraper(cacheDir string) (*sbScraper, error) {
	return newScraper(cacheDir, domain.Vidyagaems, "https://www.vidyagaems.net/sourcebans/", "",
		parseFluent, nextURLFluent, parseTrailYear)
}

func newOwlTFScraper(cacheDir string) (*sbScraper, error) {
	return newScraper(cacheDir, domain.OWL, "https://kingpandagamer.xyz/sb/", "",
		parseDefault, nextURLLast, parseDefaultTime)
}

func newZMBrasilScraper(cacheDir string) (*sbScraper, error) {
	return newScraper(cacheDir, domain.ZMBrasil, "http://bans.zmbrasil.com.br/", "",
		parseDefault, nextURLLast, parseSkialTime)
}

func newDixiGameScraper(cacheDir string) (*sbScraper, error) {
	return newScraper(cacheDir, domain.Dixigame, "https://dixigame.com/bans/", "",
		parseDefault, nextURLLast, parseDefaultTime)
}

func newScrapTFScraper(cacheDir string) (*sbScraper, error) {
	return newScraper(cacheDir, domain.ScrapTF, "https://bans.scrap.tf/", "",
		parseDefault, nextURLLast, parseDefaultTime)
}

func newWonderlandTFScraper(cacheDir string) (*sbScraper, error) {
	const siteSleepTime = time.Second * 10

	transport := newCFTransport()

	if errOpen := transport.Open(context.Background()); errOpen != nil {
		return nil, errors.Join(errOpen, errScrapeCFOpen)
	}

	scraper, errScraper := newScraperWithTransport(cacheDir,
		domain.Wonderland,
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
	scraper.nextURL = func(scraper *sbScraper, _ *goquery.Selection) string {
		scraper.curPage++

		return scraper.url(fmt.Sprintf("index.php?p=banlist&page=%d", scraper.curPage))
	}

	return scraper, nil
}

func newLazyPurpleScraper(cacheDir string) (*sbScraper, error) {
	return newScraper(cacheDir, domain.LazyPurple, "https://www.lazypurple.com/sourcebans/", "",
		parseDefault, nextURLLast, parseDefaultTime)
}

func newFirePoweredScraper(cacheDir string) (*sbScraper, error) {
	return newScraper(cacheDir, domain.FirePowered, "https://firepoweredgaming.com/sourcebanspp/", "",
		parseDefault, nextURLLast, parseSkialTime)
}

func newHarpoonScraper(cacheDir string) (*sbScraper, error) {
	return newScraper(cacheDir, domain.Harpoon, "https://bans.harpoongaming.com/", "",
		parseDefault, nextURLLast, parseDefaultTime)
}

func newPandaScraper(cacheDir string) (*sbScraper, error) {
	return newScraper(cacheDir, domain.Panda, "https://bans.panda-community.com/", "",
		parseFluent, nextURLFluent, parseDefaultTime)
}

func newNeonHeightsScraper(cacheDir string) (*sbScraper, error) {
	return newScraper(cacheDir, domain.NeonHeights, "https://neonheights.xyz/bans/", "",
		parseDefault, nextURLLast, parseSkialTime)
}

func newPancakesScraper(cacheDir string) (*sbScraper, error) {
	return newScraper(cacheDir, domain.Pancakes, "https://pancakes.tf/", "",
		parseDefault, nextURLLast, parsePancakesTime)
}

func newLOOSScraper(cacheDir string) (*sbScraper, error) {
	return newScraper(cacheDir, domain.Loos, "https://looscommunity.com/bans/", "",
		parseDefault, nextURLLast, parseDefaultTime)
}

func newPubsTFScraper(cacheDir string) (*sbScraper, error) {
	return newScraper(cacheDir, domain.PubsTF, "https://bans.pubs.tf/", "",
		parseDefault, nextURLLast, parseSkialTime)
}

func newServiliveClScraper(cacheDir string) (*sbScraper, error) {
	return newScraper(cacheDir, domain.ServiLiveCl, "https://sourcebans.servilive.cl/", "",
		parseFluent, nextURLFluent, parseDefaultTimeMonthFirst)
}

func newCutiePieScraper(cacheDir string) (*sbScraper, error) {
	return newScraper(cacheDir, domain.CutiePie, "https://bans.cutiepie.tf/", "",
		parseDefault, nextURLLast, parseDefaultTime)
}

func newSGGamingScraper(cacheDir string) (*sbScraper, error) {
	return newScraper(cacheDir, domain.SGGaming, "https://sg-gaming.net/bans/", "",
		parseDefault, nextURLLast, parseSGGamingTime)
}

func newApeModeScraper(cacheDir string) (*sbScraper, error) {
	return newScraper(cacheDir, domain.ApeMode, "https://sourcebans.apemode.tf/", "",
		parseDefault, nextURLLast, parseSkialTime)
}

func newMaxDBScraper(cacheDir string) (*sbScraper, error) {
	return newScraper(cacheDir, domain.MaxDB, "https://bans.maxdb.net/", "",
		parseDefault, nextURLLast, parseDefaultTime)
}

func newSvdosBrothersScraper(cacheDir string) (*sbScraper, error) {
	return newScraper(cacheDir, domain.SvdosBrothers, "https://bans.svdosbrothers.com/", "",
		parseFluent, nextURLFluent, parseSVDos)
}

func newElectricScraper(cacheDir string) (*sbScraper, error) {
	return newScraper(cacheDir, domain.Electric, "http://168.181.184.179/", "",
		parseFluent, nextURLFluent, parseDefaultTime)
}

func newGlobalParadiseScraper(cacheDir string) (*sbScraper, error) {
	return newScraper(cacheDir, domain.GlobalParadise, "https://bans.theglobalparadise.org/", "",
		parseDefault, nextURLLast, parseDefaultTime)
}

func newSavageServidoresScraper(cacheDir string) (*sbScraper, error) {
	return newScraper(cacheDir, domain.SavageServidores, "https://bans.savageservidores.com/", "",
		parseFluent, nextURLFluent, parseDefaultTime)
}

func newCSIServersScraper(cacheDir string) (*sbScraper, error) {
	return newScraper(cacheDir, domain.CSIServers, "https://bans.csiservers.com/", "",
		parseDefault, nextURLLast, parseDefaultTime)
}

func newLBGamingScraper(cacheDir string) (*sbScraper, error) {
	return newScraper(cacheDir, domain.LBGaming, "https://bans.lbgaming.co/", "",
		parseDefault, nextURLLast, parseSkialTime)
}

func newFluxTFScraper(cacheDir string) (*sbScraper, error) {
	return newScraper(cacheDir, domain.FluxTF, "https://bans.flux.tf/", "",
		parseDefault, nextURLLast, parseFluxTime)
}

func newDarkPyroScraper(cacheDir string) (*sbScraper, error) {
	return newScraper(cacheDir, domain.DarkPyro, "https://bans.darkpyrogaming.com/", "",
		parseDefault, nextURLLast, parseDarkPyroTime)
}

func newOpstOnlineScraper(cacheDir string) (*sbScraper, error) {
	return newScraper(cacheDir, domain.OpstOnline, "https://www.opstonline.com/bans/", "",
		parseDefault, nextURLLast, parseSkialTime)
}

func newBouncyBallScraper(cacheDir string) (*sbScraper, error) {
	return newScraper(cacheDir, domain.BouncyBall, "https://www.bouncyball.eu/bans2/", "",
		parseDefault, nextURLLast, parseSkialTime)
}

func newFurryPoundScraper(cacheDir string) (*sbScraper, error) {
	return newScraper(cacheDir, domain.FurryPound, "http://sourcebans.thefurrypound.org/", "",
		parseDefault, nextURLLast, parseFurryPoundTime)
}

func newRetroServersScraper(cacheDir string) (*sbScraper, error) {
	return newScraper(cacheDir, domain.RetroServers, "https://bans.retroservers.net/", "",
		parseDefault, nextURLLast, parseDefaultTime)
}

func newSwapShopScraper(cacheDir string) (*sbScraper, error) {
	return newScraper(cacheDir, domain.SwapShop, "http://tf2swapshop.com/sourcebans/", "",
		parseDefault, nextURLLast, parseSkialTime)
}

func newECJScraper(cacheDir string) (*sbScraper, error) {
	return newScraper(cacheDir, domain.ECJ, "https://ecj.tf/sourcebans/", "",
		parseDefault, nextURLLast, parseSkialTime)
}

func newJumpAcademyScraper(cacheDir string) (*sbScraper, error) {
	return newScraper(cacheDir, domain.JumpAcademy, "https://bans.jumpacademy.tf/", "",
		parseDefault, nextURLLast, parseDefaultTime)
}

func newTF2ROScraper(cacheDir string) (*sbScraper, error) {
	// Not enough values to page yet...
	return newScraper(cacheDir, domain.TF2Ro, "https://bans.tf2ro.com/", "",
		parseDefault, nextURLLast, parseDefaultTime)
}

func newSameTeemScraper(cacheDir string) (*sbScraper, error) {
	return newScraper(cacheDir, domain.SameTeem, "https://sameteem.com/sourcebans/", "",
		parseDefault, nextURLLast, parseDefaultTime)
}

func newPowerFPSScraper(cacheDir string) (*sbScraper, error) {
	return newScraper(cacheDir, domain.PowerFPS, "https://bans.powerfps.com/", "",
		parseDefault, nextURLLast, parseSkialTime)
}

func new7MauScraper(cacheDir string) (*sbScraper, error) {
	return newScraper(cacheDir, domain.SevenMau, "https://7-mau.com/server/", "",
		parseFluent, nextURLFluent, parseDefaultTime)
}

func newGhostCapScraper(cacheDir string) (*sbScraper, error) {
	return newScraper(cacheDir, domain.GhostCap, "https://sourcebans.ghostcap.com/", "",
		parseDefault, nextURLLast, parseDefaultTime)
}

func newSpectreScraper(cacheDir string) (*sbScraper, error) {
	return newScraper(cacheDir, domain.Spectre, "https://spectre.gg/bans/", "",
		parseDefault, nextURLLast, parseDefaultTime)
}

func newDreamFireScraper(cacheDir string) (*sbScraper, error) {
	return newScraper(cacheDir, domain.DreamFire, "https://sourcebans.dreamfire.fr/", "",
		parseDefault, nextURLLast, parseDefaultTime)
}

func newSettiScraper(cacheDir string) (*sbScraper, error) {
	return newScraper(cacheDir, domain.Setti, "https://pong.setti.info/sourcebans/", "",
		parseDefault, nextURLLast, parseDefaultTime)
}

func newGunServerScraper(cacheDir string) (*sbScraper, error) {
	return newScraper(cacheDir, domain.GunServer, "https://gunserver.ru/sourcebans/", "",
		parseDefault, nextURLFirst, parseGunServer)
}

func newHellClanScraper(cacheDir string) (*sbScraper, error) {
	return newScraper(cacheDir, domain.HellClan, "https://hellclan.co.uk/sourcebans/", "",
		parseDefault, nextURLLast, parseHellClanTime)
}

func newSneaksScraper(cacheDir string) (*sbScraper, error) {
	return newScraper(cacheDir, domain.Sneaks, "https://bans.snksrv.com/", "",
		parseDefault, nextURLLast, parseSneakTime)
}

func newNideScraper(cacheDir string) (*sbScraper, error) {
	return newScraper(cacheDir, domain.Nide, "https://bans.nide.gg/", "",
		parseFluent, nextURLFluent, parseDefaultTime)
}

func newAstraManiaScraper(cacheDir string) (*sbScraper, error) {
	return newScraper(cacheDir, domain.AstraMania, "https://astramania.ro/sban2/", "",
		parseDefault, nextURLLast, parseTrailYear)
}

func newTF2MapsScraper(cacheDir string) (*sbScraper, error) {
	return newScraper(cacheDir, domain.TF2Maps, "https://bans.tf2maps.net/", "",
		parseDefault, nextURLLast, parseDefaultTime)
}

func newPetrolTFScraper(cacheDir string) (*sbScraper, error) {
	return newScraper(cacheDir, domain.PetrolTF, "https://petrol.tf/sb/", "",
		parseDefault, nextURLLast, parseDefaultTime)
}

func newVaticanCityScraper(cacheDir string) (*sbScraper, error) {
	return newScraper(cacheDir, domain.VaticanCity, "https://www.the-vaticancity.com/sourcebans/", "",
		parseDefault, nextURLLast, parseSkialTime)
}

func newLazyNeerScraper(cacheDir string) (*sbScraper, error) {
	return newScraper(cacheDir, domain.LazyNeer, "https://www.lazyneer.com/SourceBans/", "",
		parseDefault, nextURLLast, parseSkialAltTime)
}

func newTheVilleScraper(cacheDir string) (*sbScraper, error) {
	return newScraper(cacheDir, domain.TheVille, "https://www.theville.org/sourcebans/", "",
		parseDefault, nextURLLast, parseSkialTime)
}

func newOreonScraper(cacheDir string) (*sbScraper, error) {
	return newScraper(cacheDir, domain.Oreon, "https://www.tf2-oreon.fr/sourceban/", "",
		parseDefault, nextURLLast, parseSkialTime)
}

func newTriggerHappyScraper(cacheDir string) (*sbScraper, error) {
	return newScraper(cacheDir, domain.TriggerHappy, "https://triggerhappygamers.com/sourcebans/", "",
		parseDefault, nextURLLast, parseTriggerHappyTime)
}

func newDefuseRoScraper(cacheDir string) (*sbScraper, error) {
	return newScraper(cacheDir, domain.Defusero, "https://bans.defusero.org/", "",
		parseFluent, nextURLFluent, parseDefaultTime)
}

// Has cloudflare.
// func newTawernaScraper(cacheDir string) (*sbScraper, error) {
//	return newScraper(cacheDir, Tawerna, "https://sb.tawerna.tf/", "",
//		parseDefault, nextURLLast, parseSkialTime)
// }

func newTitanScraper(cacheDir string) (*sbScraper, error) {
	return newScraper(cacheDir, domain.TitanTF, "https://bans.titan.tf/", "",
		parseDefault, nextURLLast, parseTitanTime)
}

func newDiscFFScraper(cacheDir string) (*sbScraper, error) {
	return newScraper(cacheDir, domain.DiscFF, "http://disc-ff.site.nfoservers.com/sourcebanstf2/", "",
		parseDefault, nextURLLast, parseDefaultTime)
}

// New theme
// func NewOtakuScraper( ) (*sbScraper, error){
//	return newScraper(cacheDir, Otaku, "https://bans.otaku.tf/bans", "",
//		parseDefault, nextURLLast, parseOtakuTime)
//}

func newAMSGamingScraper(cacheDir string) (*sbScraper, error) {
	return newScraper(cacheDir, domain.AMSGaming, "https://bans.amsgaming.in/", "",
		parseStar, nextURLLast, parseAMSGamingTime)
}

func newBaitedCommunityScraper(cacheDir string) (*sbScraper, error) {
	return newScraper(cacheDir, domain.BaitedCommunity, "https://bans.baitedcommunity.com/", "",
		parseStar, nextURLLast, parseBaitedTime)
}

func newCedaPugScraper(cacheDir string) (*sbScraper, error) {
	return newScraper(cacheDir, domain.CedaPug, "https://cedapug.com/sourcebans/", "",
		parseStar, nextURLLast, parseSkialTime)
}

func newGameSitesScraper(cacheDir string) (*sbScraper, error) {
	return newScraper(cacheDir, domain.GameSites, "https://banlist.gamesites.cz/tf2/", "",
		parseStar, nextURLLast, parseSkialTime)
}

func newBachuruServasScraper(cacheDir string) (*sbScraper, error) {
	return newScraper(cacheDir, domain.BachuruServas, "https://bachuruservas.lt/sb/", "",
		parseStar, nextURLLast, parseBachuruServasTime)
}

func newBierwieseScraper(cacheDir string) (*sbScraper, error) {
	return newScraper(cacheDir, domain.Bierwiese, "http://94.249.194.218/sb/", "",
		parseStar, nextURLLast, parseSkialTime)
}

func newAceKillScraper(cacheDir string) (*sbScraper, error) {
	return newScraper(cacheDir, domain.AceKill, "https://sourcebans.acekill.pl/", "",
		parseStar, nextURLLast, parseSkialTime)
}

func newMagyarhnsScraper(cacheDir string) (*sbScraper, error) {
	return newScraper(cacheDir, domain.Magyarhns, "https://magyarhns.hu/sourcebans/", "",
		parseStar, nextURLLast, parseSkialTime)
}

func newGamesTownScraper(cacheDir string) (*sbScraper, error) {
	return newScraper(cacheDir, domain.GamesTown, "https://banlist.games-town.eu/", "",
		parseStar, nextURLLast, parseTrailYear)
}

func newProGamesZetScraper(cacheDir string) (*sbScraper, error) {
	return newScraper(cacheDir, domain.ProGamesZet, "https://bans.progameszet.ru/", "",
		parseMaterial, nextURLLast, parseProGamesZetTime)
}

func newG44Scraper(cacheDir string) (*sbScraper, error) {
	return newScraper(cacheDir, domain.G44, "http://bans.allmaps.g44.rocks/", "",
		parseMaterial, nextURLLast, parseProGamesZetTime)
}

func newCuteProjectScraper(cacheDir string) (*sbScraper, error) {
	const siteSleepTime = time.Second * 4

	scraper, errScraper := newScraper(cacheDir, domain.CuteProject, "https://bans.cute-project.net/", "",
		parseMaterial, nextURLLast, parseProGamesZetTime)

	if errScraper != nil {
		return nil, errScraper
	}

	scraper.sleepTime = siteSleepTime

	return scraper, nil
}

func newPhoenixSourceScraper(cacheDir string) (*sbScraper, error) {
	return newScraper(cacheDir, domain.PhoenixSource, "https://phoenix-source.ru/sb/", "",
		parseMaterial, nextURLLast, parseProGamesZetTime)
}

func newSlavonServerScraper(cacheDir string) (*sbScraper, error) {
	return newScraper(cacheDir, domain.SlavonServer, "http://slavonserver.ru/ma/", "",
		parseMaterial, nextURLLast, parseProGamesZetTime)
}

func newGetSomeScraper(cacheDir string) (*sbScraper, error) {
	return newScraper(cacheDir, domain.GetSome, "https://bans.getsome.co.nz/", "",
		parseDefault, nextURLLast, parseSkialTime)
}

func newRushyScraper(cacheDir string) (*sbScraper, error) {
	return newScraper(cacheDir, domain.Rushy, "https://sourcebans.rushyservers.com/", "",
		parseDefault, nextURLLast, parseRushyTime)
}

func newMoevsMachineScraper(cacheDir string) (*sbScraper, error) {
	return newScraper(cacheDir, domain.MoeVsMachine, "https://moevsmachine.tf/bans/", "",
		parseDefault, nextURLLast, parseDefaultTime)
}

func newPRWHScraper(cacheDir string) (*sbScraper, error) {
	return newScraper(cacheDir, domain.Prwh, "https://sourcebans.prwh.de/", "",
		parseDefault, nextURLLast, parsePRWHTime)
}

func newVortexScraper(cacheDir string) (*sbScraper, error) {
	return newScraper(cacheDir, domain.Vortex, "http://vortex.oyunboss.net/sourcebans/", "",
		parseStar, nextURLLast, parseSkialTime)
}

func newCasualFunScraper(cacheDir string) (*sbScraper, error) {
	return newScraper(cacheDir, domain.CasualFun, "https://tf2-casual-fun.de/sourcebans/", "",
		parseDefault, nextURLLast, parsePRWHTime)
}

func newRandomTF2Scraper(cacheDir string) (*sbScraper, error) {
	return newScraper(cacheDir, domain.RandomTF2, "https://bans.randomtf2.com/", "",
		parseDefault, nextURLLast, parseSkialTime)
}

func newPlayesROScraper(cacheDir string) (*sbScraper, error) {
	return newScraper(cacheDir, domain.PlayersRo, "https://www.playes.ro/csgobans/", "",
		parseStar, nextURLLast, parseSkialTime)
}

func newEOTLGamingScraper(cacheDir string) (*sbScraper, error) {
	return newScraper(cacheDir, domain.EOTLGaming, "https://tf2.endofthelinegaming.com/sourcebans/", "",
		parseDefault, nextURLLast, parseDefaultTime)
}

func newBioCraftingScraper(cacheDir string) (*sbScraper, error) {
	return newScraper(cacheDir, domain.BioCrafting, "https://sourcebans.biocrafting.net/", "",
		parseDefault, nextURLLast, parseSkialTime)
}

func newBigBangGamersScraper(cacheDir string) (*sbScraper, error) {
	return newScraper(cacheDir, domain.BigBangGamers, "http://208.71.172.9/", "",
		parseDefault, nextURLLast, parseSkialTime)
}

func newEpicZoneScraper(cacheDir string) (*sbScraper, error) {
	return newScraper(cacheDir, domain.EpicZone, "https://sourcebans.epiczone.sk/", "",
		parseStar, nextURLLast, parseGunServer)
}

func newZubatScraper(cacheDir string) (*sbScraper, error) {
	return newScraper(cacheDir, domain.Zubat, "https://sb.zubat.ru/", "",
		parseStar, nextURLLast, parseSkialTime)
}

func newLunarioScraper(cacheDir string) (*sbScraper, error) {
	return newScraper(cacheDir, domain.Lunario, "https://sb.lunario.ro/", "",
		parseStar, nextURLLast, parseSkialTime)
}
