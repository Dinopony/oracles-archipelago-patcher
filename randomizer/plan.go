package randomizer

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"gopkg.in/yaml.v2"
)

type inputData struct {
	items             map[string]string
	dungeons          map[string]string
	portals           map[string]string
	seasons           map[string]string
	hints             map[string]string
	settings          map[string]string
	oldManRupeeValues map[string]string
	shopPrices        map[string]string
}

func newInputData() *inputData {
	return &inputData{
		items:             make(map[string]string),
		dungeons:          make(map[string]string),
		portals:           make(map[string]string),
		seasons:           make(map[string]string),
		hints:             make(map[string]string),
		settings:          make(map[string]string),
		oldManRupeeValues: make(map[string]string),
		shopPrices:        make(map[string]string),
	}
}

func parseYamlInput(path string) (*routeInfo, error) {
	fmt.Println("Processing input file " + path)
	yamlFile, err := os.ReadFile(path)
	if err != nil {
		fmt.Println("Could not read YAML file", err)
	}

	rawData := make(map[string]map[string]string)
	yaml.Unmarshal(yamlFile, rawData)

	data := newInputData()
	var section map[string]string
	for name, contents := range rawData {
		switch name {
		case "locations":
			section = data.items
		case "dungeon entrances":
			section = data.dungeons
		case "subrosia portals":
			section = data.portals
		case "default seasons":
			section = data.seasons
		case "hints":
			section = data.hints
		case "settings":
			section = data.settings
		case "old man rupee values":
			section = data.oldManRupeeValues
		case "shop prices":
			section = data.shopPrices
		default:
			return nil, fmt.Errorf("unknown section: %q", name)
		}

		for k, v := range contents {
			section[k] = v
		}
	}

	ri, err := makePlannedRoute(data)
	if err != nil {
		return nil, err
	}

	return ri, nil
}

type routeInfo struct {
	game    int
	version string
	goal    string

	companion                  int
	warpToStart                bool
	quickFlute                 bool
	openAdvanceShop            bool
	heartBeepInterval          int
	requiredEssences           int
	goldenBeastsRequirement    int
	treehouseOldManRequirement int
	tarmGateRequiredJewels     int
	signGuyRequirement         int
	revealGoldenOreTiles       bool
	turnOldMenIntoLocations    bool
	masterSmallKeys            bool
	masterBossKeys             bool

	characterSprite     string
	characterPalette    string
	archipelagoSlotName string

	entrances         map[string]string
	locationContents  map[string]string
	oldManRupeeValues map[string]int
	shopPrices        map[string]int

	// Seasons-specific
	seasons                map[string]byte
	portals                map[string]string
	foolsOreDamage         int
	receivedDamageModifier int
	pedestalSequence       [8]byte
	samasaGateSequence     []int
	removeD0AltEntrance    bool
	removeD2AltEntrance    bool
	renewableHoronShop3    bool
}

