package randomizer

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"gopkg.in/yaml.v2"
)

// loaded from yaml, then converted to asm.
type asmData struct {
	filename string
	Common   yaml.MapSlice
	Floating yaml.MapSlice
	Seasons  yaml.MapSlice
	Ages     yaml.MapSlice
}

// designates a position at which the translated asm will overwrite whatever
// else is there, and associates it with a given label (or a generated label if
// the given one is blank). if the replacement extends beyond the end of the
// bank, the EOB point is moved to the end of the replacement. if the bank
// offset of `addr` is zero, the replacement will start at the existing EOB
// point.
//
// returns the final label of the replacement.
func (rom *romState) replaceAsm(addr address, label, asm string) string {
	if data, err := rom.assembler.compile(asm); err == nil {
		return rom.replaceRaw(addr, label, data)
	} else {
		panic(fmt.Sprintf("assembler error in %s:\n%v\n", label, err))
	}
}

// as replaceAsm, but interprets the data as a literal byte string.
func (rom *romState) replaceRaw(addr address, label, data string) string {
	if addr.offset == 0 {
		addr.offset = rom.bankEnds[addr.bank]
	}

	if label == "" {
		label = fmt.Sprintf("replacement at %02x:%04x", addr.bank, addr.offset)
	} else if strings.HasPrefix(label, "dma_") && addr.offset%0x10 != 0 {
		addr.offset += 0x10 - (addr.offset % 0x10) // align to $xxx0
	}

	end := addr.offset + uint16(len(data))
	if end > rom.bankEnds[addr.bank] {
		if end > 0x8000 || (end > 0x4000 && addr.bank == 0) {
			panic(fmt.Sprintf("not enough space for %s in bank %02x",
				label, addr.bank))
		}
		rom.bankEnds[addr.bank] = end
	}

	rom.codeMutables[label] = &mutableRange{
		addr: addr,
		new:  []byte(data),
	}
	rom.assembler.defineWord(label, addr.offset)

	return label
}

// returns a byte table of (group, room, collect mode, player) entries for
// randomized items. a mode >7f means to use &7f as an index to a jump table
// for special cases.
func makeCollectPropertiesTable(game, player int, itemSlots map[string]*itemSlot) string {
	b := new(strings.Builder)
	for _, key := range orderedKeys(itemSlots) {
		slot := itemSlots[key]

		// use no pickup animation for falling small keys
		mode := slot.collectMode
		if mode == collectModes["drop"] && slot.treasure != nil && slot.treasure.id == 0x30 {
			mode &= 0xf8
		}

		if _, err := b.Write([]byte{slot.group, slot.room, mode}); err != nil {
			panic(err)
		}
		for _, groupRoom := range slot.moreRooms {
			group, room := byte(groupRoom>>8), byte(groupRoom)
			if _, err := b.Write([]byte{group, room, mode}); err != nil {
				panic(err)
			}
		}
	}

	if game == GAME_SEASONS {
		// D6 fake rupee
		if _, err := b.Write([]byte{0x04, 0xc5, collectModes["fakePoof"]}); err != nil {
			panic(err)
		}
		// Maku tree gate opening cutscene
		if _, err := b.Write([]byte{0x00, 0xd9, collectModes["spinslash"]}); err != nil {
			panic(err)
		}
	}

	b.Write([]byte{0xff})
	return b.String()
}

func makeCompassRoomsTable(itemSlots map[string]*itemSlot) string {
	b := new(strings.Builder)
	for _, key := range orderedKeys(itemSlots) {
		slot := itemSlots[key]

		dungeon := byte(0xFF)
		if slot.treasure.id == 0x30 {
			dungeon = byte(slot.treasure.subid)
		} else if slot.treasure.id == 0x31 {
			dungeon = byte(slot.treasure.subid) + 1
		}

		if dungeon != 0xFF {
			if _, err := b.Write([]byte{slot.group, slot.room, dungeon}); err != nil {
				panic(err)
			}

			for _, groupRoom := range slot.moreRooms {
				group, room := byte(groupRoom>>8), byte(groupRoom)
				if _, err := b.Write([]byte{group, room, dungeon}); err != nil {
					panic(err)
				}
			}
		}
	}

	b.Write([]byte{0xff})
	return b.String()
}

