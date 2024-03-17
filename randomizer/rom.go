package randomizer

import (
	"bytes"
	"crypto/sha1"
	"fmt"
	"regexp"
	"strings"
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
}

func newRomState(data []byte, game int) *romState {
	rom := &romState{
		game:      game,
		player:    0,
		data:      data,
		treasures: loadTreasures(data, game),
	}

	asm, err := newAssembler()
	if err != nil {
		panic(err)
	}
	rom.assembler = asm

	rom.codeMutables = make(map[string]*mutableRange)
	rom.bankEnds = loadBankEnds(gameNames[game])

	return rom
}

// set up all the pre-randomization asm changes, and track the state so that
// the randomization changes can be applied later.
func (rom *romState) initBanks(ri *routeInfo) {
	// do this before loading asm files, since the sizes of the tables vary
	// with the number of checks.
	rom.replaceRaw(address{0x06, 0}, "collectPropertiesTable", makeCollectPropertiesTable(rom.game, rom.player, rom.itemSlots))

	numOwlIds := sora(rom.game, 0x1e, 0x14).(int)
	rom.replaceRaw(address{0x3f, 0}, "owlTextOffsets", string(make([]byte, numOwlIds*2)))

	rom.replaceRaw(address{0x0a, 0}, "newSamasaCombination", makeSamasaCombinationTable(ri.samasaGateSequence))
}

// mutates the rom data in-place based on the given route. this doesn't write
// the file.
func (rom *romState) setData(ri *routeInfo) {
	// Apply location contents on slots
	for slotName, itemName := range ri.locationContents {
		slot := rom.itemSlots[slotName]
		item := rom.treasures[itemName]
		slot.treasure = item
	}

	// Create ASM defines for slot contents
	for slotName, slot := range rom.itemSlots {
		item := slot.treasure
		defineBaseName := "slot." + inflictCamelCase(slotName)
		rom.assembler.defineByte(defineBaseName+".id", item.id)
		rom.assembler.defineByte(defineBaseName+".subid", item.subid)
		rom.assembler.defineWord(defineBaseName+".full", uint16((uint16(item.id)<<8)|uint16(item.subid)))
		rom.assembler.defineWord(defineBaseName+".reverse", uint16((uint16(item.subid)<<8)|uint16(item.id)))
	}

	// Create ASM defines for default seasons
	for region, season := range ri.seasons {
		if season >= 4 {
			season = 0xff
		}
		rom.assembler.defineByte("defaultSeason."+inflictCamelCase(region), season)
	}

	// Create ASM defines for various options
	rom.assembler.defineByte("option.treehouseOldManRequirement", byte(ri.treehouseOldManRequirement))
	rom.assembler.defineByte("option.treehouseOldManRequirementTextDigit", byte(0x30+ri.treehouseOldManRequirement))

	rom.assembler.defineByte("option.goldenBeastsRequirement", byte(ri.goldenBeastsRequirement))
	rom.assembler.defineByte("option.goldenBeastsRequirementTextDigit", byte(0x30+ri.goldenBeastsRequirement))

	rom.assembler.defineByte("option.openAdvanceShop", byte(ri.treehouseOldManRequirement))

	rom.assembler.defineByte("option.foolsOreDamage", byte(ri.foolsOreDamage*-1))

	rom.assembler.defineByte("option.tarmGateRequiredJewels", byte(ri.tarmGateRequiredJewels))
	rom.assembler.defineByte("option.tarmGateRequiredJewelsTextDigit", byte(0x30+ri.tarmGateRequiredJewels))

	// Determines natzu landscape: 0b for ricky, 0c for dimitri, 0d for moosh.
	rom.assembler.defineByte("option.animalCompanion", byte(ri.companion+0x0a))

	// If enabled, put real Subrosia map group (0x01), otherwise put a fake map group that will never trigger tile changes (0xfe)
	rom.assembler.defineByte("option.revealGoldenOreTiles", ternary(ri.revealGoldenOreTiles, uint8(0x01), uint8(0xfe)).(uint8))

	// Sign guy requirement being a multi-digit requirement, it requires quite some decomposition work
	rom.assembler.defineByte("option.signGuyRequirement", byte(ri.signGuyRequirement))
	rom.defineSignGuyTextConstants(ri)
}