func processSeasonsSpecificSettings(data *inputData, ri *routeInfo) error {
	var err error

	// Set Maku Seed to be given at the specified amount of essences
	ri.requiredEssences = 8
	if str, ok := data.settings["required_essences"]; ok {
		ri.requiredEssences, err = strconv.Atoi(str)
		if err != nil {
			return fmt.Errorf("settings.required_essences is invalid (0 to 8)")
		}
	}

	// Set golden beasts requirement for golden old man check
	ri.goldenBeastsRequirement = 4
	if str, ok := data.settings["golden_beasts_requirement"]; ok {
		ri.goldenBeastsRequirement, err = strconv.Atoi(str)
		if err != nil || ri.goldenBeastsRequirement < 0 || ri.goldenBeastsRequirement > 4 {
			return fmt.Errorf("settings.golden_beasts_requirement is invalid (0 to 4)")
		}
	}

	ri.treehouseOldManRequirement = 5
	if str, ok := data.settings["treehouse_old_man_requirement"]; ok {
		ri.treehouseOldManRequirement, err = strconv.Atoi(str)
		if err != nil || ri.treehouseOldManRequirement < 0 || ri.treehouseOldManRequirement > 8 {
			return fmt.Errorf("settings.treehouse_old_man_requirement is invalid (0 to 8)")
		}
	}

	ri.tarmGateRequiredJewels = 4
	if str, ok := data.settings["tarm_gate_required_jewels"]; ok {
		ri.tarmGateRequiredJewels, err = strconv.Atoi(str)
		if err != nil || ri.tarmGateRequiredJewels < 0 || ri.tarmGateRequiredJewels > 4 {
			return fmt.Errorf("settings.tarm_gate_required_jewels is invalid (0 to 4)")
		}
	}

	ri.signGuyRequirement = 100
	if str, ok := data.settings["sign_guy_requirement"]; ok {
		ri.signGuyRequirement, err = strconv.Atoi(str)
		if err != nil || ri.signGuyRequirement < 0 || ri.signGuyRequirement > 250 {
			return fmt.Errorf("settings.sign_guy_requirement is invalid (0 to 250)")
		}
	}

	// Set Fool's Ore damage if specified
	ri.foolsOreDamage = 12
	if str, ok := data.settings["fools_ore_damage"]; ok {
		ri.foolsOreDamage, err = strconv.Atoi(str)
		if err != nil {
			return fmt.Errorf("settings.fools_ore_damage is invalid (must be a number)")
		}
	}

	ri.receivedDamageModifier = 0
	if str, ok := data.settings["received_damage_modifier"]; ok {
		ri.receivedDamageModifier, err = strconv.Atoi(str)
		if err != nil {
			return fmt.Errorf("settings.received_damage_modifier is invalid (must be a number)")
		}
	}

	// Change golden ore tiles in Subrosia to indicate they are digging spots
	ri.revealGoldenOreTiles = false
	if str, ok := data.settings["reveal_golden_ore_tiles"]; ok {
		ri.revealGoldenOreTiles = (str == "true")
	}

	ri.removeD0AltEntrance = false
	if str, ok := data.settings["remove_d0_alt_entrance"]; ok {
		ri.removeD0AltEntrance = (str == "true")
	}

	ri.removeD2AltEntrance = false
	if str, ok := data.settings["remove_d2_alt_entrance"]; ok {
		ri.removeD2AltEntrance = (str == "true")
	}

	ri.renewableHoronShop3 = false
	if str, ok := data.settings["renewable_horon_shop_3"]; ok {
		ri.renewableHoronShop3 = (str == "true")
	}

	// Set Lost Woods item sequence
	if str, ok := data.settings["lost_woods_item_sequence"]; ok {
		lostWoodsItemSequence := strings.Split(str, " ")
		for i := 0; i < 4; i++ {
			switch lostWoodsItemSequence[2*i] {
			case "spring":
				ri.pedestalSequence[i*2] = SEASON_SPRING
			case "summer":
				ri.pedestalSequence[i*2] = SEASON_SUMMER
			case "autumn":
				ri.pedestalSequence[i*2] = SEASON_AUTUMN
			case "winter":
				ri.pedestalSequence[i*2] = SEASON_WINTER
			}

			switch lostWoodsItemSequence[2*i+1] {
			case "up":
				ri.pedestalSequence[i*2+1] = DIRECTION_UP
			case "right":
				ri.pedestalSequence[i*2+1] = DIRECTION_RIGHT
			case "down":
				ri.pedestalSequence[i*2+1] = DIRECTION_DOWN
			case "left":
				ri.pedestalSequence[i*2+1] = DIRECTION_LEFT
			}
		}
	}

	// Set Samasa Gate button sequence
	ri.samasaGateSequence = []int{2, 2, 1, 0, 0, 3, 3, 3}
	if str, ok := data.settings["samasa_gate_sequence"]; ok {
		ri.samasaGateSequence = []int{}
		samasaGateSequence := strings.Split(str, " ")
		if len(samasaGateSequence) == 0 {
			return fmt.Errorf("samasa gate sequence is invalid")
		}
		for _, numStr := range samasaGateSequence {
			num, err := strconv.Atoi(numStr)
			if err != nil {
				return err
			}
			if num < 0 || num > 3 {
				return fmt.Errorf("samasa gate sequence is invalid")
			}
			ri.samasaGateSequence = append(ri.samasaGateSequence, num)
		}
	}

	return nil
}

