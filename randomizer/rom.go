package randomizer

import (
	"bytes"
	"crypto/sha1"
	"fmt"
	"regexp"
	"strings"
	"container/list"
)

const bankSize = 0x4000

// a fully-specified memory address. "offset" isn't true offset from the start
// of the bank (except for bank 0); it's bus address.
type address struct {
	bank   uint8
	offset uint16
}

// fullOffset returns the actual offset of the address in the ROM, based on
// bank number and relative address.
func (a *address) fullOffset() int {
	var bankOffset int
	if a.bank >= 2 {
		bankOffset = bankSize * (int(a.bank) - 1)
	}
	return bankOffset + int(a.offset)
}

func romIsAges(b []byte) bool {
	return string(b[0x134:0x13f]) == "ZELDA NAYRU"
}

func romIsSeasons(b []byte) bool {
	return string(b[0x134:0x13d]) == "ZELDA DIN"
}

func romIsJp(b []byte) bool {
	return b[0x014a] == 0
}

func romIsVanilla(b []byte) bool {
	knownSum := ternary(romIsSeasons(b),
		"\xba\x12\x68\x29\x0f\xb2\xb1\xb7\x05\x05\xd2\xd7\xb5\x82\x5f\xc8\xa4"+
			"\x81\x6a\x4b",
		"\x88\x03\x74\xfb\x97\x8b\x18\xaf\x4a\xa5\x29\xe2\xe3\x2f\x7f\xfb\x4d"+
			"\x7d\xd2\xf4").(string)
	sum := sha1.Sum(b)
	return string(sum[:]) == knownSum
}

// returns a 16-bit checksum of the rom data, for placing in the rom header.
// this is calculated by summing the non-global-checksum bytes in the rom.
// not to be confused with the header checksum, which is the byte before.
func makeRomChecksum(data []byte) [2]byte {
	var sum uint16
	for _, c := range data[:0x14e] {
		sum += uint16(c)
	}
	for _, c := range data[0x150:] {
		sum += uint16(c)
	}
	return [2]byte{byte(sum >> 8), byte(sum)}
}

type romState struct {
	game         int
	player       int
	data         []byte // actual contents of the file
	treasures    map[string]*treasure
	itemSlots    map[string]*itemSlot
	codeMutables map[string]*mutableRange
	bankEnds     []uint16 // bus offset of free space in each bank
	assembler    *assembler
	includes     []string // filenames
}

func newRomState(data []byte, game int) *romState {
	rom := &romState{
		game:      game,
		player:    0,
		data:      data,
		treasures: loadTreasures(data, game),
		includes:  []string{},
	}
	rom.itemSlots = rom.loadSlots()
	rom.initBanks()
	return rom
}

// getChecks converts a route info into a map of checks.
func getChecks(usedItems, usedSlots *list.List) map[string]string {
	checks := make(map[string]string)

	ei, es := usedItems.Front(), usedSlots.Front()
	for ei != nil {
		checks[es.Value.(string)] = ei.Value.(string)
		ei, es = ei.Next(), es.Next()
	}

	return checks
}

// mutates the rom data in-place based on the given route. this doesn't write
// the file.
func (rom *romState) setData(ri *routeInfo) ([]byte, error) {
    rom.setTreewarp(ri.warpToStart)

    // place selected treasures in slots
    checks := getChecks(ri.usedItems, ri.usedSlots)
    for slot, item := range checks {
        rom.itemSlots[slot].treasure = rom.treasures[item]
    }

    // set season data
    if rom.game == GAME_SEASONS {
        for area, id := range ri.seasons {
            rom.setSeason(inflictCamelCase(area+"Season"), id)
        }
    }

    rom.setHeartBeepInterval(ri.heartBeepInterval)
    rom.setRequiredEssences(ri.requiredEssences)
    rom.setAnimal(ri.companion)
    rom.setArchipelagoSlotName(ri.archipelagoSlotName)

    warps := make(map[string]string)

    if ri.game == GAME_SEASONS {
        rom.setFoolsOreDamage(ri.foolsOreDamage)
        rom.setLostWoodsPedestalSequence(ri.pedestalSequence)

        arePortalsShuffled := (ri.portals != nil && len(ri.portals) > 0)
        if arePortalsShuffled {
            for k, v := range ri.portals {
                holodrumV, _ := reverseLookup(SUBROSIAN_PORTAL_NAMES, v)
                warps[fmt.Sprintf("%s portal", k)] = fmt.Sprintf("%s portal", holodrumV)
            }
        }
    }

    areDungeonsShuffled := (ri.entrances != nil && len(ri.entrances) > 0)
    if areDungeonsShuffled {
        for k, v := range ri.entrances {
            warps[k] = v
        }
    }

    // do it! (but don't write anything)
    return rom.mutate(warps, ri.archipelagoSlotName, areDungeonsShuffled)
}