func makeSamasaCombinationTable(samasaCombination []int) string {
	b := new(strings.Builder)
	for _, val := range samasaCombination {
		b.Write([]byte{byte(val)})
	}
	return b.String()
}

func makeSamasaGateSequenceScript(samasaSequence []int) ([]byte, error) {
	if len(samasaSequence) == 0 {
		return []byte{}, nil
	}

	const DELAY_6 = 0xf6
	const CALL_SCRIPT = 0xc0
	const MOVE_DOWN = 0xee
	const MOVE_LEFT = 0xef
	const MOVE_RIGHT = 0xed
	const WRITE_OBJECT_BYTE = 0x8e
	const SHOW_TEXT_LOW_INDEX = 0x98
	const ENABLE_ALL_OBJECTS = 0xb9

	bytes := make([]byte, 0, len(samasaSequence)*10)
	currentPosition := 1

	// Add a fake last press on button 1 to make the pirate go back to its
	// original position
	samasaSequence = append(samasaSequence, 1)

	for _, buttonToPress := range samasaSequence {
		// If current button is at a different position than the current one,
		// make the pirate move
		if buttonToPress != currentPosition {
			if buttonToPress < currentPosition {
				distanceToMove := 0x8*(currentPosition-buttonToPress) + 1
				bytes = append(bytes, MOVE_LEFT, byte(distanceToMove))
			} else {
				distanceToMove := 0x8*(buttonToPress-currentPosition) + 1
				bytes = append(bytes, MOVE_RIGHT, byte(distanceToMove))
			}

			currentPosition = buttonToPress
		}

		// Close the cupboard to express a button press on the gate by calling
		// the "closeOpenCupboard" subscript
		bytes = append(bytes, CALL_SCRIPT, 0x59, 0x5e)
	}

	// Remove the "cupboard press" from last entry, since it was only meant
	// to make the pirate go back to its starting position
	bytes = bytes[:len(bytes)-3]

	// Add some termination to the script
	bytes = append(bytes, MOVE_DOWN, 0x15)
	bytes = append(bytes, WRITE_OBJECT_BYTE, 0x7c, 0x00)
	bytes = append(bytes, DELAY_6)
	bytes = append(bytes, SHOW_TEXT_LOW_INDEX, 0x0d)
	bytes = append(bytes, ENABLE_ALL_OBJECTS)
	bytes = append(bytes, 0x5e, 0x4b) // jump back to script start

	if len(bytes) >= 0xF6 {
		return nil, fmt.Errorf("samasa gate sequence is too long")
	}

	return bytes, nil
}

// that's correct
type eobThing struct {
	addr         address
	label, thing string
}

// applies the labels and EOB declarations in the asm data sets.
// returns a slice of added labels.
func (rom *romState) applyAsmData(asmFiles []*asmData) []string {
	// preprocess map slices (keys = labels, values = asm blocks)
	slices := make([]yaml.MapSlice, 0)
	for _, asmFile := range asmFiles {
		if rom.game == GAME_SEASONS {
			slices = append(slices, asmFile.Common, asmFile.Seasons)
		} else {
			slices = append(slices, asmFile.Common, asmFile.Ages)
		}
	}

	// include free code
	freeCode := make(map[string]string)
	for _, asmFile := range asmFiles {
		for _, item := range asmFile.Floating {
			k, v := item.Key.(string), item.Value.(string)
			freeCode[k] = v
		}
	}
	for _, slice := range slices {
		for name, item := range slice {
			v := item.Value.(string)
			if strings.HasPrefix(v, "/include") {
				funcName := strings.Split(v, " ")[1]
				slice[name].Value = freeCode[funcName]
			}
		}
	}

	// save original EOB boundaries
	originalBankEnds := make([]uint16, 0x40)
	copy(originalBankEnds, rom.bankEnds)

	// make placeholders for labels and accumulate EOB items
	allEobThings := make([]eobThing, 0, 3000) // 3000 is probably fine
	for _, slice := range slices {
		for _, item := range slice {
			k, v := item.Key.(string), item.Value.(string)
			addr, label := parseMetalabel(k)
			if label != "" {
				rom.assembler.defineWord(label, 0)
			}
			if addr.offset == 0 {
				allEobThings = append(allEobThings,
					eobThing{address{addr.bank, 0}, label, v})
			}
		}
	}

	// defines (which have no labels, by convention) must be processed first
	sort.Slice(allEobThings, func(i, j int) bool {
		return allEobThings[i].label == ""
	})
	// owl text must go last
	for i, thing := range allEobThings {
		if thing.label == "owlText" {
			allEobThings = append(allEobThings[:i], allEobThings[i+1:]...)
			allEobThings = append(allEobThings, thing)
			break
		}
	}

	// write EOB asm using placeholders for labels, in order to get real addrs
	for _, thing := range allEobThings {
		rom.replaceAsm(thing.addr, thing.label, thing.thing)
	}

	// also get labels for labeled replacements
	for _, slice := range slices {
		for _, item := range slice {
			addr, label := parseMetalabel(item.Key.(string))
			if addr.offset != 0 && label != "" {
				rom.assembler.defineWord(label, addr.offset)
			}
		}
	}

	// reset EOB boundaries
	copy(rom.bankEnds, originalBankEnds)

	labels := make([]string, 0, 3000) // 3000 probably still fine

	// rewrite EOB asm, using real addresses for labels
	for _, thing := range allEobThings {
		labels = append(labels,
			rom.replaceAsm(thing.addr, thing.label, thing.thing))
	}

	// make non-EOB asm replacements
	for _, slice := range slices {
		for _, item := range slice {
			k, v := item.Key.(string), item.Value.(string)
			if addr, label := parseMetalabel(k); addr.offset != 0 {
				labels = append(labels, rom.replaceAsm(addr, label, v))
			}
		}
	}

	return labels
}