func processSettings(data *inputData, ri *routeInfo) error {
	ri.goal = ""
	if val, ok := data.settings["goal"]; ok {
		ri.goal = val
	}
	if ri.goal != "beat_onox" && ri.goal != "beat_ganon" {
		return fmt.Errorf("settings.goal is invalid ('beat_onox' or 'beat_ganon')")
	}

	// Companion deciding which Natzu region will be inside the seed
	ri.companion = COMPANION_UNDEFINED
	if val, ok := data.settings["companion"]; ok {
		switch val {
		case "Ricky":
			ri.companion = COMPANION_RICKY
		case "Dimitri":
			ri.companion = COMPANION_DIMITRI
		case "Moosh":
			ri.companion = COMPANION_MOOSH
		}
	}
	if ri.companion == COMPANION_UNDEFINED {
		return fmt.Errorf("settings.companion is missing or invalid ('Ricky', 'Dimitri' or 'Moosh')")
	}

	// Warp to start enabled / disabled
	if val, ok := data.settings["warp_to_start"]; ok {
		ri.warpToStart = (val == "true")
	} else {
		return fmt.Errorf("settings.warp_to_start is missing ('true' or 'false')")
	}

	// Quick Flute enabled / disabled
	ri.quickFlute = false
	if str, ok := data.settings["quick_flute"]; ok {
		ri.quickFlute = (str == "true")
	}

	// Advance shop
	ri.openAdvanceShop = false
	if str, ok := data.settings["open_advance_shop"]; ok {
		ri.openAdvanceShop = (str == "true")
	}

	// Turn old men into locations
	ri.turnOldMenIntoLocations = false
	if str, ok := data.settings["turn_old_men_into_locations"]; ok {
		ri.turnOldMenIntoLocations = (str == "true")
	}

	// Archipelago slot name
	if str, ok := data.settings["slot_name"]; ok {
		ri.archipelagoSlotName = str
	}

	// Master Keys
	ri.masterSmallKeys = false
	ri.masterBossKeys = false
	if str, ok := data.settings["master_keys"]; ok {
		if str == "all_dungeon_keys" {
			ri.masterSmallKeys = true
			ri.masterBossKeys = true
		} else if str == "all_small_keys" {
			ri.masterSmallKeys = true
		}
	}

	// Set heart beep interval if specified
	ri.heartBeepInterval = HEART_BEEP_DEFAULT
	if str, ok := data.settings["heart_beep_interval"]; ok {
		switch str {
		case "half":
			ri.heartBeepInterval = HEART_BEEP_HALF
		case "quarter":
			ri.heartBeepInterval = HEART_BEEP_QUARTER
		case "disabled":
			ri.heartBeepInterval = HEART_BEEP_DISABLED
		}
	}

	ri.characterSprite = "link"
	if str, ok := data.settings["character_sprite"]; ok {
		ri.characterSprite = str
	}

	ri.characterPalette = "green"
	if str, ok := data.settings["character_palette"]; ok {
		ri.characterPalette = str
	}

	if ri.game == GAME_SEASONS {
		return processSeasonsSpecificSettings(data, ri)
	}

	return nil
}