// changes the contents of loaded ROM bytes in place. returns a checksum of the
// result or an error.
func (rom *romState) mutate(warpMap map[string]string, archipelagoSlotName string, areDungeonsShuffled bool) ([]byte, error) {
	// need to set this *before* treasure map data
	if len(warpMap) != 0 {
		rom.setWarps(warpMap, areDungeonsShuffled)
	}

	if rom.game == GAME_SEASONS {
		northHoronSeason :=
			rom.codeMutables["northHoronSeason"].new[0]
		rom.codeMutables["initialSeason"].new =
			[]byte{0x2d, northHoronSeason}
		westernCoastSeason :=
			rom.codeMutables["westernCoastSeason"].new[0]
		rom.codeMutables["seasonAfterPirateCutscene"].new =
			[]byte{westernCoastSeason}

		rom.setTreasureMapData()

		// explicitly set these addresses and IDs after their functions
		codeAddr := rom.codeMutables["setStarOreIds"].addr
		rom.itemSlots["subrosia seaside"].idAddrs[0].offset = codeAddr.offset + 2
		rom.itemSlots["subrosia seaside"].subidAddrs[0].offset = codeAddr.offset + 5
		codeAddr = rom.codeMutables["setHardOreIds"].addr
		rom.itemSlots["great furnace"].idAddrs[0].offset = codeAddr.offset + 2
		rom.itemSlots["great furnace"].subidAddrs[0].offset = codeAddr.offset + 5
		codeAddr = rom.codeMutables["script_diverGiveItem"].addr
		rom.itemSlots["master diver's reward"].idAddrs[0].offset = codeAddr.offset + 1
		rom.itemSlots["master diver's reward"].subidAddrs[0].offset = codeAddr.offset + 2
		codeAddr = rom.codeMutables["createMtCuccoItem"].addr
		rom.itemSlots["mt. cucco, platform cave"].idAddrs[0].offset = codeAddr.offset + 2
		rom.itemSlots["mt. cucco, platform cave"].subidAddrs[0].offset = codeAddr.offset + 1

		wildsDigSpotSlot := rom.itemSlots["subrosian wilds digging spot"]
		wildsDigSpotSlot.idAddrs[0].offset = rom.codeMutables["subrosianWildsDiggingSpotItem"].addr.offset
		wildsDigSpotSlot.subidAddrs[0].offset = rom.codeMutables["subrosianWildsDiggingSpotItemSubid"].addr.offset

		spoolDigSpotSlot := rom.itemSlots["spool swamp digging spot"]
		spoolDigSpotSlot.idAddrs[0].offset = rom.codeMutables["vasuSignDiggingSpotItem"].addr.offset
		spoolDigSpotSlot.subidAddrs[0].offset = rom.codeMutables["vasuSignDiggingSpotItemSubid"].addr.offset

		// prepare static items addresses
		codeAddr = rom.codeMutables["staticItemsReplacementsTable"].addr
		var i uint16 = 0
		for _, key := range STATIC_SLOTS {
			rom.itemSlots[key].idAddrs = []address{{codeAddr.bank, codeAddr.offset + (i*3) + 1}}
			rom.itemSlots[key].subidAddrs = []address{{codeAddr.bank, codeAddr.offset + (i*3) + 2}}
			i++
		}
	} else {
		// explicitly set these addresses and IDs after their functions
		mut := rom.codeMutables["script_soldierGiveItem"]
		slot := rom.itemSlots["deku forest soldier"]
		slot.idAddrs[0].offset = mut.addr.offset + 13
		slot.subidAddrs[0].offset = mut.addr.offset + 14
		mut = rom.codeMutables["script_giveTargetCartsSecondPrize"]
		codeAddr := mut.addr
		rom.itemSlots["target carts 2"].idAddrs[1].offset = codeAddr.offset + 1
		rom.itemSlots["target carts 2"].subidAddrs[1].offset = codeAddr.offset + 2
	}

	rom.setBossItemAddrs()
	rom.setSeedData()
	rom.setRoomTreasureData()
	rom.setFileSelectText(archipelagoSlotName)
	rom.attachText()
	rom.codeMutables["multiPlayerNumber"].new[0] = byte(rom.player)

	// regenerate collect mode table to accommodate changes based on contents.
	rom.codeMutables["collectPropertiesTable"].new =
		[]byte(makeCollectPropertiesTable(rom.game, rom.player, rom.itemSlots))

	mutables := rom.getAllMutables()
	for _, k := range orderedKeys(mutables) {
		mutables[k].mutate(rom.data)
	}

	// explicitly set these items after their functions are written
	rom.writeBossItems()
	if rom.game == GAME_SEASONS {
		rom.itemSlots["subrosia seaside"].mutate(rom.data)
		rom.itemSlots["great furnace"].mutate(rom.data)
		rom.itemSlots["master diver's reward"].mutate(rom.data)
		rom.itemSlots["subrosian wilds digging spot"].mutate(rom.data)
		rom.itemSlots["spool swamp digging spot"].mutate(rom.data)

		codeAddr := rom.codeMutables["vasuGiveItem"].addr
		slotToMutate := rom.itemSlots["vasu's gift"]
		slotToMutate.idAddrs = []address{{codeAddr.bank, codeAddr.offset + 2}}
		slotToMutate.subidAddrs = []address{{codeAddr.bank, codeAddr.offset + 1}}
		slotToMutate.mutate(rom.data)

		// annoying special case to prevent text on key drop
		mut := rom.itemSlots["d7 armos puzzle"]
		if mut.treasure.id == rom.treasures["Small Key (Explorer's Crypt)"].id {
			rom.data[mut.subidAddrs[0].fullOffset()] = 0x01
		}

		// mutate static items
		for _, key := range STATIC_SLOTS {
			rom.itemSlots[key].mutate(rom.data)
		}
		
	} else {
		rom.itemSlots["nayru's house"].mutate(rom.data)
		rom.itemSlots["deku forest soldier"].mutate(rom.data)
		rom.itemSlots["target carts 2"].mutate(rom.data)
		rom.itemSlots["hidden tokay cave"].mutate(rom.data)

		// other special case to prevent text on key drop
		mut := rom.itemSlots["d8 stalfos"]
		if mut.treasure.id == rom.treasures["Small Key (Sword & Shield Maze)"].id {
			rom.data[mut.subidAddrs[0].fullOffset()] = 0x00
		}
	}

	rom.setCompassData()
	rom.setLinkedData()

	// do this last; includes have precendence over everything else
	rom.addIncludes()

	sum := makeRomChecksum(rom.data)
	rom.data[0x14e] = sum[0]
	rom.data[0x14f] = sum[1]

	outSum := sha1.Sum(rom.data)
	return outSum[:], nil
}

