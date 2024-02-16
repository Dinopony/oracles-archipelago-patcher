package randomizer

import (
	"container/list"
	"fmt"
	"math/rand"
	"reflect"
	"sort"
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

// adds nodes to the map based on default contents of item slots.
func addDefaultItemNodes(rom *romState, nodes map[string]*prenode) {
	for _, slot := range rom.itemSlots {
		tName, _ := reverseLookup(rom.treasures, slot.treasure)
		nodes[tName.(string)] = rootPrenode()
	}
}

func addNodes(prenodes map[string]*prenode, g graph) {
	for key, pn := range prenodes {
		switch pn.nType {
		case andNode, orNode, rupeesNode:
			g[key] = newNode(key, pn.nType)
		case countNode:
			g[key] = newNode(key, countNode)
			g[key].minCount = pn.minCount
		default:
			panic("unknown logic type for " + key)
		}
	}
}

func addNodeParents(prenodes map[string]*prenode, g graph) {
	for k, pn := range prenodes {
		if g[k] == nil {
			continue
		}
		for _, parent := range pn.parents {
			if g[parent.(string)] == nil {
				continue
			}
			g.addParents(map[string][]string{k: []string{parent.(string)}})
		}
	}
}

type routeInfo struct {
	graph        graph
	slots        map[string]*node
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

const (
	ricky   = 1
	dimitri = 2
	moosh   = 3
)

func newRouteGraph(rom *romState) graph {
	g := newGraph()
	totalPrenodes := getPrenodes(rom.game)
	addDefaultItemNodes(rom, totalPrenodes)
	addNodes(totalPrenodes, g)
	addNodeParents(totalPrenodes, g)
	return g
}

// getChecks converts a route info into a map of checks.
func getChecks(usedItems, usedSlots *list.List) map[*node]*node {
	checks := make(map[*node]*node)

	ei, es := usedItems.Front(), usedSlots.Front()
	for ei != nil {
		checks[es.Value.(*node)] = ei.Value.(*node)
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

// set the default seasons for all the applicable areas in the game, and return
// a mapping of area name to season value.
func rollSeasons(src *rand.Rand, g graph) map[string]byte {
	seasonMap := make(map[string]byte, len(seasonAreas))
	for _, area := range seasonAreas {
		id := src.Intn(len(seasonsById))
		season := seasonsById[id]
		g[fmt.Sprintf("%s default %s", area, season)].addParent(g["start"])
		seasonMap[area] = byte(id)
	}
	return seasonMap
}

var seedNames = []string{"Ember Seeds", "Scent Seeds",
	"Pegasus Seeds", "Gale Seeds", "Mystery Seeds"}

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

// checks whether the item fits in the slot due to things like seeds only going
// in trees, certain item slots not accomodating sub IDs. this doesn't check
// for softlocks or the availability of the slot and item.
func itemFitsInSlot(itemNode, slotNode *node) bool {
	// dummy shop slots 1 and 2 can only hold their vanilla items.
	switch {
	case slotNode.name == "shop, 20 rupees" && itemNode.name != "Bombs (10)":
		fallthrough
	case slotNode.name == "shop, 30 rupees" && itemNode.name != "Wooden Shield":
		fallthrough
	case itemNode.name == "Wooden Shield" && slotNode.name != "shop, 30 rupees":
		return false
	}

	// bomb flower has special graphics something. this could probably be
	// worked around like with the temple of seasons, but i'm not super
	// interested in doing that.
	if itemNode.name == "bomb flower" {
		switch slotNode.name {
		case "cheval's test", "cheval's invention", "wild tokay game",
			"hidden tokay cave", "library present", "library past":
			return false
		}
	}

	// dungeons can only hold their respective dungeon-specific items. the
	// HasPrefix is specifically for ages d6 boss key.
	dungeonName := getDungeonName(itemNode.name)
	if dungeonName != "" &&
		!strings.HasPrefix(getDungeonName(slotNode.name), dungeonName) {
		return false
	}

	// and only seeds can be slotted in seed trees, of course
	switch itemNode.name {
	case "Ember Seeds", "Mystery Seeds", "Scent Seeds",
		"Pegasus Seeds", "Gale Seeds":
		return seedTreeNames[slotNode.name]
	default:
		return !seedTreeNames[slotNode.name]
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

// returns true iff a is a slice and v is a value in that slice. panics if a is
// not a slice.
func sliceContains(a interface{}, v interface{}) bool {
	aValue := reflect.ValueOf(a)
	for i := 0; i < aValue.Len(); i++ {
		v2 := aValue.Index(i).Interface()
		if reflect.DeepEqual(v, v2) {
			return true
		}
	}
	return false
}

// return alphabetically sorted string values from a map.
func orderedValues(m map[string]string) []string {
	a, i := make([]string, len(m)), 0
	for _, v := range m {
		a[i] = v
		i++
	}
	sort.Strings(a)
	return a
}
