package randomizer

import (
    "container/list"
    "fmt"
    "os"
    "strings"
    "strconv"
    "gopkg.in/yaml.v2"
)

type inputData struct {
    items                map[string]string
    dungeons             map[string]string
    portals              map[string]string
    seasons              map[string]string
    hints                map[string]string
    settings             map[string]string
    oldManRupeeValues    map[string]string
}

func newInputData() *inputData {
    return &inputData{
        items:                make(map[string]string),
        dungeons:             make(map[string]string),
        portals:              make(map[string]string),
        seasons:              make(map[string]string),
        hints:                make(map[string]string),
        settings:             make(map[string]string),
        oldManRupeeValues:    make(map[string]string),
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
    section := data.items
    for name, contents := range rawData {
        switch(name) {
            case "locations":            section = data.items
            case "dungeon entrances":    section = data.dungeons
            case "subrosia portals":     section = data.portals
            case "default seasons":      section = data.seasons
            case "hints":                section = data.hints
            case "settings":             section = data.settings
            case "old man rupee values": section = data.oldManRupeeValues
            default: return nil, fmt.Errorf("unknown section: %q", name)
        }

        for k,v := range contents {
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
    game                int

	companion           int
    warpToStart         bool
    heartBeepInterval   int
    requiredEssences    int
    goldenBeastsRequirement int
    treehouseOldManRequirement int
	archipelagoSlotName string
    
	entrances           map[string]string
	usedItems           *list.List
	usedSlots           *list.List
    oldManRupeeValues   map[string]int

    // Seasons-specific
    seasons             map[string]byte
    portals             map[string]string
    foolsOreDamage      int
    pedestalSequence    [8]byte
}

func processSeasonsSpecificSettings(data *inputData, ri *routeInfo) (error) {
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

    // Set Fool's Ore damage if specified
    ri.foolsOreDamage = 12
    if str, ok := data.settings["fools_ore_damage"]; ok {
        ri.foolsOreDamage, err = strconv.Atoi(str); 
        if err != nil {
            return fmt.Errorf("settings.fools_ore_damage is invalid (must be a number)")
        }
    }

    // Set Lost Woods item sequence
    if str, ok := data.settings["lost_woods_item_sequence"]; ok {
        lostWoodsItemSequence := strings.Split(str, " ")
        for i:=0 ; i<4 ; i++ {
            switch lostWoodsItemSequence[2*i] {
            case "spring":  ri.pedestalSequence[i*2] = SEASON_SPRING
            case "summer":  ri.pedestalSequence[i*2] = SEASON_SUMMER
            case "autumn":  ri.pedestalSequence[i*2] = SEASON_AUTUMN
            case "winter":  ri.pedestalSequence[i*2] = SEASON_WINTER
            }
    
            switch lostWoodsItemSequence[2*i+1] {
            case "up":      ri.pedestalSequence[i*2+1] = DIRECTION_UP
            case "right":   ri.pedestalSequence[i*2+1] = DIRECTION_RIGHT
            case "down":    ri.pedestalSequence[i*2+1] = DIRECTION_DOWN
            case "left":    ri.pedestalSequence[i*2+1] = DIRECTION_LEFT
            }
        }
    }

    return nil
}

func processSettings(data *inputData, ri *routeInfo) (error) {
    var err error

    // Companion deciding which Natzu region will be inside the seed
    ri.companion = COMPANION_UNDEFINED
    if val, ok := data.settings["companion"]; ok {
        switch val {
        case "Ricky":   ri.companion = COMPANION_RICKY
        case "Dimitri": ri.companion = COMPANION_DIMITRI
        case "Moosh":   ri.companion = COMPANION_MOOSH
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

    // Archipelago slot name
    if str, ok := data.settings["slot_name"]; ok {
        ri.archipelagoSlotName = str
    }

    // Set heart beep interval if specified
    ri.heartBeepInterval = HEART_BEEP_DEFAULT
    if str, ok := data.settings["heart_beep_interval"]; ok {
        switch str {
        case "half":     ri.heartBeepInterval = HEART_BEEP_HALF
        case "quarter":  ri.heartBeepInterval = HEART_BEEP_QUARTER
        case "disabled": ri.heartBeepInterval = HEART_BEEP_DISABLED
        }
    }

    // Set old man rupee values
    for key, val := range data.oldManRupeeValues {
        ri.oldManRupeeValues[key], err = strconv.Atoi(val)
        if err != nil {
            return err
        }

        absValue := ri.oldManRupeeValues[key]
        if absValue < 0 {
            absValue *= -1
        }
        if _, ok := RUPEE_VALUES[absValue]; !ok {
            return fmt.Errorf("Unsupported rupee value for old man")
        }
    }

    if ri.game == GAME_SEASONS {
        processSeasonsSpecificSettings(data, ri)
    }

    return nil
}

// like findRoute, but uses a specified configuration instead of a random one.
func makePlannedRoute(data *inputData) (*routeInfo, error) {
    ri := &routeInfo{
        game:              GAME_UNDEFINED,
        entrances:         make(map[string]string),
        usedItems:         list.New(),
        usedSlots:         list.New(),
        oldManRupeeValues: make(map[string]int),
    }

    if data.settings["game"] == "seasons" {
        ri.game = GAME_SEASONS
    } else if data.settings["game"] == "ages" {
        ri.game = GAME_AGES
    } else {
        return nil, fmt.Errorf("Invalid game")
    }

    err := processSettings(data, ri)
    if err != nil {
        return nil, err
    }

    // Locations
    for slot, item := range data.items {
        if !itemFitsInSlot(item, slot) {
            return nil, fmt.Errorf("%s doesn't fit in %s", item, slot)
        }

        ri.usedItems.PushBack(item)
        ri.usedSlots.PushBack(slot)
    }

    // Dungeon entrances
    for entrance, dungeon := range data.dungeons {
        entrance = strings.Replace(entrance, " entrance", "", 1)
        for _, s := range []string{entrance, dungeon} {
            if s == "d0" || getStringIndex(DUNGEON_CODES[ri.game], s) == -1 {
                return nil, fmt.Errorf("no such dungeon: %s", s)
            }
        }
        ri.entrances[entrance] = dungeon
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
