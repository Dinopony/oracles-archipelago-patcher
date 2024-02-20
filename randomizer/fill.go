package randomizer

import (
	"container/list"
	"math/rand"
	"strings"
)

// give up completely if routing fails too many times
const maxTries = 200

// names of portals from the subrosia side.
var subrosianPortalNames = map[string]string{
	"eastern suburbs":      "volcanoes east",
	"spool swamp":          "subrosia market",
	"mt. cucco":            "strange brothers",
	"eyeglass lake":        "great furnace",
	"horon village":        "house of pirates",
	"temple remains lower": "volcanoes west",
	"temple remains upper": "d8 entrance",
}

var dungeonNames = map[int][]string{
	gameSeasons: []string{
		"d0", "d1", "d2", "d3", "d4", "d5", "d6", "d7", "d8"},
	gameAges: []string{
		"d1", "d2", "d3", "d4", "d5", "d6 present", "d6 past", "d7", "d8"},
}

type routeInfo struct {
	seed         uint32
	seasons      map[string]byte
	entrances    map[string]string
	portals      map[string]string
	companion    int // 1 to 3
	usedItems    *list.List
	usedSlots    *list.List
	ringMap      map[string]string
	attemptCount int
	src          *rand.Rand
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

var (
	seasonsById = []string{"spring", "summer", "autumn", "winter"}
	seasonAreas = []string{
		"north horon", "eastern suburbs", "woods of winter", "spool swamp",
		"holodrum plain", "sunken city", "lost woods", "tarm ruins",
		"western coast", "temple remains",
	}
)

var seedNames = []string{
	"Ember Seeds", 
	"Scent Seeds", 
	"Pegasus Seeds", 
	"Gale Seeds", 
	"Mystery Seeds",
}

var seedTreeNames = map[string]bool{
	"horon village tree":      true,
	"woods of winter tree":    true,
	"north horon tree":        true,
	"spool swamp tree":        true,
	"sunken city tree":        true,
	"tarm ruins tree":         true,
	"south lynna tree":        true,
	"deku forest tree":        true,
	"crescent island tree":    true,
	"symmetry city tree":      true,
	"rolling ridge west tree": true,
	"rolling ridge east tree": true,
	"ambi's palace tree":      true,
	"zora village tree":       true,
}

var rupeeValues = map[string]int{
	"Rupees (1)":   1,
	"Rupees (5)":   5,
	"Rupees (10)":  10,
	"Rupees (20)":  20,
	"Rupees (30)":  30,
	"Rupees (50)":  50,
	"Rupees (100)": 100,
	"Rupees (200)": 200,

	"goron mountain old man":      300,
	"western coast old man":       300,
	"holodrum plain east old man": 200,
	"horon village old man":       100,
	"north horon old man":         100,

	// rng is involved: each rupee is worth 1, 5, 10, or 20.
	// these totals are about 2 standard deviations below mean.
	"d2 rupee room": 150,
	"d6 rupee room": 90,
}

// checks whether the item fits in the slot due to things like seeds only going
// in trees, certain item slots not accomodating sub IDs. this doesn't check
// for softlocks or the availability of the slot and item.
func itemFitsInSlot(item string, slot string) bool {
	// dummy shop slots 1 and 2 can only hold their vanilla items.
	switch {
	case slot == "shop, 20 rupees" && item != "Bombs (10)":
		return false
	case slot == "shop, 30 rupees" && item != "Wooden Shield":
		return false
	case slot != "shop, 30 rupees" && item == "Wooden Shield":
		return false
	}

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

	// dungeons can only hold their respective dungeon-specific items. the
	// HasPrefix is specifically for ages d6 boss key.
	dungeonName := getDungeonName(item)
	if dungeonName != "" &&
		!strings.HasPrefix(getDungeonName(slot), dungeonName) {
		return false
	}

	// and only seeds can be slotted in seed trees, of course
	switch item {
	case "Ember Seeds", "Mystery Seeds", "Scent Seeds",
		"Pegasus Seeds", "Gale Seeds":
		return seedTreeNames[slot]
	default:
		return !seedTreeNames[slot]
	}
}

// return the name of a dungeon associated with a given item or slot name. ages
// d6 boss key returns "d6". non-dungeon names return "".
func getDungeonName(name string) string {
	if strings.HasPrefix(name, "d6 present") {
		return "d6 present"
	} else if strings.HasPrefix(name, "d6 past") {
		return "d6 past"
	} else if strings.HasPrefix(name, "maku path") {
		return "d0"
	} else if name == "slate" {
		return "d8"
	}

	switch name[:2] {
	case "d0", "d1", "d2", "d3", "d4", "d5", "d6", "d7", "d8":
		return name[:2]
	default:
		return ""
	}
}