// checks all the package's data against the ROM to see if it matches. It
// returns a slice of errors describing each mismatch.
func (rom *romState) verify() []error {
	errors := make([]error, 0)
	for k, m := range rom.getAllMutables() {
		// ignore special cases that would error even when correct
		switch k {
		// seasons shop items
		case "zero shop text", "Member's Card", "Treasure Map",
			"Rare Peach Stone", "Ribbon":
		// flutes
		case "Ricky's Flute", "Dimitri's Flute", "Moosh's Flute":
		// seasons linked chests
		case "spool swamp cave", "woods of winter, 2nd cave",
			"dry eyeglass lake, west cave":
		// seasons misc.
		case "Bracelet", "temple of seasons", "Fool's Ore", "blaino prize",
			"mt. cucco, platform cave", "diving spot outside D4", "vasu's gift",
			"goron's gift", "dr. left reward", "malon trade", "mrs. ruul trade",
			"subrosian chef trade", "biggoron trade", "ingo trade", "old man trade",
			"talon trade", "syrup trade", "tick tock trade", "guru-guru trade":
		// ages progressive w/ different item IDs
		case "nayru's house", "tokkey's composition", "rescue nayru",
			"d6 present vire chest":
		// ages misc.
		case "south shore dirt", "target carts 2", "sea of storms past",
			"starting chest", "graveyard poe":
		// additional misc.
		case "Archipelago Item":
		default:
			if err := m.check(rom.data); err != nil {
				errors = append(errors, fmt.Errorf("%s: %v", k, err))
			}
		}
	}

	if len(errors) > 0 {
		return errors
	}
	return nil
}