func (rom *romState) defineSignGuyTextConstants(ri *routeInfo) {
	signDigit3 := ri.signGuyRequirement % 10
	signDigit2 := (ri.signGuyRequirement / 10) % 10
	signDigit1 := int(ri.signGuyRequirement / 100)

	textDigit1 := '0' + signDigit1
	if signDigit1 == 0 {
		textDigit1 = ' '
	}
	textDigit2 := '0' + signDigit2
	if signDigit1 == 0 && signDigit2 == 0 {
		textDigit2 = ' '
	}
	textDigit3 := '0' + signDigit3

	rom.assembler.defineByte("option.signGuyRequirement.digit1", byte(textDigit1))
	rom.assembler.defineByte("option.signGuyRequirement.digit2", byte(textDigit2))
	rom.assembler.defineByte("option.signGuyRequirement.digit3", byte(textDigit3))
}

// changes the contents of loaded ROM bytes in place. returns a checksum of the
// result or an error.
func (rom *romState) mutate(ri *routeInfo) ([]byte, error) {
	var err error

	rom.setHeartBeepInterval(ri.heartBeepInterval)
	rom.setRequiredEssences(ri.requiredEssences)

	rom.setArchipelagoSlotName(ri.archipelagoSlotName)
	rom.setOldManRupeeValues(ri.oldManRupeeValues)
	rom.setCharacterSprite(ri.characterSprite, ri.characterPalette)

	warps := make(map[string]string)

	if ri.game == GAME_SEASONS {
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

	// need to set this *before* treasure map data
	if len(warps) != 0 {
		rom.setWarps(warps, areDungeonsShuffled)
	}

	if rom.game == GAME_SEASONS {
		rom.setTreasureMapData()

		rom.codeMutables["newShowSamasaCombination"].new, err = makeSamasaGateSequenceScript(ri.samasaGateSequence)
		if err != nil {
			return nil, err
		}
		rom.codeMutables["newSamasaCombinationLengthMinusOne"].new[0] = byte(len(ri.samasaGateSequence) - 1)

		rom.codeMutables["makuSignText"].new[3] = 0x30 + byte(ri.requiredEssences)
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

	rom.setShopPrices(ri.shopPrices)
	rom.setSeedData()
	rom.setFileSelectText(ri.archipelagoSlotName)
	rom.attachText()

	// regenerate collect mode table to accommodate changes based on contents.
	rom.codeMutables["collectPropertiesTable"].new = []byte(makeCollectPropertiesTable(rom.game, rom.player, rom.itemSlots))

	if rom.game == GAME_SEASONS {
		// annoying special case to prevent text on key drop
		mut := rom.itemSlots["d7 armos puzzle"]
		if mut.treasure.id == rom.treasures["Small Key (Explorer's Crypt)"].id {
			rom.data[mut.subidAddrs[0].fullOffset()] = 0x01
		}
	} else {
		// other special case to prevent text on key drop
		mut := rom.itemSlots["d8 stalfos"]
		if mut.treasure.id == rom.treasures["Small Key (Sword & Shield Dungeon)"].id {
			rom.data[mut.subidAddrs[0].fullOffset()] = 0x00
		}
	}

	for _, k := range orderedKeys(rom.itemSlots) {
		rom.itemSlots[k].mutate(rom.data)
	}
	for _, k := range orderedKeys(rom.codeMutables) {
		rom.codeMutables[k].mutate(rom.data)
	}
	for _, k := range orderedKeys(rom.treasures) {
		treasure := rom.treasures[k]
		if treasure.id != 0x2d {
			// Don't mutate rings, as they are handled in asm/rings.yaml
			rom.treasures[k].mutate(rom.data)
		}
	}

	if ri.goal == "beat_ganon" {
		// Re-cable state 0 of DIN_DESCENDING_CRYSTAL to middle of state27
		// to directly get warped to the Room of Rites
		copy(rom.data[0xd539:], []byte{0x06 + 0x19, 0x5a})
	}

	rom.setCompassData()
	//	rom.setLinkedData()

	sum := makeRomChecksum(rom.data)
	rom.data[0x14e] = sum[0]
	rom.data[0x14f] = sum[1]

	outSum := sha1.Sum(rom.data)
	return outSum[:], nil
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
		setTreeNybble(rom.codeMutables["symmetryCityTreeSubId"], rom.itemSlots["symmetry city tree"])
		setTreeNybble(rom.codeMutables["southLynnaPresentTreeSubId"], rom.itemSlots["south lynna tree"])
		setTreeNybble(rom.codeMutables["crescentIslandTreeSubId"], rom.itemSlots["crescent island tree"])
		setTreeNybble(rom.codeMutables["zoraVillagePresentTreeSubId"], rom.itemSlots["zora village tree"])
		setTreeNybble(rom.codeMutables["rollingRidgeWestTreeSubId"], rom.itemSlots["rolling ridge west tree"])
		setTreeNybble(rom.codeMutables["ambisPalaceTreeSubId"], rom.itemSlots["ambi's palace tree"])
		setTreeNybble(rom.codeMutables["rollingRidgeEastTreeSubId"], rom.itemSlots["rolling ridge east tree"])
		setTreeNybble(rom.codeMutables["southLynnaPastTreeSubId"], rom.itemSlots["south lynna tree"])
		setTreeNybble(rom.codeMutables["dekuForestTreeSubId"], rom.itemSlots["deku forest tree"])
		setTreeNybble(rom.codeMutables["zoraVillagePastTreeSubId"], rom.itemSlots["zora village tree"])

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
	s = strings.ReplaceAll(s, ".", "")
	s = strings.ReplaceAll(s, ",", "")
	s = strings.ReplaceAll(s, "-", " ")
	return fmt.Sprintf("%c%s", s[0], strings.ReplaceAll(
		strings.Title(strings.ReplaceAll(s, "'", "")), " ", "")[1:])
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
			rom.codeMutables[label].new[0] = slot.mapTile
		}
	}
}