// like findRoute, but uses a specified configuration instead of a random one.
func makePlannedRoute(data *inputData) (*routeInfo, error) {
	ri := &routeInfo{
		game:              GAME_UNDEFINED,
		entrances:         make(map[string]string),
		locationContents:  make(map[string]string),
		oldManRupeeValues: make(map[string]int),
		shopPrices:        make(map[string]int),
	}

	// Determine game
	if data.settings["game"] == "seasons" {
		ri.game = GAME_SEASONS
	} else if data.settings["game"] == "ages" {
		ri.game = GAME_AGES
	} else {
		return nil, fmt.Errorf("invalid game")
	}

	// Test apworld version compatibility
	var ok bool
	if ri.version, ok = data.settings["version"]; !ok {
		return nil, fmt.Errorf("invalid version")
	}
	if ri.version != VERSION {
		return nil, fmt.Errorf("invalid version (%s instead of %s)", ri.version, VERSION)
	}

	// Settings
	err := processSettings(data, ri)
	if err != nil {
		return nil, err
	}

	// Locations
	for slot, item := range data.items {
		if !itemFitsInSlot(item, slot) {
			return nil, fmt.Errorf("%s doesn't fit in %s", item, slot)
		}

		// Master Keys are fake items which are just Small Keys deep in their heart
		if strings.HasPrefix(item, "Master Key") {
			item = strings.ReplaceAll(item, "Master Key", "Small Key")
		}

		ri.locationContents[slot] = item
	}

	// Dungeon entrances
	for entrance, dungeon := range data.dungeons {
		entrance = strings.Replace(entrance, " entrance", "", 1)
		for _, s := range []string{entrance, dungeon} {
			if getStringIndex(DUNGEON_CODES[ri.game], s) == -1 {
				return nil, fmt.Errorf("no such dungeon: %s", s)
			}
		}
		ri.entrances[entrance] = dungeon
	}

	// Set old man rupee values
	for key, val := range data.oldManRupeeValues {
		ri.oldManRupeeValues[key], err = strconv.Atoi(val)
		if err != nil {
			return nil, err
		}

		absValue := ri.oldManRupeeValues[key]
		if absValue < 0 {
			absValue *= -1
		}
		if _, ok := RUPEE_VALUES[absValue]; !ok {
			return nil, fmt.Errorf("unsupported rupee value for old man")
		}
	}

	// Set shop prices
	for key, val := range data.shopPrices {
		ri.shopPrices[key], err = strconv.Atoi(val)
		if err != nil {
			return nil, err
		}

		if _, ok := RUPEE_VALUES[ri.shopPrices[key]]; !ok {
			return nil, fmt.Errorf("unsupported rupee value for shop price")
		}
	}

	if ri.game == GAME_SEASONS {
		// Default seasons
		ri.seasons = make(map[string]byte, len(data.seasons))
		for area, season := range data.seasons {
			id := getStringIndex(SEASONS_BY_ID, season)
			if id == -1 {
				return nil, fmt.Errorf("invalid default season: %s", season)
			}
			if getStringIndex(SEASON_AREAS, area) == -1 {
				return nil, fmt.Errorf("invalid season area: %s", area)
			}
			ri.seasons[area] = byte(id)
		}

		// Subrosia portals
		ri.portals = make(map[string]string, len(data.portals))
		for portal, connect := range data.portals {
			if _, ok := SUBROSIAN_PORTAL_NAMES[portal]; !ok {
				return nil, fmt.Errorf("invalid holodrum portal: %s", portal)
			}
			if _, ok := reverseLookup(SUBROSIAN_PORTAL_NAMES, connect); !ok {
				return nil, fmt.Errorf("invalid subrosia portal: %s", connect)
			}
			ri.portals[portal] = connect
		}
	}

	return ri, nil
}

// checks whether the item fits in the slot due to things like seeds only going
// in trees, certain item slots not accomodating sub IDs. this doesn't check
// for softlocks or the availability of the slot and item.
func itemFitsInSlot(item string, slot string) bool {
	// bomb flower has special graphics something. this could probably be
	// worked around like with the temple of seasons, but i'm not super
	// interested in doing that.
	if item == "bomb flower" {
		switch slot {
		case "cheval's test", "cheval's invention", "wild tokay game",
			"hidden tokay cave", "library present", "library past":
			return false
		}
	}

	// and only seeds can be slotted in seed trees, of course
	switch item {
	case "Ember Seeds", "Mystery Seeds", "Scent Seeds",
		"Pegasus Seeds", "Gale Seeds":
		return SEED_TREE_NAMES[slot]
	default:
		return !SEED_TREE_NAMES[slot]
	}
}

// overwrites regular owl hints with planned ones.
/*
func planOwlHints(p *plan, h *hinter, owlHints map[string]string) error {
    // sanity check first
    for owl, hint := range data.hints {
        hint = strings.Trim(hint, `"`)
        if _, ok := owlHints[owl]; !ok {
            return fmt.Errorf("no such owl: %s", owl)
        }
        if !isValidGameText(hint) {
            return fmt.Errorf("invalid hint text: %q", hint)
        }
    }

    // use hint if planned hint found, placeholder if not
    for owl := range owlHints {
        if hint, ok := data.hints[owl]; ok {
            owlHints[owl] = h.format(strings.Trim(hint, `"`))
        } else {
            owlHints[owl] = "..."
        }
    }

    return nil
}
*/

// returns the index of s in a, or -1 if not found.
func getStringIndex(a []string, s string) int {
	for i, v := range a {
		if v == s {
			return i
		}
	}
	return -1
}
