package randomizer

const VERSION = "6.0"
const INPUT_FILE_EXTENSION = ".patcherdata"

const (
	GAME_UNDEFINED = iota
	GAME_AGES      = 1
	GAME_SEASONS   = 2
)

var gameNames = map[int]string{
	GAME_UNDEFINED: "nil",
	GAME_AGES:      "ages",
	GAME_SEASONS:   "seasons",
}

const (
	COMPANION_UNDEFINED = 0
	COMPANION_RICKY     = 1
	COMPANION_DIMITRI   = 2
	COMPANION_MOOSH     = 3
)

const (
	HEART_BEEP_DISABLED = 0
	HEART_BEEP_DEFAULT  = 1
	HEART_BEEP_HALF     = 2
	HEART_BEEP_QUARTER  = 3
)

const (
	DIRECTION_UP    = 0
	DIRECTION_RIGHT = 1
	DIRECTION_DOWN  = 2
	DIRECTION_LEFT  = 3
)

const (
	SEASON_SPRING = 0
	SEASON_SUMMER = 1
	SEASON_AUTUMN = 2
	SEASON_WINTER = 3
)

var RUPEE_VALUES = map[int]byte{
	0:   0x00,
	1:   0x01,
	2:   0x02,
	5:   0x03,
	10:  0x04,
	20:  0x05,
	40:  0x06,
	30:  0x07,
	60:  0x08,
	70:  0x09,
	25:  0x0a,
	50:  0x0b,
	100: 0x0c,
	200: 0x0d,
	400: 0x0e,
	150: 0x0f,
	300: 0x10,
	500: 0x11,
	900: 0x12,
	80:  0x13,
	999: 0x14,
}

var SEED_TREE_NAMES = map[string]bool{
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

// names of portals from the subrosia side.
var SUBROSIAN_PORTAL_NAMES = map[string]string{
	"eastern suburbs":      "volcanoes east",
	"spool swamp":          "subrosia market",
	"mt. cucco":            "strange brothers",
	"eyeglass lake":        "great furnace",
	"horon village":        "house of pirates",
	"temple remains lower": "volcanoes west",
	"temple remains upper": "d8 entrance",
}

var DUNGEON_CODES = map[int][]string{
	GAME_SEASONS: {
		"d0", "d1", "d2", "d3", "d4", "d5", "d6", "d7", "d8"},
	GAME_AGES: {
		"d1", "d2", "d3", "d4", "d5", "d6 present", "d6 past", "d7", "d8"},
}

var DUNGEON_NAMES = map[int][]string{
	GAME_SEASONS: {
		"Hero's Cave",
		"Gnarled Root Dungeon",
		"Snake's Remains",
		"Poison Moth's Lair",
		"Dancing Dragon Dungeon",
		"Unicorn's Cave",
		"Ancient Ruins",
		"Explorer's Crypt",
		"Sword & Shield Dungeon"},
	GAME_AGES: {
		"d0",
		"d1",
		"d2",
		"d3",
		"d4",
		"d5",
		"d6 present",
		"d6 past",
		"d7",
		"d8"},
}

var SEASONS_BY_ID = []string{"spring", "summer", "autumn", "winter", "chaotic"}
var SEASON_AREAS = []string{
	"north horon", "eastern suburbs", "woods of winter", "spool swamp",
	"holodrum plain", "sunken city", "lost woods", "tarm ruins",
	"western coast", "temple remains", "horon village",
}

var PALETTE_BYTES = map[string]byte{
	"green":  0x00,
	"blue":   0x01,
	"red":    0x02,
	"orange": 0x03,
}