// set the initial satchel and slingshot seeds (and selections) based on what
// grows on the horon village tree, and set the map icon for each tree to match
// the seed type.
func (rom *romState) setSeedData() {
	treeName := sora(rom.game, "horon village tree", "south lynna tree").(string)
	seedType := rom.itemSlots[treeName].treasure.id

	if rom.game == GAME_SEASONS {
		// satchel/slingshot starting seeds
		rom.codeMutables["satchelInitialSeeds"].new[0] = 0x20 + seedType
		rom.codeMutables["editGainLoseItemsTables"].new[1] = 0x20 + seedType

		for _, name := range []string{
			"satchelInitialSelection", "slingshotInitialSelection"} {
			rom.codeMutables[name].new[1] = seedType
		}

		for _, names := range [][]string{
			{"horon village tree", "horonVillageTreeMapIcon"},
			{"north horon tree", "northHoronTreeMapIcon"},
			{"woods of winter tree", "woodsOfWinterTreeMapIcon"},
			{"spool swamp tree", "spoolSwampTreeMapIcon"},
			{"sunken city tree", "sunkenCityTreeMapIcon"},
			{"tarm ruins tree", "tarmRuinsTreeMapIcon"},
		} {
			id := rom.itemSlots[names[0]].treasure.id
			rom.codeMutables[names[1]].new[0] = 0x15 + id
		}
	} else {
		// set high nybbles (seed types) of seed tree interactions
		setTreeNybble(rom.codeMutables["symmetryCityTreeSubId"],
			rom.itemSlots["symmetry city tree"])
		setTreeNybble(rom.codeMutables["southLynnaPresentTreeSubId"],
			rom.itemSlots["south lynna tree"])
		setTreeNybble(rom.codeMutables["crescentIslandTreeSubId"],
			rom.itemSlots["crescent island tree"])
		setTreeNybble(rom.codeMutables["zoraVillagePresentTreeSubId"],
			rom.itemSlots["zora village tree"])
		setTreeNybble(rom.codeMutables["rollingRidgeWestTreeSubId"],
			rom.itemSlots["rolling ridge west tree"])
		setTreeNybble(rom.codeMutables["ambisPalaceTreeSubId"],
			rom.itemSlots["ambi's palace tree"])
		setTreeNybble(rom.codeMutables["rollingRidgeEastTreeSubId"],
			rom.itemSlots["rolling ridge east tree"])
		setTreeNybble(rom.codeMutables["southLynnaPastTreeSubId"],
			rom.itemSlots["south lynna tree"])
		setTreeNybble(rom.codeMutables["dekuForestTreeSubId"],
			rom.itemSlots["deku forest tree"])
		setTreeNybble(rom.codeMutables["zoraVillagePastTreeSubId"],
			rom.itemSlots["zora village tree"])

		// satchel and shooter come with south lynna tree seeds
		rom.codeMutables["satchelInitialSeeds"].new[0] = 0x20 + seedType
		rom.codeMutables["seedShooterGiveSeeds"].new[6] = 0x20 + seedType
		for _, name := range []string{"satchelInitialSelection",
			"shooterInitialSelection"} {
			rom.codeMutables[name].new[1] = seedType
		}

		// set map icons
		for _, name := range []string{"crescent island tree",
			"symmetry city tree", "south lynna tree", "zora village tree",
			"rolling ridge west tree", "ambi's palace tree",
			"rolling ridge east tree", "deku forest tree"} {
			codeName := inflictCamelCase(name) + "MapIcon"
			if name == "south lynna tree" || name == "zora village tree" {
				for _, n := range []string{"1", "2"} {
					rom.codeMutables[codeName+n].new[0] =
						0x15 + rom.itemSlots[name].treasure.id
				}
			} else {
				rom.codeMutables[codeName].new[0] =
					0x15 + rom.itemSlots[name].treasure.id
			}
		}
	}
}