// set dungeon properties so that the compass beeps in the rooms actually
// containing small keys and boss keys.
func (rom *romState) setCompassData() {
	dungeonCodes := sora(rom.game, DUNGEON_CODES[GAME_SEASONS], DUNGEON_CODES[GAME_AGES]).([]string)
	dungeonNames := sora(rom.game, DUNGEON_NAMES[GAME_SEASONS], DUNGEON_NAMES[GAME_AGES]).([]string)

	// clear key flags
	for _, code := range dungeonCodes {
		for name, slot := range rom.itemSlots {
			if strings.HasPrefix(name, code+" ") {
				offset := getDungeonPropertiesAddr(rom.game, slot.group, slot.room).fullOffset()
				rom.data[offset] = rom.data[offset] & 0xed // reset bit 4
			}
		}
	}

	// set key flags
	for _, dungeonName := range dungeonNames {
		slots := rom.lookupAllItemSlots(fmt.Sprintf("Small Key (%s)", dungeonName))

		// boss keys can be absent in plando, so handle the nil case
		switch dungeonName {
		case "Hero's Cave", "d6 present":
			// do nothing
		case "d6 past":
			if slot := rom.lookupItemSlot("Boss Key (Ancient Ruins)"); slot != nil {
				slots = append(slots, slot)
			}
		default:
			keyName := fmt.Sprintf("Boss Key (%s)", dungeonName)
			if slot := rom.lookupItemSlot(keyName); slot != nil {
				slots = append(slots, slot)
			}
		}

		for _, slot := range slots {
			offset := getDungeonPropertiesAddr(rom.game, slot.group, slot.room).fullOffset()
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

func (rom *romState) setShopPrices(shopPrices map[string]int) {
	for shopSlotName, price := range shopPrices {
		priceByte := RUPEE_VALUES[price]
		mutableName := inflictCamelCase(shopSlotName + " price")
		rom.codeMutables[mutableName].new = []byte{priceByte}
	}
}

// set data to make linked playthroughs isomorphic to unlinked ones.
/*
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
*/
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