// applies the labels and EOB declarations in the given asm data files.
func (rom *romState) applyAsmFiles(ri *routeInfo) {
	asmPaths := []string{
		"defines",
		"util",
		"vars",
		"animals",
		"bossitems",
		"collect",
		"cutscenes",
		"font",
		"gfx",
		"itemevents",
		"keysanity",
		"layouts",
		"linked",
		"misc",
		"multi",
		"newgame",
		"options",
		"progressives",
		"rings",
		"static_items",
		"text",
		"trade_items",
		"triggers",
		"warp_to_start",
		"seasons/compass_chimes",
		"seasons/get_item_behavior",
		"seasons/impa_refill",
		"seasons/maku_tree",
		"seasons/samasa_combination",
		"seasons/seasons_handling",
		"seasons/shops_handling",
		"seasons/specific_checks",
		"seasons/subscreen_1_improvement",
		"seasons/tarm_gate_requirement",
		"seasons/combat_difficulty",
	}

	if ri.warpToStart {
		asmPaths = append(asmPaths, "seasons/warp_to_start")
	}
	if ri.quickFlute {
		asmPaths = append(asmPaths, "seasons/quick_flute")
	}
	if ri.turnOldMenIntoLocations {
		asmPaths = append(asmPaths, "seasons/old_men_as_locations")
	}
	if ri.removeD0AltEntrance {
		asmPaths = append(asmPaths, "seasons/remove_d0_alt_entrance")
	}
	if ri.removeD2AltEntrance {
		asmPaths = append(asmPaths, "seasons/remove_d2_alt_entrance")
	}

	if ri.masterSmallKeys {
		rom.data[0x18357] = byte(0)
		// Change obtention text
		rom.data[0x7546F] = byte(' ')
		rom.data[0x75470] = byte(0x02)
		rom.data[0x75471] = byte(0xe5)
		rom.data[0x75472] = byte(' ')
	}
	if ri.masterBossKeys {
		rom.data[0x1834F] = byte(0)
		rom.data[0x18350] = byte(0)
	}

	exe, err := os.Executable()
	if err != nil {
		return
	}
	dirName := filepath.Dir(exe)

	asmFiles := make([]*asmData, len(asmPaths))

	asmBaseDir := filepath.Join(dirName, "asm")
	for i, path := range asmPaths {
		fullPath := filepath.Join(asmBaseDir, path) + ".yaml"

		asmFiles[i] = new(asmData)
		asmFiles[i].filename = fullPath

		b, err := os.ReadFile(fullPath)
		if err != nil {
			panic(err)
		}

		if err := yaml.Unmarshal(b, asmFiles[i]); err != nil {
			fmt.Println("Error while unmarshaling ", asmFiles[i])
			panic(err)
		}
	}

	rom.applyAsmData(asmFiles)
}