// converts e.g. "hello world" to "helloWorld". disgusting tbh
func inflictCamelCase(s string) string {
	return fmt.Sprintf("%c%s", s[0], strings.ReplaceAll(
		strings.Title(strings.ReplaceAll(s, "'", "")), " ", "")[1:])
}

// fill table. initial table is blank, since it's created before items are
// placed.
func (rom *romState) setRoomTreasureData() {
	rom.codeMutables["roomTreasures"].new =
		[]byte(makeRoomTreasureTable(rom.game, rom.itemSlots))
	if rom.game == GAME_SEASONS {
		t := rom.itemSlots["d7 zol button"].treasure
		rom.codeMutables["aboveD7ZolButtonId"].new = []byte{t.id}
		rom.codeMutables["aboveD7ZolButtonSubid"].new = []byte{t.subid}
	}
}

// sets the high nybble (seed type) of a seed tree interaction in ages.
func setTreeNybble(subid *mutableRange, slot *itemSlot) {
	subid.new[0] = (subid.new[0] & 0x0f) | (slot.treasure.id << 4)
}

// set the locations of the sparkles for the jewels on the treasure map.
func (rom *romState) setTreasureMapData() {
	for _, name := range []string{"Round", "Pyramid", "Square", "X-Shaped"} {
		label := "jewelCoords" + strings.ReplaceAll(name, "-", "")
		rom.codeMutables[label].new[0] = 0x63 // default to tarm gate
		for _, slot := range rom.lookupAllItemSlots(name + " Jewel") {
			if int(slot.player) == 0 || int(slot.player) == rom.player {
				rom.codeMutables[label].new[0] = slot.mapTile
			}
		}
	}
}

// set dungeon properties so that the compass beeps in the rooms actually
// containing small keys and boss keys.
func (rom *romState) setCompassData() {
	prefixes := sora(rom.game,
		[]string{"d0", "d1", "d2", "d3", "d4", "d5", "d6", "d7", "d8"},
		[]string{"d0", "d1", "d2", "d3", "d4", "d5", "d6 present", "d6 past",
			"d7", "d8"}).([]string)

	// clear key flags
	for _, prefix := range prefixes {
		for name, slot := range rom.itemSlots {
			if strings.HasPrefix(name, prefix+" ") {
				offset := getDungeonPropertiesAddr(
					rom.game, slot.group, slot.room).fullOffset()
				rom.data[offset] = rom.data[offset] & 0xed // reset bit 4
			}
		}
	}

	// set key flags
	for _, prefix := range prefixes {
		slots := rom.lookupAllItemSlots(fmt.Sprintf("%s small key", prefix))

		// boss keys can be absent in plando, so handle the nil case
		switch prefix {
		case "d0", "d6 present":
			break
		case "d6 past":
			if slot := rom.lookupItemSlot("Boss Key (Ancient Ruins)"); slot != nil {
				slots = append(slots, slot)
			}
		default:
			keyName := fmt.Sprintf("%s boss key", prefix)
			if slot := rom.lookupItemSlot(keyName); slot != nil {
				slots = append(slots, slot)
			}
		}

		for _, slot := range slots {
			offset := getDungeonPropertiesAddr(
				rom.game, slot.group, slot.room).fullOffset()
			rom.data[offset] = (rom.data[offset] & 0xbf) | 0x10 // set bit 4, reset bit 6
		}
	}
}

// returns the slot where the named item was placed. this only works for unique
// items, of course.
func (rom *romState) lookupItemSlot(itemName string) *itemSlot {
	if slots := rom.lookupAllItemSlots(itemName); len(slots) > 0 {
		return slots[0]
	}
	return nil
}

// returns all slots where the named item was placed.
func (rom *romState) lookupAllItemSlots(itemName string) []*itemSlot {
	t := rom.treasures[itemName]
	slots := make([]*itemSlot, 0)
	for _, slot := range rom.itemSlots {
		if slot.treasure == t {
			slots = append(slots, slot)
		}
	}
	return slots
}

// get the location of the dungeon properties byte for a specific room.
func getDungeonPropertiesAddr(game int, group, room byte) *address {
	offset := uint16(room)
	offset += uint16(sora(game, 0x4d41, 0x4dce).(int))
	if group%2 != 0 {
		offset += 0x100
	}
	return &address{0x01, offset}
}

func (rom *romState) setBossItemAddrs() {
	table := rom.codeMutables["bossItemTable"]
	for i := uint16(1); i <= 8; i++ {
		slot := rom.itemSlots[fmt.Sprintf("d%d boss", i)]
		slot.idAddrs[0].offset = table.addr.offset + i*2
		slot.subidAddrs[0].offset = table.addr.offset + i*2 + 1
	}
}

func (rom *romState) writeBossItems() {
	for i := 1; i <= 8; i++ {
		rom.itemSlots[fmt.Sprintf("d%d boss", i)].mutate(rom.data)
	}
}

// set data to make linked playthroughs isomorphic to unlinked ones.
func (rom *romState) setLinkedData() {
	if rom.game == GAME_SEASONS {
		// set linked starting / hero's cave terrace items based on which items
		// in unlinked hero's cave aren't keys. order matters.
		var tStart, tCave *treasure
		if rom.itemSlots["d0 key chest"].treasure.id == 0x30 {
			tStart = rom.itemSlots["d0 sword chest"].treasure
			tCave = rom.itemSlots["d0 rupee chest"].treasure
		} else {
			tStart = rom.itemSlots["d0 key chest"].treasure
			tCave = rom.itemSlots["d0 sword chest"].treasure
		}

		// give this item at start
		linkedStartItem := &itemSlot{
			idAddrs:    []address{{0x0a, 0x7ffd}},
			subidAddrs: []address{{0x0a, 0x7ffe}},
			treasure:   tStart,
		}
		linkedStartItem.mutate(rom.data)

		// create slot for linked hero's cave terrace
		linkedChest := &itemSlot{
			treasure:    rom.treasures["Rupees (20)"],
			idAddrs:     []address{{0x15, 0x50e2}},
			subidAddrs:  []address{{0x15, 0x50e3}},
			group:       0x05,
			room:        0x2c,
			collectMode: collectModes["chest"],
			mapTile:     0xd4,
		}
		linkedChest.treasure = tCave
		linkedChest.mutate(rom.data)
	}
}

// -- dungeon entrance / subrosia portal connections --

type warpData struct {
	// loaded from yaml
	Entry, Exit uint16
	MapTile     byte

	// set after loading
	bank, vanillaMapTile         byte
	len, entryOffset, exitOffset int

	vanillaEntryData, vanillaExitData []byte // read from rom
}