// returns the address and label components of a meta-label such as
// "02/openRingList" or "02/56a1/". see asm/README.md for details.
func parseMetalabel(ml string) (addr address, label string) {
	switch tokens := strings.Split(ml, "/"); len(tokens) {
	case 1:
		fmt.Sscanf(ml, "%s", &label)
	case 2:
		fmt.Sscanf(ml, "%x/%s", &addr.bank, &label)
	case 3:
		fmt.Sscanf(ml, "%x/%x/%s", &addr.bank, &addr.offset, &label)
	default:
		panic("invalid metalabel: " + ml)
	}
	return
}

// returns a $40-entry slice of addresses of the ends of rom banks for the
// given game.
func loadBankEnds(game string) []uint16 {
	eobs := make(map[string][]uint16)
	if err := ReadEmbeddedYaml("romdata/eob.yaml", eobs); err != nil {
		panic(err)
	}
	return eobs[game]
}

// loads text, processes it, and attaches it to matching labels.
func (rom *romState) attachText() {
	// load initial text
	textMap := make(map[string]map[string]string)
	if err := ReadEmbeddedYaml("romdata/text.yaml", textMap); err != nil {
		panic(err)
	}
	for label, rawText := range textMap[gameNames[rom.game]] {
		if mut, ok := rom.codeMutables[label]; ok {
			mut.new = processText(rawText)
		} else {
			panic(fmt.Sprintf("no code label matches text label %q", label))
		}
	}

	// insert randomized item names into shop text
	shopNames := loadShopNames(gameNames[rom.game])
	shopMap := map[string]string{}
	if rom.game == GAME_SEASONS {
		shopMap["horonShop1Text"] = "shop, 20 rupees"
		shopMap["horonShop2Text"] = "shop, 30 rupees"
		shopMap["horonShop3Text"] = "shop, 150 rupees"
		shopMap["advanceShop1Text"] = "advance shop 1"
		shopMap["advanceShop2Text"] = "advance shop 2"
		shopMap["advanceShop3Text"] = "advance shop 3"
		shopMap["syrupShop1Text"] = "syrup shop 1"
		shopMap["syrupShop2Text"] = "syrup shop 2"
		shopMap["syrupShop3Text"] = "syrup shop 3"
		shopMap["membersShopSatchelText"] = "member's shop 1"
		shopMap["membersShopGashaText"] = "member's shop 2"
		shopMap["membersShopMapText"] = "member's shop 3"
		shopMap["marketItem1Text"] = "subrosia market, 1st item"
		shopMap["marketItem2Text"] = "subrosia market, 2nd item"
		shopMap["marketItem3Text"] = "subrosia market, 3rd item"
		shopMap["marketItem4Text"] = "subrosia market, 4th item"
		shopMap["marketItem5Text"] = "subrosia market, 5th item"
	}
	for codeName, slotName := range shopMap {
		code := rom.codeMutables[codeName]
		itemName := shopNames[rom.itemSlots[slotName].treasure.displayName]
		code.new = append(code.new[:2],
			append([]byte(itemName), code.new[2:]...)...)
	}
}

var hashCommentRegexp = regexp.MustCompile(" #.+?\n")

// processes a raw text string as a go string literal, converting escape
// sequences to their actual values. "comments" and literal newlines are
// stripped.
func processText(s string) []byte {
	var err error
	s = hashCommentRegexp.ReplaceAllString(s, "")
	s, err = strconv.Unquote("\"" + s + "\"")
	if err != nil {
		panic(err)
	}
	return []byte(s)
}

var articleRegexp = regexp.MustCompile("^(an?|the|some) ")

// return a map of internal item names to text that should be displayed for the
// item in shops.
func loadShopNames(game string) map[string]string {
	m := make(map[string]string)

	// load names used for owl hints
	itemFiles := []string{
		"hints/common_items.yaml",
		fmt.Sprintf("hints/%s_items.yaml", game),
	}
	for _, filename := range itemFiles {
		if err := ReadEmbeddedYaml(filename, m); err != nil {
			panic(err)
		}
	}

	// remove articles
	for k, v := range m {
		m[k] = articleRegexp.ReplaceAllString(v, "")
	}

	return m
}