func (rom *romState) setWarps(warpMap map[string]string, dungeons bool) {
	// load yaml data
	wd := make(map[string](map[string]*warpData))
	if err := ReadEmbeddedYaml("romdata/warps.yaml", wd); err != nil {
		panic(err)
	}
	warps := sora(rom.game, wd["seasons"], wd["ages"]).(map[string]*warpData)

	// read vanilla data
	for name, warp := range warps {
		if strings.HasSuffix(name, "essence") {
			warp.len = 4
			warp.bank = byte(sora(rom.game, 0x09, 0x0a).(int))
		} else {
			warp.bank, warp.len = 0x04, 2
		}
		warp.entryOffset = (&address{warp.bank, warp.Entry}).fullOffset()
		warp.vanillaEntryData = make([]byte, warp.len)
		copy(warp.vanillaEntryData,
			rom.data[warp.entryOffset:warp.entryOffset+warp.len])
		warp.exitOffset = (&address{warp.bank, warp.Exit}).fullOffset()
		warp.vanillaExitData = make([]byte, warp.len)
		copy(warp.vanillaExitData,
			rom.data[warp.exitOffset:warp.exitOffset+warp.len])

		warp.vanillaMapTile = warp.MapTile
	}

	// ages needs essence warp data to d6 present entrance, even though it
	// doesn't exist in vanilla.
	if rom.game == GAME_AGES {
		warps["d6 present essence"] = &warpData{
			vanillaExitData: []byte{0x81, 0x0e, 0x16, 0x01},
		}
	}

	// set randomized data
	for srcName, destName := range warpMap {
		src, dest := warps[srcName], warps[destName]
		for i := 0; i < src.len; i++ {
			rom.data[src.entryOffset+i] = dest.vanillaEntryData[i]
			rom.data[dest.exitOffset+i] = src.vanillaExitData[i]
		}
		dest.MapTile = src.vanillaMapTile

		destEssence := warps[destName+" essence"]
		if destEssence != nil && destEssence.exitOffset != 0 {
			srcEssence := warps[srcName+" essence"]
			for i := 0; i < destEssence.len; i++ {
				rom.data[destEssence.exitOffset+i] = srcEssence.vanillaExitData[i]
			}
		}
	}

	if rom.game == GAME_SEASONS {
		// set treasure map data. because of d8, portals go first, then dungeon
		// entrances.
		dungeonNameRegexp := regexp.MustCompile(`^d[1-8]$`)
		conditions := [](func(string) bool){
			dungeonNameRegexp.MatchString,
			func(s string) bool { return strings.HasSuffix(s, "portal") },
		}
		for _, cond := range conditions {
			changeTreasureMapTiles(rom.itemSlots, func(c chan byteChange) {
				for name, warp := range warps {
					if cond(name) {
						c <- byteChange{warp.vanillaMapTile, warp.MapTile}
					}
				}
				close(c)
			})
		}

		if dungeons {
			// remove alternate d2 entrances and connect d2 stairs exits
			// directly to each other
			src, dest := warps["d2 alt left"], warps["d2 alt right"]
			rom.data[src.exitOffset] = dest.vanillaEntryData[0]
			rom.data[src.exitOffset+1] = dest.vanillaEntryData[1]
			rom.data[dest.exitOffset] = src.vanillaEntryData[0]
			rom.data[dest.exitOffset+1] = src.vanillaEntryData[1]

			// also enable removal of the stair tiles
			mut := rom.codeMutables["d2AltEntranceTileSubs"]
			mut.new[0], mut.new[5] = 0x00, 0x00
		}
	}
}

type byteChange struct {
	old, new byte
}

// process a set of treasure map tile changes in a way that ensures each tile
// is substituted only once (per call to this function).
func changeTreasureMapTiles(slots map[string]*itemSlot,
	generate func(chan byteChange)) {
	pendingTiles := make(map[*itemSlot]byte)
	c := make(chan byteChange)
	go generate(c)

	for change := range c {
		for _, slot := range slots {
			// diving spot outside d4 would be mistaken for a d4 check
			if slot.mapTile == change.old &&
				slot != slots["diving spot outside D4"] {
				pendingTiles[slot] = change.new
			}
		}
	}

	for slot, tile := range pendingTiles {
		slot.mapTile = tile
	}
}

// set the string to display on the file select screen.
func (rom *romState) setFileSelectText(row2 string) {
	// construct tiles from strings
	fileSelectRow1 := stringToTiles(strings.ToUpper(fmt.Sprintf("archipelago %s", VERSION)))
	fileSelectRow2 := stringToTiles(strings.ToUpper(strings.ReplaceAll(row2, "-", " ")))

	tiles := rom.codeMutables["dma_FileSelectStringTiles"]
	buf := new(bytes.Buffer)
	buf.Write(tiles.new[:2])
	buf.Write(fileSelectRow1)

	padding := 16 - len(fileSelectRow2) // bias toward right padding
	buf.Write(tiles.new[2+len(fileSelectRow1) : 0x22+padding/2])
	buf.Write(fileSelectRow2)
	buf.Write(tiles.new[0x22+len(fileSelectRow2)+padding/2:])

	tiles.new = buf.Bytes()
}

// returns a conversion of the string to file select screen tile indexes, using
// the custom font.
func stringToTiles(s string) []byte {
	b := make([]byte, len(s))
	for i, c := range []byte(s) {
		b[i] = func() byte {
			switch {
			case c >= '0' && c <= '9':
				return c - 0x20
			case c >= 'A' && c <= 'Z':
				return c + 0xa1
			case c == ' ':
				return '\xfc'
			case c == '+':
				return '\xfd'
			case c == '-':
				return '\xfe'
			case c == '.':
				return '\xff'
			default:
				return '\xfc' // leave other characters blank
			}
		}()
	}
	return b
}
